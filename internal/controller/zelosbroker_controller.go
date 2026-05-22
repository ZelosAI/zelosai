package controller

import (
	"context"

	zelosv1alpha1 "github.com/ZelosAI/zelosai/api/v1alpha1"
	"github.com/ZelosAI/zelosai/internal/controller/render"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ZelosBrokerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=zelos.zelosai.io,resources=zelosbrokers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=zelos.zelosai.io,resources=zelosbrokers/status,verbs=get;update;patch

func (r *ZelosBrokerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var br zelosv1alpha1.ZelosBroker
	if err := r.Get(ctx, req.NamespacedName, &br); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	c := render.Defaults("zelosbroker")

	cfg := map[string]string{}
	for k, v := range br.Spec.Config {
		cfg[k] = v
	}
	if br.Spec.TunnelTransport != "" {
		cfg["ZELOSBROKER_TUNNEL_TRANSPORT"] = br.Spec.TunnelTransport
	}

	cm := render.BuildConfigMap(&br, c, cfg)
	if err := applyObject(ctx, r.Client, r.Scheme, &br, cm); err != nil {
		return ctrl.Result{}, err
	}

	if pvc := render.BuildPVC(&br, c, br.Spec.Persistence); pvc != nil {
		if err := applyObject(ctx, r.Client, r.Scheme, &br, pvc); err != nil {
			return ctrl.Result{}, err
		}
	}

	dep := render.BuildDeployment(render.DeploymentInput{
		Owner:       &br,
		Component:   c,
		Workload:    br.Spec.WorkloadSpec,
		ConfigMap:   cm.Name,
		TelemetryCM: render.TelemetryEnvConfigMapName(br.Name),
		PullSecret:  "ghcr-pull-secret",
	})
	if err := applyObject(ctx, r.Client, r.Scheme, &br, dep); err != nil {
		return ctrl.Result{}, err
	}

	svc := render.BuildService(&br, c)
	if err := applyObject(ctx, r.Client, r.Scheme, &br, svc); err != nil {
		return ctrl.Result{}, err
	}

	br.Status.ObservedGeneration = br.Generation
	_ = r.Status().Update(ctx, &br)
	return ctrl.Result{}, nil
}

func (r *ZelosBrokerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&zelosv1alpha1.ZelosBroker{}).
		Complete(r)
}
