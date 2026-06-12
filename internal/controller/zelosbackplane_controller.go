package controller

import (
	"context"

	zelosv1alpha1 "github.com/ZelosAI/zelosai/api/v1alpha1"
	"github.com/ZelosAI/zelosai/internal/controller/render"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ZelosBackplaneReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=zelos.zelosai.io,resources=zelosbackplanes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=zelos.zelosai.io,resources=zelosbackplanes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete

func (r *ZelosBackplaneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var bp zelosv1alpha1.ZelosBackplane
	if err := r.Get(ctx, req.NamespacedName, &bp); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	substrate := bp.Spec.Substrate
	if substrate == "" {
		substrate = "nats"
	}

	// NATS: operator-installed StatefulSet + headless Service when no externalURL.
	operatorManaged := substrate == "nats" && bp.Spec.ExternalURL == ""
	var wr workloadReadiness
	if operatorManaged {
		sts := render.BuildNATSStatefulSet(&bp, bp.Spec)
		if err := applyObject(ctx, r.Client, r.Scheme, &bp, sts); err != nil {
			return ctrl.Result{}, err
		}
		svc := render.BuildNATSService(&bp)
		if err := applyObject(ctx, r.Client, r.Scheme, &bp, svc); err != nil {
			return ctrl.Result{}, err
		}
		bp.Status.URL = render.NATSURL(bp.Name, bp.Namespace)
		wr = statefulSetReadiness(ctx, r.Client, types.NamespacedName{Name: sts.Name, Namespace: sts.Namespace})
	} else {
		// Redis / Kafka / external NATS: never installed by the operator, so we
		// can't poll a workload — treat a configured substrate as Available.
		bp.Status.URL = bp.Spec.ExternalURL
		wr = workloadReadiness{
			Found:     true,
			Available: bp.Spec.ExternalURL != "",
			Reason:    ReasonWorkloadAvailable,
			Message:   "External substrate (not operator-managed)",
		}
		if bp.Spec.ExternalURL == "" {
			wr.Available = false
			wr.Reason = ReasonWorkloadUnavailable
			wr.Message = "External substrate selected but externalURL is unset"
		}
	}

	applyWorkloadConditions(&bp.Status.CommonStatus, bp.Generation, wr)
	bp.Status.ObservedGeneration = bp.Generation
	_ = r.Status().Update(ctx, &bp)
	if operatorManaged && !wr.Available {
		return ctrl.Result{RequeueAfter: componentRequeueInterval()}, nil
	}
	return ctrl.Result{}, nil
}

func (r *ZelosBackplaneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&zelosv1alpha1.ZelosBackplane{}).
		Complete(r)
}
