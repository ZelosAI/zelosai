//go:build e2e

// Package e2e holds the kind-based end-to-end smoke for the zelosai operator.
//
// IMPORTANT: this file is guarded by the `e2e` build tag so it is NOT compiled
// or run by the default `go test ./...` unit pass. It requires a live cluster
// (a kind cluster in CI) with the operator + CRDs + the e2e ZelosPlatform
// overlay already installed. The cluster bring-up, image build/load, and
// manifest apply are performed by `make test-e2e` (see the Makefile) and the
// `.github/workflows/e2e.yml` job — this test only performs the assertions.
//
// Run via:  make test-e2e
// or, against an already-prepared cluster:
//
//	go test -tags e2e ./test/e2e/... -timeout 12m
//
// It is build-only in any environment without kind; `go vet -tags e2e ./test/...`
// type-checks it without a cluster.
package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	zelosv1alpha1 "github.com/ZelosAI/zelosai/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// platformReadyBudget is the runtime budget from issue #35 — fail fast
	// above ~8 minutes. Overridable via E2E_READY_TIMEOUT (Go duration).
	platformReadyBudget = 8 * time.Minute
	pollInterval        = 5 * time.Second

	platformName      = "default"
	platformNamespace = "zelos"

	// conditionReady mirrors internal/controller.ConditionReady. Duplicated
	// here so the e2e package doesn't import internal/.
	conditionReady = "Ready"
)

func readyBudget() time.Duration {
	if v := os.Getenv("E2E_READY_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return platformReadyBudget
}

func e2eScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(s); err != nil {
		t.Fatalf("add client-go scheme: %v", err)
	}
	if err := zelosv1alpha1.AddToScheme(s); err != nil {
		t.Fatalf("add zelos scheme: %v", err)
	}
	return s
}

func newClient(t *testing.T) client.Client {
	t.Helper()
	cfg, err := ctrl.GetConfig()
	if err != nil {
		t.Fatalf("load kubeconfig (is KUBECONFIG pointed at the kind cluster?): %v", err)
	}
	c, err := client.New(cfg, client.Options{Scheme: e2eScheme(t)})
	if err != nil {
		t.Fatalf("build client: %v", err)
	}
	return c
}

// TestPlatformReachesReady is the core #35 assertion: after the operator +
// e2e ZelosPlatform overlay are installed, ZelosPlatform/default must reach a
// Ready=True condition with a populated .status.conditions block (one per
// enabled component plus the top-level Ready) within the runtime budget.
func TestPlatformReachesReady(t *testing.T) {
	c := newClient(t)
	ctx := context.Background()
	budget := readyBudget()
	deadline := time.Now().Add(budget)
	key := types.NamespacedName{Name: platformName, Namespace: platformNamespace}

	t.Logf("waiting up to %s for ZelosPlatform/%s to report Ready=True", budget, platformName)

	var last zelosv1alpha1.ZelosPlatform
	for time.Now().Before(deadline) {
		var p zelosv1alpha1.ZelosPlatform
		if err := c.Get(ctx, key, &p); err != nil {
			t.Logf("get ZelosPlatform: %v (retrying)", err)
			time.Sleep(pollInterval)
			continue
		}
		last = p

		ready := meta.FindStatusCondition(p.Status.Conditions, conditionReady)
		if ready != nil {
			t.Logf("phase=%q Ready=%s reason=%q (%d conditions)",
				p.Status.Phase, ready.Status, ready.Reason, len(p.Status.Conditions))
			if ready.Status == metav1.ConditionTrue {
				assertConditionsPopulated(t, p)
				t.Logf("ZelosPlatform/%s reached Ready in %s", platformName, budget-time.Until(deadline))
				return
			}
		} else {
			t.Logf("phase=%q Ready condition not present yet (%d conditions)",
				p.Status.Phase, len(p.Status.Conditions))
		}
		time.Sleep(pollInterval)
	}

	dumpDiagnostics(t, c, last)
	t.Fatalf("ZelosPlatform/%s did not reach Ready=True within %s", platformName, budget)
}

// assertConditionsPopulated verifies the status block carries the per-component
// Available conditions plus the children roll-up, proving the #34 feature is
// wired, not just the top-level Ready.
func assertConditionsPopulated(t *testing.T, p zelosv1alpha1.ZelosPlatform) {
	t.Helper()
	if len(p.Status.Conditions) == 0 {
		t.Errorf(".status.conditions is empty on a Ready platform")
	}
	if p.Status.Phase == "" {
		t.Errorf(".status.phase is empty on a Ready platform")
	}
	// Every enabled child should have a per-component Available=True condition
	// and a rolled-up child Ready status.
	for _, child := range p.Status.Children {
		label := child.Kind
		if len(label) > 5 && label[:5] == "Zelos" {
			label = label[5:]
		}
		condType := label + "Available"
		cond := meta.FindStatusCondition(p.Status.Conditions, condType)
		if cond == nil {
			t.Errorf("missing per-component condition %q for child %s/%s", condType, child.Kind, child.Name)
			continue
		}
		if cond.Status != metav1.ConditionTrue {
			t.Errorf("component %q condition = %s, want True on a Ready platform", condType, cond.Status)
		}
		if child.Ready != string(metav1.ConditionTrue) {
			t.Errorf("child %s/%s rolled-up Ready = %q, want True", child.Kind, child.Name, child.Ready)
		}
	}
}

// dumpDiagnostics logs the platform status + child CR conditions on failure so
// the CI log carries the expected condition message (per the #35 verification:
// a broken component must surface its condition message in the logs).
func dumpDiagnostics(t *testing.T, c client.Client, p zelosv1alpha1.ZelosPlatform) {
	t.Helper()
	t.Logf("=== ZelosPlatform/%s diagnostics ===", platformName)
	t.Logf("phase=%q observedGeneration=%d children=%d",
		p.Status.Phase, p.Status.ObservedGeneration, len(p.Status.Children))
	for _, cond := range p.Status.Conditions {
		t.Logf("  condition %-22s status=%-7s reason=%-24s msg=%q",
			cond.Type, cond.Status, cond.Reason, cond.Message)
	}
	for _, child := range p.Status.Children {
		t.Logf("  child %-16s %-12s ready=%-7s msg=%q",
			child.Kind, child.Name, child.Ready, child.Message)
	}
	dumpChildCR(t, c, &zelosv1alpha1.ZelosBackplane{}, "ZelosBackplane")
	dumpChildCR(t, c, &zelosv1alpha1.ZelosGateway{}, "ZelosGateway")
	dumpChildCR(t, c, &zelosv1alpha1.ZelosMCP{}, "ZelosMCP")
}

func dumpChildCR(t *testing.T, c client.Client, obj client.Object, kind string) {
	t.Helper()
	key := types.NamespacedName{Name: platformName, Namespace: platformNamespace}
	if err := c.Get(context.Background(), key, obj); err != nil {
		return
	}
	conds := childConditions(obj)
	t.Logf("--- %s/%s conditions ---", kind, platformName)
	for _, cond := range conds {
		t.Logf("  %-12s status=%-7s reason=%-24s msg=%q", cond.Type, cond.Status, cond.Reason, cond.Message)
	}
}

func childConditions(obj client.Object) []metav1.Condition {
	switch o := obj.(type) {
	case *zelosv1alpha1.ZelosBackplane:
		return o.Status.Conditions
	case *zelosv1alpha1.ZelosGateway:
		return o.Status.Conditions
	case *zelosv1alpha1.ZelosMCP:
		return o.Status.Conditions
	}
	return nil
}
