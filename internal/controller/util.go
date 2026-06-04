package controller

import (
	"context"
	"fmt"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// applyObject creates the object if absent or updates it if its spec changed.
// `current` and `desired` MUST be the same concrete type. Caller is expected to
// have set the owner reference on `desired`.
func applyObject(ctx context.Context, c client.Client, scheme *runtime.Scheme, owner metav1.Object, desired client.Object) error {
	if err := controllerutil.SetControllerReference(owner, desired, scheme); err != nil {
		return fmt.Errorf("set owner ref: %w", err)
	}

	key := client.ObjectKeyFromObject(desired)
	current := desired.DeepCopyObject().(client.Object)
	err := c.Get(ctx, key, current)
	if apierrors.IsNotFound(err) {
		return c.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	// Preserve resourceVersion for update.
	desired.SetResourceVersion(current.GetResourceVersion())
	if equality.Semantic.DeepEqual(current, desired) {
		return nil
	}
	return c.Update(ctx, desired)
}

// requeueResult is the default short requeue for transient errors.
func requeueResult() (ctrl.Result, error) { return ctrl.Result{Requeue: true}, nil }

// defaultReconcileInterval is the EA poll cadence used to re-evaluate managed
// workload readiness so status conditions track Deployment/StatefulSet health
// even without a watch event. Tunable via ZELOS_RECONCILE_INTERVAL (a Go
// duration string, e.g. "15s"). See issue #34.
const defaultReconcileInterval = 30 * time.Second

// componentRequeueInterval returns the readiness re-evaluation cadence,
// honoring the ZELOS_RECONCILE_INTERVAL env override when it parses to a
// positive duration.
func componentRequeueInterval() time.Duration {
	if v := os.Getenv("ZELOS_RECONCILE_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return defaultReconcileInterval
}
