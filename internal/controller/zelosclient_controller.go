package controller

import (
	"context"
	"fmt"

	zelosv1alpha1 "github.com/ZelosAI/zelosai/api/v1alpha1"
	"github.com/ZelosAI/zelosai/internal/controller/render"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ZelosClientReconciler handles the optional in-cluster DaemonSet for ZelosClient.
// Default deployment is off-cluster (Ansible via zelos.dgx); this controller only
// reconciles when spec.inCluster=true.
type ZelosClientReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=zelos.zelosai.io,resources=zelosclients,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=zelos.zelosai.io,resources=zelosclients/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete

func (r *ZelosClientReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var cl zelosv1alpha1.ZelosClient
	if err := r.Get(ctx, req.NamespacedName, &cl); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Off-cluster default: nothing for the operator to do.
	if !cl.Spec.InCluster {
		cl.Status.ObservedGeneration = cl.Generation
		_ = r.Status().Update(ctx, &cl)
		return ctrl.Result{}, nil
	}

	c := render.Defaults("zelosclient")

	cfg := map[string]string{}
	for k, v := range cl.Spec.Config {
		cfg[k] = v
	}
	if cl.Spec.Runtime != "" {
		cfg["ZELOSCLIENT_RUNTIME"] = cl.Spec.Runtime
	}
	if cl.Spec.RuntimeURL != "" {
		cfg["ZELOSCLIENT_RUNTIME_URL"] = cl.Spec.RuntimeURL
	}
	if cl.Spec.Model != "" {
		cfg["ZELOSCLIENT_MODEL"] = cl.Spec.Model
	}
	if cl.Spec.BackplaneRef != "" {
		cfg["ZELOSBACKPLANE_URL"] = render.NATSURL(cl.Spec.BackplaneRef, cl.Namespace)
	}

	cm := render.BuildConfigMap(&cl, c, cfg)
	if err := applyObject(ctx, r.Client, r.Scheme, &cl, cm); err != nil {
		return ctrl.Result{}, err
	}

	ds := buildClientDaemonSet(&cl, c, cm.Name)
	if err := applyObject(ctx, r.Client, r.Scheme, &cl, ds); err != nil {
		return ctrl.Result{}, err
	}

	cl.Status.ObservedGeneration = cl.Generation
	_ = r.Status().Update(ctx, &cl)
	return ctrl.Result{}, nil
}

func (r *ZelosClientReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&zelosv1alpha1.ZelosClient{}).
		Complete(r)
}

func buildClientDaemonSet(cl *zelosv1alpha1.ZelosClient, c render.Component, cmName string) *appsv1.DaemonSet {
	labels := render.Labels(c, cl.Name)
	selector := render.SelectorLabels(c, cl.Name)

	image := c.DefaultImageRepo
	if cl.Spec.Image.Repository != "" {
		image = cl.Spec.Image.Repository
	}
	tag := "develop"
	if cl.Spec.Image.Tag != "" {
		tag = cl.Spec.Image.Tag
	}

	res := cl.Spec.Resources
	if res.Requests == nil {
		res.Requests = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		}
	}

	pullSecrets := append([]corev1.LocalObjectReference{}, cl.Spec.ImagePullSecrets...)
	pullSecrets = append(pullSecrets, corev1.LocalObjectReference{Name: "ghcr-pull-secret"})

	envFrom := []corev1.EnvFromSource{
		{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: render.TelemetryEnvConfigMapName(cl.Name)}}},
		{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: cmName}}},
	}

	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", c.Name, cl.Name),
			Namespace: cl.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: selector},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					ServiceAccountName: fmt.Sprintf("zelos-%s", c.Name),
					ImagePullSecrets:   pullSecrets,
					NodeSelector:       cl.Spec.NodeSelector,
					Tolerations:        cl.Spec.Tolerations,
					Containers: []corev1.Container{{
						Name:    c.Name,
						Image:   render.ImageRef(image, tag),
						EnvFrom: envFrom,
						Env: []corev1.EnvVar{
							{Name: "OTEL_SERVICE_NAME", Value: c.Name},
							{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}},
							{Name: "POD_NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
						},
						Ports: []corev1.ContainerPort{
							{Name: "admin", ContainerPort: c.Port, Protocol: corev1.ProtocolTCP},
						},
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{Path: "/healthz", Port: intstr.FromString("admin")},
							},
							InitialDelaySeconds: 5,
							PeriodSeconds:       10,
						},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{Path: "/readyz", Port: intstr.FromString("admin")},
							},
							InitialDelaySeconds: 3,
							PeriodSeconds:       5,
						},
						Resources: res,
						SecurityContext: &corev1.SecurityContext{
							AllowPrivilegeEscalation: boolPtrCl(false),
							ReadOnlyRootFilesystem:   boolPtrCl(true),
							RunAsNonRoot:             boolPtrCl(true),
							Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
						},
					}},
				},
			},
		},
	}
}

func boolPtrCl(b bool) *bool { return &b }
