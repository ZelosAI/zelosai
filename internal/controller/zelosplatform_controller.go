package controller

import (
	"context"

	zelosv1alpha1 "github.com/ZelosAI/zelosai/api/v1alpha1"
	"github.com/ZelosAI/zelosai/internal/controller/render"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ZelosPlatformReconciler owns the umbrella ZelosPlatform resource. It does not
// create Deployments directly; it materializes per-component CRs (ZelosGateway,
// ZelosBackplane, ZelosMCP, ZelosBroker, ZelosServer, ZelosClient) plus the
// suite-wide OTel collector + env ConfigMap.
type ZelosPlatformReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=zelos.zelosai.io,resources=zelosplatforms,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=zelos.zelosai.io,resources=zelosplatforms/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=zelos.zelosai.io,resources=zelosplatforms/finalizers,verbs=update
// +kubebuilder:rbac:groups=zelos.zelosai.io,resources=zelosgateways;zelosbackplanes;zelosmcps;zelosbrokers;zelosservers;zelosclients,verbs=get;list;watch;create;update;patch;delete

func (r *ZelosPlatformReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var platform zelosv1alpha1.ZelosPlatform
	if err := r.Get(ctx, req.NamespacedName, &platform); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// 1. Telemetry env ConfigMap (always rendered; mounted by every child workload).
	envCM := render.BuildTelemetryEnvConfigMap(&platform, platform.Spec.Telemetry)
	if err := applyObject(ctx, r.Client, r.Scheme, &platform, envCM); err != nil {
		return ctrl.Result{}, err
	}

	// 2. OTel collector (skipped when externalEndpoint is set).
	enabled := true
	if platform.Spec.Telemetry.Enabled != nil {
		enabled = *platform.Spec.Telemetry.Enabled
	}
	if enabled && platform.Spec.Telemetry.ExternalEndpoint == "" {
		if err := applyObject(ctx, r.Client, r.Scheme, &platform, render.BuildCollectorConfigMap(&platform)); err != nil {
			return ctrl.Result{}, err
		}
		if err := applyObject(ctx, r.Client, r.Scheme, &platform, render.BuildCollectorDeployment(&platform, platform.Spec.Telemetry)); err != nil {
			return ctrl.Result{}, err
		}
		if err := applyObject(ctx, r.Client, r.Scheme, &platform, render.BuildCollectorService(&platform)); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 3. Materialize per-component CRs from the umbrella spec.
	var children []zelosv1alpha1.ChildRef

	pullSecret := platform.Spec.ImagePullSecret
	if pullSecret == "" {
		pullSecret = "ghcr-pull-secret"
	}

	if platform.Spec.Backplane.Enabled {
		bp := &zelosv1alpha1.ZelosBackplane{
			ObjectMeta: metav1.ObjectMeta{Name: platform.Name, Namespace: platform.Namespace},
			Spec: zelosv1alpha1.ZelosBackplaneSpec{
				WorkloadSpec: zelosv1alpha1.WorkloadSpec{
					Replicas:         platform.Spec.Backplane.Replicas,
					Image:            mergeImage(platform.Spec.Defaults.Image, platform.Spec.Backplane.Image),
					Resources:        platform.Spec.Defaults.Resources,
					Persistence:      platform.Spec.Backplane.Persistence,
					Config:           platform.Spec.Backplane.Config,
					SecretRefs:       platform.Spec.Backplane.SecretRefs,
					Autoscaling:      platform.Spec.Backplane.Autoscaling,
					ImagePullSecrets: []corev1.LocalObjectReference{{Name: pullSecret}},
					LogLevel:         platform.Spec.Telemetry.LogLevel,
				},
				Substrate:    platform.Spec.Backplane.Substrate,
				ExternalURL:  platform.Spec.Backplane.ExternalURL,
				TLSSecretRef: platform.Spec.Backplane.TLSSecretRef,
			},
		}
		if err := applyObject(ctx, r.Client, r.Scheme, &platform, bp); err != nil {
			return ctrl.Result{}, err
		}
		children = append(children, zelosv1alpha1.ChildRef{Kind: "ZelosBackplane", Name: bp.Name})
	}

	if platform.Spec.Gateway.Enabled {
		gw := &zelosv1alpha1.ZelosGateway{
			ObjectMeta: metav1.ObjectMeta{Name: platform.Name, Namespace: platform.Namespace},
			Spec: zelosv1alpha1.ZelosGatewaySpec{
				WorkloadSpec: zelosv1alpha1.WorkloadSpec{
					Replicas:         platform.Spec.Gateway.Replicas,
					Image:            mergeImage(platform.Spec.Defaults.Image, platform.Spec.Gateway.Image),
					Resources:        platform.Spec.Defaults.Resources,
					Config:           platform.Spec.Gateway.Config,
					SecretRefs:       platform.Spec.Gateway.SecretRefs,
					Autoscaling:      platform.Spec.Gateway.Autoscaling,
					ImagePullSecrets: []corev1.LocalObjectReference{{Name: pullSecret}},
					LogLevel:         platform.Spec.Telemetry.LogLevel,
				},
				BackplaneRef: platform.Name,
				MCPRef:       platform.Name,
			},
		}
		if err := applyObject(ctx, r.Client, r.Scheme, &platform, gw); err != nil {
			return ctrl.Result{}, err
		}
		children = append(children, zelosv1alpha1.ChildRef{Kind: "ZelosGateway", Name: gw.Name})
	}

	if platform.Spec.MCP.Enabled {
		mcp := &zelosv1alpha1.ZelosMCP{
			ObjectMeta: metav1.ObjectMeta{Name: platform.Name, Namespace: platform.Namespace},
			Spec: zelosv1alpha1.ZelosMCPSpec{
				WorkloadSpec: zelosv1alpha1.WorkloadSpec{
					Replicas:         platform.Spec.MCP.Replicas,
					Image:            mergeImage(platform.Spec.Defaults.Image, platform.Spec.MCP.Image),
					Resources:        platform.Spec.Defaults.Resources,
					Persistence:      platform.Spec.MCP.Persistence,
					Config:           platform.Spec.MCP.Config,
					SecretRefs:       platform.Spec.MCP.SecretRefs,
					ImagePullSecrets: []corev1.LocalObjectReference{{Name: pullSecret}},
					LogLevel:         platform.Spec.Telemetry.LogLevel,
				},
				AuthProviderSecretRef:  platform.Spec.MCP.AuthProviderSecretRef,
				MCPServersConfigMapRef: platform.Spec.MCP.MCPServersConfigMapRef,
			},
		}
		if err := applyObject(ctx, r.Client, r.Scheme, &platform, mcp); err != nil {
			return ctrl.Result{}, err
		}
		children = append(children, zelosv1alpha1.ChildRef{Kind: "ZelosMCP", Name: mcp.Name})
	}

	if platform.Spec.Broker.Enabled {
		br := &zelosv1alpha1.ZelosBroker{
			ObjectMeta: metav1.ObjectMeta{Name: platform.Name, Namespace: platform.Namespace},
			Spec: zelosv1alpha1.ZelosBrokerSpec{
				WorkloadSpec: zelosv1alpha1.WorkloadSpec{
					Replicas:         platform.Spec.Broker.Replicas,
					Image:            mergeImage(platform.Spec.Defaults.Image, platform.Spec.Broker.Image),
					Resources:        platform.Spec.Defaults.Resources,
					Persistence:      platform.Spec.Broker.Persistence,
					Config:           platform.Spec.Broker.Config,
					SecretRefs:       platform.Spec.Broker.SecretRefs,
					ImagePullSecrets: []corev1.LocalObjectReference{{Name: pullSecret}},
					LogLevel:         platform.Spec.Telemetry.LogLevel,
				},
				TunnelTransport: platform.Spec.Broker.TunnelTransport,
				AllowedLLMHosts: platform.Spec.Broker.AllowedLLMHosts,
			},
		}
		if err := applyObject(ctx, r.Client, r.Scheme, &platform, br); err != nil {
			return ctrl.Result{}, err
		}
		children = append(children, zelosv1alpha1.ChildRef{Kind: "ZelosBroker", Name: br.Name})
	}

	if platform.Spec.Server.Enabled {
		sv := &zelosv1alpha1.ZelosServer{
			ObjectMeta: metav1.ObjectMeta{Name: platform.Name, Namespace: platform.Namespace},
			Spec: zelosv1alpha1.ZelosServerSpec{
				WorkloadSpec: zelosv1alpha1.WorkloadSpec{
					Replicas:         platform.Spec.Server.Replicas,
					Image:            mergeImage(platform.Spec.Defaults.Image, platform.Spec.Server.Image),
					Resources:        platform.Spec.Defaults.Resources,
					Config:           platform.Spec.Server.Config,
					SecretRefs:       platform.Spec.Server.SecretRefs,
					ImagePullSecrets: []corev1.LocalObjectReference{{Name: pullSecret}},
					LogLevel:         platform.Spec.Telemetry.LogLevel,
				},
			},
		}
		if err := applyObject(ctx, r.Client, r.Scheme, &platform, sv); err != nil {
			return ctrl.Result{}, err
		}
		children = append(children, zelosv1alpha1.ChildRef{Kind: "ZelosServer", Name: sv.Name})
	}

	if platform.Spec.Client.Enabled {
		cl := &zelosv1alpha1.ZelosClient{
			ObjectMeta: metav1.ObjectMeta{Name: platform.Name, Namespace: platform.Namespace},
			Spec: zelosv1alpha1.ZelosClientSpec{
				WorkloadSpec: zelosv1alpha1.WorkloadSpec{
					Replicas:         platform.Spec.Client.Replicas,
					Image:            mergeImage(platform.Spec.Defaults.Image, platform.Spec.Client.Image),
					Resources:        platform.Spec.Defaults.Resources,
					Config:           platform.Spec.Client.Config,
					SecretRefs:       platform.Spec.Client.SecretRefs,
					ImagePullSecrets: []corev1.LocalObjectReference{{Name: pullSecret}},
					LogLevel:         platform.Spec.Telemetry.LogLevel,
				},
				InCluster:       platform.Spec.Client.InCluster,
				Runtime:         platform.Spec.Client.Runtime,
				RuntimeURL:      platform.Spec.Client.RuntimeURL,
				Model:           platform.Spec.Client.Model,
				SubscribeTopics: platform.Spec.Client.SubscribeTopics,
				BackplaneRef:    platform.Name,
			},
		}
		if err := applyObject(ctx, r.Client, r.Scheme, &platform, cl); err != nil {
			return ctrl.Result{}, err
		}
		children = append(children, zelosv1alpha1.ChildRef{Kind: "ZelosClient", Name: cl.Name})
	}

	// 4. Status (best-effort; ignore conflicts).
	platform.Status.ObservedGeneration = platform.Generation
	platform.Status.Children = children
	if err := r.Status().Update(ctx, &platform); err != nil && !apierrors.IsConflict(err) {
		logger.Error(err, "status update failed")
	}

	return ctrl.Result{}, nil
}

// SetupWithManager wires the ZelosPlatform reconciler.
func (r *ZelosPlatformReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&zelosv1alpha1.ZelosPlatform{}).
		Owns(&zelosv1alpha1.ZelosBackplane{}).
		Owns(&zelosv1alpha1.ZelosGateway{}).
		Owns(&zelosv1alpha1.ZelosMCP{}).
		Owns(&zelosv1alpha1.ZelosBroker{}).
		Owns(&zelosv1alpha1.ZelosServer{}).
		Owns(&zelosv1alpha1.ZelosClient{}).
		Complete(r)
}

// --- helpers -----------------------------------------------------------------

func mergeImage(base, override zelosv1alpha1.ImageSpec) zelosv1alpha1.ImageSpec {
	out := base
	if override.Repository != "" {
		out.Repository = override.Repository
	}
	if override.Tag != "" {
		out.Tag = override.Tag
	}
	if override.PullPolicy != "" {
		out.PullPolicy = override.PullPolicy
	}
	return out
}
