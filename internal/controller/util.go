package controller

import (
	"context"
	"fmt"

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
