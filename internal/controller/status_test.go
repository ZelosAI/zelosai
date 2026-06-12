package controller

import (
	"context"
	"testing"

	zelosv1alpha1 "github.com/ZelosAI/zelosai/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func int32Ptr(i int32) *int32 { return &i }

func newDeployment(name string, replicas, available int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "zelos"},
		Spec:       appsv1.DeploymentSpec{Replicas: int32Ptr(replicas)},
		Status:     appsv1.DeploymentStatus{AvailableReplicas: available},
	}
}

func newStatefulSet(name string, replicas, ready int32) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "zelos"},
		Spec:       appsv1.StatefulSetSpec{Replicas: int32Ptr(replicas)},
		Status:     appsv1.StatefulSetStatus{ReadyReplicas: ready},
	}
}

func newDaemonSet(name string, desired, available int32) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "zelos"},
		Status: appsv1.DaemonSetStatus{
			DesiredNumberScheduled: desired,
			NumberAvailable:        available,
		},
	}
}

func TestDeploymentReadiness(t *testing.T) {
	cases := []struct {
		name          string
		objs          []client.Object
		key           types.NamespacedName
		wantAvailable bool
		wantFound     bool
		wantReason    string
	}{
		{
			name:          "all replicas available",
			objs:          []client.Object{newDeployment("zelosgateway-default", 2, 2)},
			key:           types.NamespacedName{Name: "zelosgateway-default", Namespace: "zelos"},
			wantAvailable: true, wantFound: true, wantReason: ReasonWorkloadAvailable,
		},
		{
			name:          "partial replicas available",
			objs:          []client.Object{newDeployment("zelosgateway-default", 3, 1)},
			key:           types.NamespacedName{Name: "zelosgateway-default", Namespace: "zelos"},
			wantAvailable: false, wantFound: true, wantReason: ReasonWorkloadUnavailable,
		},
		{
			name:          "zero replicas available",
			objs:          []client.Object{newDeployment("zelosgateway-default", 1, 0)},
			key:           types.NamespacedName{Name: "zelosgateway-default", Namespace: "zelos"},
			wantAvailable: false, wantFound: true, wantReason: ReasonWorkloadUnavailable,
		},
		{
			name:          "deployment not found",
			objs:          nil,
			key:           types.NamespacedName{Name: "missing", Namespace: "zelos"},
			wantAvailable: false, wantFound: false, wantReason: ReasonWorkloadNotFound,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(tc.objs...).Build()
			wr := deploymentReadiness(context.Background(), c, tc.key)
			if wr.Available != tc.wantAvailable {
				t.Errorf("Available = %v, want %v", wr.Available, tc.wantAvailable)
			}
			if wr.Found != tc.wantFound {
				t.Errorf("Found = %v, want %v", wr.Found, tc.wantFound)
			}
			if wr.Reason != tc.wantReason {
				t.Errorf("Reason = %q, want %q", wr.Reason, tc.wantReason)
			}
		})
	}
}

func TestStatefulSetReadiness(t *testing.T) {
	cases := []struct {
		name          string
		obj           client.Object
		wantAvailable bool
		wantReason    string
	}{
		{"ready", newStatefulSet("zelos-backplane-nats-default", 1, 1), true, ReasonWorkloadAvailable},
		{"not ready", newStatefulSet("zelos-backplane-nats-default", 1, 0), false, ReasonWorkloadUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(tc.obj).Build()
			wr := statefulSetReadiness(context.Background(), c, types.NamespacedName{Name: "zelos-backplane-nats-default", Namespace: "zelos"})
			if wr.Available != tc.wantAvailable {
				t.Errorf("Available = %v, want %v", wr.Available, tc.wantAvailable)
			}
			if wr.Reason != tc.wantReason {
				t.Errorf("Reason = %q, want %q", wr.Reason, tc.wantReason)
			}
		})
	}
}

func TestDaemonSetReadiness(t *testing.T) {
	cases := []struct {
		name          string
		obj           client.Object
		wantAvailable bool
	}{
		{"all nodes available", newDaemonSet("zelosclient-default", 3, 3), true},
		{"some nodes available", newDaemonSet("zelosclient-default", 3, 1), false},
		{"no matching nodes", newDaemonSet("zelosclient-default", 0, 0), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(tc.obj).Build()
			wr := daemonSetReadiness(context.Background(), c, types.NamespacedName{Name: "zelosclient-default", Namespace: "zelos"})
			if wr.Available != tc.wantAvailable {
				t.Errorf("Available = %v, want %v", wr.Available, tc.wantAvailable)
			}
		})
	}
}

func TestApplyWorkloadConditions(t *testing.T) {
	cases := []struct {
		name       string
		wr         workloadReadiness
		wantReady  metav1.ConditionStatus
		wantAvail  metav1.ConditionStatus
		wantReason string
	}{
		{
			name:       "available sets both true",
			wr:         workloadReadiness{Found: true, Available: true, Reason: ReasonWorkloadAvailable, Message: "2/2 replicas available"},
			wantReady:  metav1.ConditionTrue,
			wantAvail:  metav1.ConditionTrue,
			wantReason: ReasonWorkloadAvailable,
		},
		{
			name:       "unavailable sets both false",
			wr:         workloadReadiness{Found: true, Available: false, Reason: ReasonWorkloadUnavailable, Message: "0/2 replicas available"},
			wantReady:  metav1.ConditionFalse,
			wantAvail:  metav1.ConditionFalse,
			wantReason: ReasonWorkloadUnavailable,
		},
		{
			name:       "not found sets both false",
			wr:         workloadReadiness{Found: false, Available: false, Reason: ReasonWorkloadNotFound, Message: "Deployment x not created yet"},
			wantReady:  metav1.ConditionFalse,
			wantAvail:  metav1.ConditionFalse,
			wantReason: ReasonWorkloadNotFound,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cs := &zelosv1alpha1.CommonStatus{}
			got := applyWorkloadConditions(cs, 7, tc.wr)
			if got != tc.wantReady {
				t.Errorf("returned Ready = %q, want %q", got, tc.wantReady)
			}
			avail := meta.FindStatusCondition(cs.Conditions, ConditionAvailable)
			if avail == nil {
				t.Fatalf("Available condition not set")
			}
			if avail.Status != tc.wantAvail {
				t.Errorf("Available status = %q, want %q", avail.Status, tc.wantAvail)
			}
			if avail.Reason != tc.wantReason {
				t.Errorf("Available reason = %q, want %q", avail.Reason, tc.wantReason)
			}
			if avail.ObservedGeneration != 7 {
				t.Errorf("Available observedGeneration = %d, want 7", avail.ObservedGeneration)
			}
			ready := meta.FindStatusCondition(cs.Conditions, ConditionReady)
			if ready == nil {
				t.Fatalf("Ready condition not set")
			}
			if ready.Status != tc.wantReady {
				t.Errorf("Ready status = %q, want %q", ready.Status, tc.wantReady)
			}
		})
	}
}

// TestApplyWorkloadConditions_PreservesTransitionTime asserts the condition
// transition time is preserved across re-applies when the status is unchanged,
// proving we use meta.SetStatusCondition semantics rather than overwriting.
func TestApplyWorkloadConditions_PreservesTransitionTime(t *testing.T) {
	cs := &zelosv1alpha1.CommonStatus{}
	applyWorkloadConditions(cs, 1, workloadReadiness{Found: true, Available: true, Reason: ReasonWorkloadAvailable, Message: "1/1"})
	first := meta.FindStatusCondition(cs.Conditions, ConditionReady).LastTransitionTime

	// Re-apply with the same status: transition time must not move.
	applyWorkloadConditions(cs, 2, workloadReadiness{Found: true, Available: true, Reason: ReasonWorkloadAvailable, Message: "1/1"})
	second := meta.FindStatusCondition(cs.Conditions, ConditionReady).LastTransitionTime
	if !first.Equal(&second) {
		t.Errorf("transition time moved on unchanged status: %v -> %v", first, second)
	}

	// Flip status: transition time should now change.
	applyWorkloadConditions(cs, 3, workloadReadiness{Found: true, Available: false, Reason: ReasonWorkloadUnavailable, Message: "0/1"})
	third := meta.FindStatusCondition(cs.Conditions, ConditionReady).LastTransitionTime
	if third.Equal(&second) {
		t.Errorf("transition time did not change on status flip")
	}
}
