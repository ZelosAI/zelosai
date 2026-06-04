package controller

import (
	"context"
	"fmt"

	zelosv1alpha1 "github.com/ZelosAI/zelosai/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Condition types used across every Zelos CR status, following Kubernetes
// convention (see issue #34). `Available` is the substrate-level signal
// (workload exists and has its desired replicas Available); `Ready` is the
// rolled-up readiness consumed by automation.
const (
	ConditionAvailable = "Available"
	ConditionReady     = "Ready"
)

// Condition reasons. Reasons are machine-readable CamelCase tokens.
const (
	ReasonWorkloadAvailable   = "WorkloadAvailable"
	ReasonWorkloadUnavailable = "WorkloadUnavailable"
	ReasonWorkloadNotFound    = "WorkloadNotFound"
	ReasonAllComponentsReady  = "AllComponentsReady"
	ReasonComponentsNotReady  = "ComponentsNotReady"
	ReasonNoComponents        = "NoComponentsEnabled"
)

// Phase strings surfaced in ZelosPlatform.status.phase for human display.
const (
	PhasePending     = "Pending"
	PhaseProgressing = "Progressing"
	PhaseReady       = "Ready"
)

// workloadReadiness is the readiness verdict derived from a Deployment or
// StatefulSet's observed status.
type workloadReadiness struct {
	// Available is true when the workload reports at least one Available
	// replica and the Available count meets the desired replica count.
	Available bool
	// Found is false when the workload object does not (yet) exist.
	Found bool
	// Reason is a machine-readable CamelCase token for the condition.
	Reason string
	// Message is a human-readable explanation (e.g. "2/2 replicas available").
	Message string
}

// deploymentReadiness maps a Deployment's AvailableReplicas against its desired
// replica count into a readiness verdict.
func deploymentReadiness(ctx context.Context, c client.Client, key types.NamespacedName) workloadReadiness {
	var dep appsv1.Deployment
	if err := c.Get(ctx, key, &dep); err != nil {
		if apierrors.IsNotFound(err) {
			return workloadReadiness{
				Found:   false,
				Reason:  ReasonWorkloadNotFound,
				Message: fmt.Sprintf("Deployment %s not created yet", key.Name),
			}
		}
		return workloadReadiness{
			Found:   false,
			Reason:  ReasonWorkloadNotFound,
			Message: fmt.Sprintf("Deployment %s lookup failed: %v", key.Name, err),
		}
	}

	desired := int32(1)
	if dep.Spec.Replicas != nil {
		desired = *dep.Spec.Replicas
	}
	avail := dep.Status.AvailableReplicas
	if avail >= desired && desired > 0 {
		return workloadReadiness{
			Found:     true,
			Available: true,
			Reason:    ReasonWorkloadAvailable,
			Message:   fmt.Sprintf("%d/%d replicas available", avail, desired),
		}
	}
	return workloadReadiness{
		Found:     true,
		Available: false,
		Reason:    ReasonWorkloadUnavailable,
		Message:   fmt.Sprintf("%d/%d replicas available", avail, desired),
	}
}

// statefulSetReadiness maps a StatefulSet's ReadyReplicas against its desired
// replica count into a readiness verdict.
func statefulSetReadiness(ctx context.Context, c client.Client, key types.NamespacedName) workloadReadiness {
	var sts appsv1.StatefulSet
	if err := c.Get(ctx, key, &sts); err != nil {
		if apierrors.IsNotFound(err) {
			return workloadReadiness{
				Found:   false,
				Reason:  ReasonWorkloadNotFound,
				Message: fmt.Sprintf("StatefulSet %s not created yet", key.Name),
			}
		}
		return workloadReadiness{
			Found:   false,
			Reason:  ReasonWorkloadNotFound,
			Message: fmt.Sprintf("StatefulSet %s lookup failed: %v", key.Name, err),
		}
	}

	desired := int32(1)
	if sts.Spec.Replicas != nil {
		desired = *sts.Spec.Replicas
	}
	ready := sts.Status.ReadyReplicas
	if ready >= desired && desired > 0 {
		return workloadReadiness{
			Found:     true,
			Available: true,
			Reason:    ReasonWorkloadAvailable,
			Message:   fmt.Sprintf("%d/%d replicas ready", ready, desired),
		}
	}
	return workloadReadiness{
		Found:     true,
		Available: false,
		Reason:    ReasonWorkloadUnavailable,
		Message:   fmt.Sprintf("%d/%d replicas ready", ready, desired),
	}
}

// daemonSetReadiness maps a DaemonSet's NumberAvailable against its desired
// scheduled count into a readiness verdict. A DaemonSet with zero desired
// nodes (no nodes match the selector) is treated as available with a note,
// since there is nothing to schedule.
func daemonSetReadiness(ctx context.Context, c client.Client, key types.NamespacedName) workloadReadiness {
	var ds appsv1.DaemonSet
	if err := c.Get(ctx, key, &ds); err != nil {
		if apierrors.IsNotFound(err) {
			return workloadReadiness{
				Found:   false,
				Reason:  ReasonWorkloadNotFound,
				Message: fmt.Sprintf("DaemonSet %s not created yet", key.Name),
			}
		}
		return workloadReadiness{
			Found:   false,
			Reason:  ReasonWorkloadNotFound,
			Message: fmt.Sprintf("DaemonSet %s lookup failed: %v", key.Name, err),
		}
	}

	desired := ds.Status.DesiredNumberScheduled
	avail := ds.Status.NumberAvailable
	if desired == 0 {
		return workloadReadiness{
			Found:     true,
			Available: true,
			Reason:    ReasonWorkloadAvailable,
			Message:   "0 nodes match the DaemonSet selector",
		}
	}
	if avail >= desired {
		return workloadReadiness{
			Found:     true,
			Available: true,
			Reason:    ReasonWorkloadAvailable,
			Message:   fmt.Sprintf("%d/%d nodes available", avail, desired),
		}
	}
	return workloadReadiness{
		Found:     true,
		Available: false,
		Reason:    ReasonWorkloadUnavailable,
		Message:   fmt.Sprintf("%d/%d nodes available", avail, desired),
	}
}

// applyWorkloadConditions sets the standard Available + Ready conditions on a
// component CR's CommonStatus from a workloadReadiness verdict, using
// meta.SetStatusCondition so transition times are preserved. observedGeneration
// stamps each condition with the generation it was evaluated against. It
// returns the resulting Ready condition status as a string for convenience.
func applyWorkloadConditions(cs *zelosv1alpha1.CommonStatus, generation int64, wr workloadReadiness) metav1.ConditionStatus {
	availStatus := metav1.ConditionFalse
	if wr.Available {
		availStatus = metav1.ConditionTrue
	}
	meta.SetStatusCondition(&cs.Conditions, metav1.Condition{
		Type:               ConditionAvailable,
		Status:             availStatus,
		ObservedGeneration: generation,
		Reason:             wr.Reason,
		Message:            wr.Message,
	})

	// Ready mirrors Available for single-workload components; the umbrella
	// reconciler rolls per-component Ready up into the platform Ready.
	readyStatus := metav1.ConditionFalse
	readyReason := ReasonWorkloadUnavailable
	if wr.Available {
		readyStatus = metav1.ConditionTrue
		readyReason = ReasonWorkloadAvailable
	}
	meta.SetStatusCondition(&cs.Conditions, metav1.Condition{
		Type:               ConditionReady,
		Status:             readyStatus,
		ObservedGeneration: generation,
		Reason:             readyReason,
		Message:            wr.Message,
	})
	return readyStatus
}
