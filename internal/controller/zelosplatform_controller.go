package controller

import (
	"context"
	"fmt"
	"sort"
	"strings"

	zelosv1alpha1 "github.com/ZelosAI/zelosai/api/v1alpha1"
	"github.com/ZelosAI/zelosai/internal/controller/render"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
// +kubebuilder:rbac:groups=apps,resources=deployments;statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps;services,verbs=get;list;watch;create;update;patch;delete

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
	collectorDeployed := enabled && platform.Spec.Telemetry.ExternalEndpoint == ""
	if collectorDeployed {
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

	// 4. Roll up component readiness into status conditions + a human phase.
	enrichedChildren, allReady := r.reconcileStatusConditions(ctx, &platform, children, collectorDeployed)

	platform.Status.ObservedGeneration = platform.Generation
	platform.Status.Children = enrichedChildren
	if err := r.Status().Update(ctx, &platform); err != nil && !apierrors.IsConflict(err) {
		logger.Error(err, "status update failed")
	}

	// Requeue until everything is Ready so conditions converge even without a
	// child-status watch event firing.
	if !allReady {
		return ctrl.Result{RequeueAfter: componentRequeueInterval()}, nil
	}
	return ctrl.Result{}, nil
}

// componentCondition is the platform-level Available condition type for a
// single managed component or piece of operator-managed infrastructure.
func componentConditionType(label string) string {
	return label + "Available"
}

// childReadyStatus returns the child CR's Ready condition status, or Unknown
// when the child or its condition isn't observed yet, plus a message.
func childReadyStatus(conds []metav1.Condition) (metav1.ConditionStatus, string) {
	if c := meta.FindStatusCondition(conds, ConditionReady); c != nil {
		return c.Status, c.Message
	}
	return metav1.ConditionUnknown, "Ready condition not reported yet"
}

// reconcileStatusConditions polls each enabled component's workload/child CR
// readiness and writes one Available condition per component
// (Operator, NATS, Backplane, Gateway, MCP, Broker, Server, Client) plus the
// top-level Available / Ready conditions and a human phase. It returns the
// children list enriched with rolled-up readiness, and whether every enabled
// component is Ready.
func (r *ZelosPlatformReconciler) reconcileStatusConditions(
	ctx context.Context,
	platform *zelosv1alpha1.ZelosPlatform,
	children []zelosv1alpha1.ChildRef,
	collectorEnabled bool,
) ([]zelosv1alpha1.ChildRef, bool) {
	gen := platform.Generation
	ns := platform.Namespace

	// component records one managed unit's readiness for roll-up.
	type component struct {
		label     string // condition prefix, e.g. "Gateway"
		available bool
		reason    string
		message   string
		// childIdx points at the children entry to enrich, or -1 when this
		// component is operator-managed infra with no child CR.
		childIdx int
	}
	var comps []component

	// Operator-managed OTel collector ("operator" component from issue #34).
	if collectorEnabled {
		wr := deploymentReadiness(ctx, r.Client, types.NamespacedName{Name: render.TelemetryCollectorName, Namespace: ns})
		comps = append(comps, component{
			label:     "Operator",
			available: wr.Available,
			reason:    wr.Reason,
			message:   "OTel collector: " + wr.Message,
			childIdx:  -1,
		})
	}

	// Per-component child CRs: read their rolled-up Ready condition.
	for i := range children {
		child := children[i]
		conds, found := r.childConditions(ctx, child.Kind, types.NamespacedName{Name: child.Name, Namespace: ns})
		label := strings.TrimPrefix(child.Kind, "Zelos") // ZelosGateway -> Gateway
		status := metav1.ConditionUnknown
		msg := "child CR not observed yet"
		if found {
			status, msg = childReadyStatus(conds)
		}
		available := status == metav1.ConditionTrue
		reason := ReasonWorkloadUnavailable
		if available {
			reason = ReasonWorkloadAvailable
		}
		comps = append(comps, component{
			label:     label,
			available: available,
			reason:    reason,
			message:   msg,
			childIdx:  i,
		})

		// NATS is the workload behind an operator-managed backplane; surface
		// it as its own component condition per issue #34.
		if child.Kind == "ZelosBackplane" && r.backplaneIsOperatorNATS(platform) {
			wr := statefulSetReadiness(ctx, r.Client, types.NamespacedName{Name: render.NATSName(child.Name), Namespace: ns})
			comps = append(comps, component{
				label:     "NATS",
				available: wr.Available,
				reason:    wr.Reason,
				message:   "NATS StatefulSet: " + wr.Message,
				childIdx:  -1,
			})
		}
	}

	// Write per-component Available conditions + enrich children.
	enriched := append([]zelosv1alpha1.ChildRef{}, children...)
	allReady := true
	var notReady []string
	for _, comp := range comps {
		st := metav1.ConditionFalse
		if comp.available {
			st = metav1.ConditionTrue
		} else {
			allReady = false
			notReady = append(notReady, comp.label)
		}
		meta.SetStatusCondition(&platform.Status.Conditions, metav1.Condition{
			Type:               componentConditionType(comp.label),
			Status:             st,
			ObservedGeneration: gen,
			Reason:             comp.reason,
			Message:            comp.message,
		})
		if comp.childIdx >= 0 {
			enriched[comp.childIdx].Ready = string(st)
			enriched[comp.childIdx].Message = comp.message
		}
	}

	// Top-level Available + Ready + phase.
	r.writeTopLevelConditions(platform, gen, len(comps), allReady, notReady)

	// An empty platform (no enabled components) is not "Ready" — it's Pending.
	if len(comps) == 0 {
		return enriched, false
	}
	return enriched, allReady
}

// writeTopLevelConditions sets the aggregate Available/Ready conditions and the
// human phase from the per-component roll-up.
func (r *ZelosPlatformReconciler) writeTopLevelConditions(
	platform *zelosv1alpha1.ZelosPlatform,
	gen int64,
	componentCount int,
	allReady bool,
	notReady []string,
) {
	if componentCount == 0 {
		meta.SetStatusCondition(&platform.Status.Conditions, metav1.Condition{
			Type: ConditionAvailable, Status: metav1.ConditionFalse,
			ObservedGeneration: gen, Reason: ReasonNoComponents,
			Message: "No components enabled in spec",
		})
		meta.SetStatusCondition(&platform.Status.Conditions, metav1.Condition{
			Type: ConditionReady, Status: metav1.ConditionFalse,
			ObservedGeneration: gen, Reason: ReasonNoComponents,
			Message: "No components enabled in spec",
		})
		platform.Status.Phase = PhasePending
		return
	}

	if allReady {
		msg := fmt.Sprintf("All %d enabled components are Available", componentCount)
		meta.SetStatusCondition(&platform.Status.Conditions, metav1.Condition{
			Type: ConditionAvailable, Status: metav1.ConditionTrue,
			ObservedGeneration: gen, Reason: ReasonAllComponentsReady, Message: msg,
		})
		meta.SetStatusCondition(&platform.Status.Conditions, metav1.Condition{
			Type: ConditionReady, Status: metav1.ConditionTrue,
			ObservedGeneration: gen, Reason: ReasonAllComponentsReady, Message: msg,
		})
		platform.Status.Phase = PhaseReady
		return
	}

	sort.Strings(notReady)
	msg := fmt.Sprintf("Waiting on components: %s", strings.Join(notReady, ", "))
	meta.SetStatusCondition(&platform.Status.Conditions, metav1.Condition{
		Type: ConditionAvailable, Status: metav1.ConditionFalse,
		ObservedGeneration: gen, Reason: ReasonComponentsNotReady, Message: msg,
	})
	meta.SetStatusCondition(&platform.Status.Conditions, metav1.Condition{
		Type: ConditionReady, Status: metav1.ConditionFalse,
		ObservedGeneration: gen, Reason: ReasonComponentsNotReady, Message: msg,
	})
	platform.Status.Phase = PhaseProgressing
}

// backplaneIsOperatorNATS reports whether the platform's backplane resolves to
// an operator-installed NATS StatefulSet (substrate nats, no externalURL).
func (r *ZelosPlatformReconciler) backplaneIsOperatorNATS(platform *zelosv1alpha1.ZelosPlatform) bool {
	bp := platform.Spec.Backplane
	substrate := bp.Substrate
	if substrate == "" {
		substrate = "nats"
	}
	return substrate == "nats" && bp.ExternalURL == ""
}

// childConditions fetches the named child CR by Kind and returns its status
// conditions. The bool is false when the child isn't found.
func (r *ZelosPlatformReconciler) childConditions(ctx context.Context, kind string, key types.NamespacedName) ([]metav1.Condition, bool) {
	get := func(o client.Object, cs func() []metav1.Condition) ([]metav1.Condition, bool) {
		if err := r.Get(ctx, key, o); err != nil {
			return nil, false
		}
		return cs(), true
	}
	switch kind {
	case "ZelosGateway":
		var c zelosv1alpha1.ZelosGateway
		return get(&c, func() []metav1.Condition { return c.Status.Conditions })
	case "ZelosBackplane":
		var c zelosv1alpha1.ZelosBackplane
		return get(&c, func() []metav1.Condition { return c.Status.Conditions })
	case "ZelosMCP":
		var c zelosv1alpha1.ZelosMCP
		return get(&c, func() []metav1.Condition { return c.Status.Conditions })
	case "ZelosBroker":
		var c zelosv1alpha1.ZelosBroker
		return get(&c, func() []metav1.Condition { return c.Status.Conditions })
	case "ZelosServer":
		var c zelosv1alpha1.ZelosServer
		return get(&c, func() []metav1.Condition { return c.Status.Conditions })
	case "ZelosClient":
		var c zelosv1alpha1.ZelosClient
		return get(&c, func() []metav1.Condition { return c.Status.Conditions })
	default:
		return nil, false
	}
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
