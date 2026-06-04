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

type ZelosServerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=zelos.zelosai.io,resources=zelosservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=zelos.zelosai.io,resources=zelosservers/status,verbs=get;update;patch

func (r *ZelosServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var sv zelosv1alpha1.ZelosServer
	if err := r.Get(ctx, req.NamespacedName, &sv); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	c := render.Defaults("zelosserver")

	cm := render.BuildConfigMap(&sv, c, sv.Spec.Config)
	if err := applyObject(ctx, r.Client, r.Scheme, &sv, cm); err != nil {
		return ctrl.Result{}, err
	}

	dep := render.BuildDeployment(render.DeploymentInput{
		Owner:       &sv,
		Component:   c,
		Workload:    sv.Spec.WorkloadSpec,
		ConfigMap:   cm.Name,
		TelemetryCM: render.TelemetryEnvConfigMapName(sv.Name),
		PullSecret:  "ghcr-pull-secret",
	})
	if err := applyObject(ctx, r.Client, r.Scheme, &sv, dep); err != nil {
		return ctrl.Result{}, err
	}

	svc := render.BuildService(&sv, c)
	if err := applyObject(ctx, r.Client, r.Scheme, &sv, svc); err != nil {
		return ctrl.Result{}, err
	}

	wr := deploymentReadiness(ctx, r.Client, types.NamespacedName{Name: dep.Name, Namespace: dep.Namespace})
	applyWorkloadConditions(&sv.Status.CommonStatus, sv.Generation, wr)
	sv.Status.ObservedGeneration = sv.Generation
	_ = r.Status().Update(ctx, &sv)
	if !wr.Available {
		return ctrl.Result{RequeueAfter: componentRequeueInterval()}, nil
	}
	return ctrl.Result{}, nil
}

func (r *ZelosServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&zelosv1alpha1.ZelosServer{}).
		Complete(r)
}
