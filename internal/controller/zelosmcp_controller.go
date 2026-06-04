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

type ZelosMCPReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=zelos.zelosai.io,resources=zelosmcps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=zelos.zelosai.io,resources=zelosmcps/status,verbs=get;update;patch

func (r *ZelosMCPReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var mcp zelosv1alpha1.ZelosMCP
	if err := r.Get(ctx, req.NamespacedName, &mcp); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	c := render.Defaults("zelosmcp")

	// Default to persistence on for MCP (SQLite + auth.key).
	if mcp.Spec.Persistence == nil {
		mcp.Spec.Persistence = &zelosv1alpha1.PersistenceSpec{Enabled: true, Size: "1Gi"}
	}

	// If AuthProviderSecretRef is set, inject standard SecretMounts for auth.key + providers.json.
	if mcp.Spec.AuthProviderSecretRef != "" {
		need := []zelosv1alpha1.SecretMount{
			{Name: mcp.Spec.AuthProviderSecretRef, Key: "auth.key"},
			{Name: mcp.Spec.AuthProviderSecretRef, Key: "providers.json"},
		}
		for _, n := range need {
			present := false
			for _, m := range mcp.Spec.SecretRefs {
				if m.Name == n.Name && m.Key == n.Key {
					present = true
					break
				}
			}
			if !present {
				mcp.Spec.SecretRefs = append(mcp.Spec.SecretRefs, n)
			}
		}
	}

	cfg := map[string]string{}
	for k, v := range mcp.Spec.Config {
		cfg[k] = v
	}
	if mcp.Spec.MCPServersConfigMapRef != "" {
		cfg["ZELOSMCP_SERVERS_CONFIGMAP"] = mcp.Spec.MCPServersConfigMapRef
	}
	cm := render.BuildConfigMap(&mcp, c, cfg)
	if err := applyObject(ctx, r.Client, r.Scheme, &mcp, cm); err != nil {
		return ctrl.Result{}, err
	}

	if pvc := render.BuildPVC(&mcp, c, mcp.Spec.Persistence); pvc != nil {
		if err := applyObject(ctx, r.Client, r.Scheme, &mcp, pvc); err != nil {
			return ctrl.Result{}, err
		}
	}

	dep := render.BuildDeployment(render.DeploymentInput{
		Owner:       &mcp,
		Component:   c,
		Workload:    mcp.Spec.WorkloadSpec,
		ConfigMap:   cm.Name,
		TelemetryCM: render.TelemetryEnvConfigMapName(mcp.Name),
		PullSecret:  "ghcr-pull-secret",
	})
	if err := applyObject(ctx, r.Client, r.Scheme, &mcp, dep); err != nil {
		return ctrl.Result{}, err
	}

	svc := render.BuildService(&mcp, c)
	if err := applyObject(ctx, r.Client, r.Scheme, &mcp, svc); err != nil {
		return ctrl.Result{}, err
	}

	wr := deploymentReadiness(ctx, r.Client, types.NamespacedName{Name: dep.Name, Namespace: dep.Namespace})
	applyWorkloadConditions(&mcp.Status.CommonStatus, mcp.Generation, wr)
	mcp.Status.ObservedGeneration = mcp.Generation
	_ = r.Status().Update(ctx, &mcp)
	if !wr.Available {
		return ctrl.Result{RequeueAfter: componentRequeueInterval()}, nil
	}
	return ctrl.Result{}, nil
}

func (r *ZelosMCPReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&zelosv1alpha1.ZelosMCP{}).
		Complete(r)
}
