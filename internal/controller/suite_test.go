package controller

import (
	"testing"

	zelosv1alpha1 "github.com/ZelosAI/zelosai/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

// testScheme builds a runtime.Scheme with the core/apps types plus the Zelos
// v1alpha1 CRDs registered, for use with the controller-runtime fake client.
func testScheme(t *testing.T) *runtime.Scheme {
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
