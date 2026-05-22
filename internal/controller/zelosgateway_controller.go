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

type ZelosGatewayReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=zelos.zelosai.io,resources=zelosgateways,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=zelos.zelosai.io,resources=zelosgateways/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=zelos.zelosai.io,resources=zelosbackplanes,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services;configmaps;serviceaccounts;persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete

func (r *ZelosGatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var gw zelosv1alpha1.ZelosGateway
	if err := r.Get(ctx, req.NamespacedName, &gw); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	c := render.Defaults("zelosgateway")
	platformName := gw.Name // single ZelosPlatform per namespace shares its name

	// Component-specific config (gateway needs the backplane URL etc.)
	cfg := map[string]string{}
	for k, v := range gw.Spec.Config {
		cfg[k] = v
	}
	if gw.Spec.BackplaneRef != "" {
		cfg["ZELOSBACKPLANE_URL"] = render.NATSURL(gw.Spec.BackplaneRef, gw.Namespace)
	}
	if gw.Spec.AuthProviderSecretRef != "" {
		cfg["ZELOSGATEWAY_AUTH_PROVIDER"] = "oidc"
	}
	cm := render.BuildConfigMap(&gw, c, cfg)
	if err := applyObject(ctx, r.Client, r.Scheme, &gw, cm); err != nil {
		return ctrl.Result{}, err
	}

	dep := render.BuildDeployment(render.DeploymentInput{
		Owner:       &gw,
		Component:   c,
		Workload:    gw.Spec.WorkloadSpec,
		ConfigMap:   cm.Name,
		TelemetryCM: render.TelemetryEnvConfigMapName(platformName),
		PullSecret:  "ghcr-pull-secret",
	})
	if err := applyObject(ctx, r.Client, r.Scheme, &gw, dep); err != nil {
		return ctrl.Result{}, err
	}

	svc := render.BuildService(&gw, c)
	if err := applyObject(ctx, r.Client, r.Scheme, &gw, svc); err != nil {
		return ctrl.Result{}, err
	}

	if hpa := render.BuildHPA(&gw, c, gw.Spec.Autoscaling); hpa != nil {
		if err := applyObject(ctx, r.Client, r.Scheme, &gw, hpa); err != nil {
			return ctrl.Result{}, err
		}
	}

	gw.Status.ObservedGeneration = gw.Generation
	_ = r.Status().Update(ctx, &gw)

	return ctrl.Result{}, nil
}

func (r *ZelosGatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&zelosv1alpha1.ZelosGateway{}).
		Complete(r)
}
