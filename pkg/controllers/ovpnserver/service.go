package ovpnserver

import (
	api "github.com/borchero/meerkat-operator/pkg/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// GetServiceSpec returns the expected service spec for the given server.
func GetServiceSpec(server *api.OvpnServer) corev1.ServiceSpec {
	var nodePort int32
	if server.Spec.Service.DefaultedServiceType() == corev1.ServiceTypeNodePort {
		nodePort = int32(server.Spec.Service.DefaultedPort())
	}
	return corev1.ServiceSpec{
		Type: server.Spec.Service.DefaultedServiceType(),
		Selector: map[string]string{
			selectorKey: server.ObjectRefDeployment().Name,
		},
		Ports: []corev1.ServicePort{{
			Name:       "ovpn",
			Protocol:   server.Spec.Network.DefaultedProtocol(),
			Port:       int32(server.Spec.Service.DefaultedPort()),
			TargetPort: intstr.FromInt(1194),
			NodePort:   nodePort,
		}},
	}
}
