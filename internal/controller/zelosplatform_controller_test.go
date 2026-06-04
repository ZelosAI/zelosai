package controller

import (
	"context"
	"testing"

	zelosv1alpha1 "github.com/ZelosAI/zelosai/api/v1alpha1"
	"github.com/ZelosAI/zelosai/internal/controller/render"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// readyChild returns a child CR object (by Kind) carrying a Ready condition of
// the given status, for seeding the fake client.
func readyChild(kind, name string, ready metav1.ConditionStatus) client.Object {
	cs := zelosv1alpha1.CommonStatus{
		Conditions: []metav1.Condition{{
			Type:               ConditionReady,
			Status:             ready,
			Reason:             "Test",
			Message:            "seeded by test",
			LastTransitionTime: metav1.Now(),
		}},
	}
	om := metav1.ObjectMeta{Name: name, Namespace: "zelos"}
	switch kind {
	case "ZelosGateway":
		return &zelosv1alpha1.ZelosGateway{ObjectMeta: om, Status: zelosv1alpha1.ZelosGatewayStatus{CommonStatus: cs}}
	case "ZelosBackplane":
		return &zelosv1alpha1.ZelosBackplane{ObjectMeta: om, Status: zelosv1alpha1.ZelosBackplaneStatus{CommonStatus: cs}}
	case "ZelosMCP":
		return &zelosv1alpha1.ZelosMCP{ObjectMeta: om, Status: zelosv1alpha1.ZelosMCPStatus{CommonStatus: cs}}
	case "ZelosBroker":
		return &zelosv1alpha1.ZelosBroker{ObjectMeta: om, Status: zelosv1alpha1.ZelosBrokerStatus{CommonStatus: cs}}
	case "ZelosServer":
		return &zelosv1alpha1.ZelosServer{ObjectMeta: om, Status: zelosv1alpha1.ZelosServerStatus{CommonStatus: cs}}
	case "ZelosClient":
		return &zelosv1alpha1.ZelosClient{ObjectMeta: om, Status: zelosv1alpha1.ZelosClientStatus{CommonStatus: cs}}
	}
	return nil
}

func condStatus(conds []metav1.Condition, t string) metav1.ConditionStatus {
	if c := meta.FindStatusCondition(conds, t); c != nil {
		return c.Status
	}
	return metav1.ConditionUnknown
}

// TestReconcileStatusConditions_RollUp drives the umbrella roll-up across a
// matrix of component readiness states, asserting per-component Available
// conditions, the top-level Ready condition, and the human phase.
func TestReconcileStatusConditions_RollUp(t *testing.T) {
	const ns = "zelos"

	cases := []struct {
		name              string
		gatewayReady      metav1.ConditionStatus
		backplaneReady    metav1.ConditionStatus
		mcpReady          metav1.ConditionStatus
		natsReady         bool // underlying NATS StatefulSet ready
		collectorReady    bool // OTel collector deployment ready
		wantTopReady      metav1.ConditionStatus
		wantPhase         string
		wantGatewayAvail  metav1.ConditionStatus
		wantNATSAvailable metav1.ConditionStatus
	}{
		{
			name:         "everything ready",
			gatewayReady: metav1.ConditionTrue, backplaneReady: metav1.ConditionTrue, mcpReady: metav1.ConditionTrue,
			natsReady: true, collectorReady: true,
			wantTopReady: metav1.ConditionTrue, wantPhase: PhaseReady,
			wantGatewayAvail: metav1.ConditionTrue, wantNATSAvailable: metav1.ConditionTrue,
		},
		{
			name:         "gateway not ready",
			gatewayReady: metav1.ConditionFalse, backplaneReady: metav1.ConditionTrue, mcpReady: metav1.ConditionTrue,
			natsReady: true, collectorReady: true,
			wantTopReady: metav1.ConditionFalse, wantPhase: PhaseProgressing,
			wantGatewayAvail: metav1.ConditionFalse, wantNATSAvailable: metav1.ConditionTrue,
		},
		{
			name:         "nats statefulset not ready",
			gatewayReady: metav1.ConditionTrue, backplaneReady: metav1.ConditionTrue, mcpReady: metav1.ConditionTrue,
			natsReady: false, collectorReady: true,
			wantTopReady: metav1.ConditionFalse, wantPhase: PhaseProgressing,
			wantGatewayAvail: metav1.ConditionTrue, wantNATSAvailable: metav1.ConditionFalse,
		},
		{
			name:         "collector not ready blocks operator component",
			gatewayReady: metav1.ConditionTrue, backplaneReady: metav1.ConditionTrue, mcpReady: metav1.ConditionTrue,
			natsReady: true, collectorReady: false,
			wantTopReady: metav1.ConditionFalse, wantPhase: PhaseProgressing,
			wantGatewayAvail: metav1.ConditionTrue, wantNATSAvailable: metav1.ConditionTrue,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			platform := &zelosv1alpha1.ZelosPlatform{
				ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: ns, Generation: 1},
				Spec: zelosv1alpha1.ZelosPlatformSpec{
					Backplane: zelosv1alpha1.PlatformBackplane{
						PlatformComponent: zelosv1alpha1.PlatformComponent{Enabled: true},
						Substrate:         "nats",
					},
				},
			}

			objs := []client.Object{
				platform,
				readyChild("ZelosBackplane", "default", tc.backplaneReady),
				readyChild("ZelosGateway", "default", tc.gatewayReady),
				readyChild("ZelosMCP", "default", tc.mcpReady),
			}

			// Underlying workloads polled directly by the umbrella.
			collectorAvail := int32(0)
			if tc.collectorReady {
				collectorAvail = 1
			}
			objs = append(objs, newDeployment(render.TelemetryCollectorName, 1, collectorAvail))

			natsReady := int32(0)
			if tc.natsReady {
				natsReady = 1
			}
			objs = append(objs, newStatefulSet(render.NATSName("default"), 1, natsReady))

			c := fake.NewClientBuilder().
				WithScheme(testScheme(t)).
				WithObjects(objs...).
				WithStatusSubresource(
					&zelosv1alpha1.ZelosPlatform{},
					&zelosv1alpha1.ZelosGateway{},
					&zelosv1alpha1.ZelosBackplane{},
					&zelosv1alpha1.ZelosMCP{},
				).
				Build()

			r := &ZelosPlatformReconciler{Client: c, Scheme: testScheme(t)}

			// children in the order the umbrella appends them.
			children := []zelosv1alpha1.ChildRef{
				{Kind: "ZelosBackplane", Name: "default"},
				{Kind: "ZelosGateway", Name: "default"},
				{Kind: "ZelosMCP", Name: "default"},
			}

			enriched, allReady := r.reconcileStatusConditions(context.Background(), platform, children, true)

			if got := condStatus(platform.Status.Conditions, ConditionReady); got != tc.wantTopReady {
				t.Errorf("top-level Ready = %q, want %q", got, tc.wantTopReady)
			}
			if platform.Status.Phase != tc.wantPhase {
				t.Errorf("phase = %q, want %q", platform.Status.Phase, tc.wantPhase)
			}
			if got := condStatus(platform.Status.Conditions, componentConditionType("Gateway")); got != tc.wantGatewayAvail {
				t.Errorf("GatewayAvailable = %q, want %q", got, tc.wantGatewayAvail)
			}
			if got := condStatus(platform.Status.Conditions, componentConditionType("NATS")); got != tc.wantNATSAvailable {
				t.Errorf("NATSAvailable = %q, want %q", got, tc.wantNATSAvailable)
			}
			// Operator (OTel collector) condition reflects collector readiness.
			wantOperator := metav1.ConditionFalse
			if tc.collectorReady {
				wantOperator = metav1.ConditionTrue
			}
			if got := condStatus(platform.Status.Conditions, componentConditionType("Operator")); got != wantOperator {
				t.Errorf("OperatorAvailable = %q, want %q", got, wantOperator)
			}

			if allReady != (tc.wantTopReady == metav1.ConditionTrue) {
				t.Errorf("allReady = %v, want %v", allReady, tc.wantTopReady == metav1.ConditionTrue)
			}

			// Enriched children carry their rolled-up Ready status.
			for _, ch := range enriched {
				if ch.Kind == "ZelosGateway" && ch.Ready != string(tc.wantGatewayAvail) {
					t.Errorf("child Gateway Ready = %q, want %q", ch.Ready, tc.wantGatewayAvail)
				}
			}
		})
	}
}

// TestReconcileStatusConditions_NoComponents asserts the empty-spec path sets
// the NoComponentsEnabled reason and Pending phase.
func TestReconcileStatusConditions_NoComponents(t *testing.T) {
	platform := &zelosv1alpha1.ZelosPlatform{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "zelos", Generation: 1},
	}
	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(platform).Build()
	r := &ZelosPlatformReconciler{Client: c, Scheme: testScheme(t)}

	_, allReady := r.reconcileStatusConditions(context.Background(), platform, nil, false)
	if allReady {
		t.Errorf("allReady = true, want false for empty spec")
	}
	if platform.Status.Phase != PhasePending {
		t.Errorf("phase = %q, want %q", platform.Status.Phase, PhasePending)
	}
	ready := meta.FindStatusCondition(platform.Status.Conditions, ConditionReady)
	if ready == nil || ready.Reason != ReasonNoComponents {
		t.Errorf("Ready condition = %+v, want reason %q", ready, ReasonNoComponents)
	}
}

// TestReconcile_FullPath drives the full Reconcile entrypoint against a fake
// client to prove it materializes children and writes a populated status block
// (Progressing, since the workloads it just created report no replicas yet).
func TestReconcile_FullPath(t *testing.T) {
	platform := &zelosv1alpha1.ZelosPlatform{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "zelos", Generation: 1},
		Spec: zelosv1alpha1.ZelosPlatformSpec{
			Backplane: zelosv1alpha1.PlatformBackplane{
				PlatformComponent: zelosv1alpha1.PlatformComponent{Enabled: true},
				Substrate:         "nats",
			},
			Gateway: zelosv1alpha1.PlatformComponent{Enabled: true},
		},
	}
	s := testScheme(t)
	c := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(platform).
		WithStatusSubresource(
			&zelosv1alpha1.ZelosPlatform{},
			&zelosv1alpha1.ZelosGateway{},
			&zelosv1alpha1.ZelosBackplane{},
		).
		Build()
	r := &ZelosPlatformReconciler{Client: c, Scheme: s}

	res, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "default", Namespace: "zelos"}})
	if err != nil {
		t.Fatalf("Reconcile error: %v", err)
	}
	if res.RequeueAfter == 0 {
		t.Errorf("expected a requeue while components are not ready")
	}

	// Children were materialized.
	var bp zelosv1alpha1.ZelosBackplane
	if err := c.Get(context.Background(), types.NamespacedName{Name: "default", Namespace: "zelos"}, &bp); err != nil {
		t.Fatalf("expected ZelosBackplane to be created: %v", err)
	}

	// Platform status has a populated conditions block + a phase.
	var got zelosv1alpha1.ZelosPlatform
	if err := c.Get(context.Background(), types.NamespacedName{Name: "default", Namespace: "zelos"}, &got); err != nil {
		t.Fatalf("get platform: %v", err)
	}
	if len(got.Status.Conditions) == 0 {
		t.Fatalf("status.conditions is empty, want populated")
	}
	if got.Status.Phase != PhaseProgressing {
		t.Errorf("phase = %q, want %q (workloads not ready yet)", got.Status.Phase, PhaseProgressing)
	}
	if meta.FindStatusCondition(got.Status.Conditions, ConditionReady) == nil {
		t.Errorf("top-level Ready condition missing")
	}
	if meta.FindStatusCondition(got.Status.Conditions, componentConditionType("Gateway")) == nil {
		t.Errorf("GatewayAvailable condition missing")
	}
}
