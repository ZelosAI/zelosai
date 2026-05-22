package render

import (
	"fmt"
	"path"

	zelosv1alpha1 "github.com/ZelosAI/zelosai/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// SecretVolumes builds the Volume / VolumeMount / EnvVar triplet required to
// project each SecretMount into the pod as a file under /etc/zelos/secrets/
// and expose its absolute path via env (default: ZELOS<COMPONENT>_<KEY>_FILE).
//
// One Volume is created per Secret name. Each key in the same Secret becomes
// a separate VolumeMount with subPath, so multiple keys from one Secret share
// a single Volume.
func SecretVolumes(c Component, refs []zelosv1alpha1.SecretMount) ([]corev1.Volume, []corev1.VolumeMount, []corev1.EnvVar) {
	if len(refs) == 0 {
		return nil, nil, nil
	}
	seen := map[string]bool{}
	var volumes []corev1.Volume
	var mounts []corev1.VolumeMount
	var envs []corev1.EnvVar

	for _, ref := range refs {
		volName := fmt.Sprintf("secret-%s", ref.Name)
		if !seen[ref.Name] {
			volumes = append(volumes, corev1.Volume{
				Name: volName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{SecretName: ref.Name},
				},
			})
			seen[ref.Name] = true
		}
		mountPath := ref.Path
		if mountPath == "" {
			mountPath = path.Join(c.SecretsDir, ref.Key)
		}
		mounts = append(mounts, corev1.VolumeMount{
			Name:      volName,
			MountPath: mountPath,
			SubPath:   ref.Key,
			ReadOnly:  true,
		})
		envName := ref.Env
		if envName == "" {
			envName = FileEnvName(c, ref.Key)
		}
		envs = append(envs, corev1.EnvVar{Name: envName, Value: mountPath})
	}
	return volumes, mounts, envs
}
