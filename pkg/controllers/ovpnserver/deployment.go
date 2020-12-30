package ovpnserver

import (
	api "github.com/borchero/meerkat-operator/pkg/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	volumeNameConfig        = "config"
	volumeNameEntrypoint    = "entrypoint"
	volumeNameTLSKeys       = "tls-keys"
	volumeNameSharedSecrets = "shared-secrets"
	volumeNameCrl           = "crl"

	// MountPathOpenVpnConfig is the mount path of the VPN config.
	MountPathOpenVpnConfig = "/etc/openvpn"
	// MountPathEntrypoint is the mount path of the server entrypoint.
	MountPathEntrypoint = "/app"
	// MountPathTLSKeys is the mount path of the server TLS keys.
	MountPathTLSKeys = "/secrets/tls"
	// MountPathSharedSecrets is the mount path of the server shared secrets.
	MountPathSharedSecrets = "/secrets/shared"
	// MountPathCrl is the mount for the PKI CRL.
	MountPathCrl = "/secrets/crl"

	selectorKey = "app.kubernetes.io/name"
)

// GetDeploymentSpec returns the expected deployment spec for the given server and the provided
// container image.
func GetDeploymentSpec(
	server *api.OvpnServer, image string, podAnnotations map[string]string,
) appsv1.DeploymentSpec {
	var replicas int32 = 1
	var progressDeadline int32 = 600
	var revisionLimit int32 = 10
	return appsv1.DeploymentSpec{
		Replicas:                &replicas,
		ProgressDeadlineSeconds: &progressDeadline,
		RevisionHistoryLimit:    &revisionLimit,
		Strategy: appsv1.DeploymentStrategy{
			Type: appsv1.RecreateDeploymentStrategyType,
		},
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				selectorKey: server.ObjectRefDeployment().Name,
			},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					selectorKey: server.ObjectRefDeployment().Name,
				},
				Annotations: podAnnotations,
			},
			Spec: getPodSpec(server, image),
		},
	}
}

func getPodSpec(server *api.OvpnServer, image string) corev1.PodSpec {
	var gracePeriod int64 = 30
	return corev1.PodSpec{
		Containers: []corev1.Container{{
			Name:            "openvpn",
			Image:           image,
			ImagePullPolicy: corev1.PullIfNotPresent,
			SecurityContext: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Add: []corev1.Capability{corev1.Capability("NET_ADMIN")},
				},
			},
			VolumeMounts:             getVolumeMounts(server),
			Resources:                corev1.ResourceRequirements{},
			TerminationMessagePath:   "/dev/termination-log",
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
		}},
		Volumes:                       getVolumes(server),
		RestartPolicy:                 corev1.RestartPolicyAlways,
		DNSPolicy:                     corev1.DNSClusterFirst,
		SchedulerName:                 corev1.DefaultSchedulerName,
		TerminationGracePeriodSeconds: &gracePeriod,
		SecurityContext:               &corev1.PodSecurityContext{},
	}
}

func getVolumeMounts(server *api.OvpnServer) []corev1.VolumeMount {
	return []corev1.VolumeMount{{
		Name:      volumeNameConfig,
		MountPath: MountPathOpenVpnConfig,
	}, {
		Name:      volumeNameEntrypoint,
		MountPath: MountPathEntrypoint,
	}, {
		Name:      volumeNameTLSKeys,
		MountPath: MountPathTLSKeys,
	}, {
		Name:      volumeNameSharedSecrets,
		MountPath: MountPathSharedSecrets,
	}, {
		Name:      volumeNameCrl,
		MountPath: MountPathCrl,
	}}
}

func getVolumes(server *api.OvpnServer) []corev1.Volume {
	var execMode int32 = 0775
	var readMode int32 = 0644
	return []corev1.Volume{{
		Name: volumeNameConfig,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: server.ObjectRefOvpnConfigMap().Name,
				},
				DefaultMode: &readMode,
			},
		},
	}, {
		Name: volumeNameEntrypoint,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: server.ObjectRefEntrypointConfigMap().Name,
				},
				DefaultMode: &execMode,
			},
		},
	}, {
		Name: volumeNameTLSKeys,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName:  server.ObjectRefServerCertificateSecret().Name,
				DefaultMode: &readMode,
			},
		},
	}, {
		Name: volumeNameSharedSecrets,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName:  server.ObjectRefSharedSecrets().Name,
				DefaultMode: &readMode,
			},
		},
	}, {
		Name: volumeNameCrl,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName:  server.ObjectRefCrlSecret().Name,
				DefaultMode: &readMode,
			},
		},
	}}
}
