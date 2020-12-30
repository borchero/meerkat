package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&OvpnServer{}, &OvpnServerList{})
}

// OvpnServer defines the schema for the OVPN server.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type OvpnServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OvpnServerSpec   `json:"spec"`
	Status OvpnServerStatus `json:"status,omitempty"`
}

// OvpnServerList defines the schema for a list of OVPN servers.
// +kubebuilder:object:root=true
type OvpnServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []OvpnServer `json:"items"`
}

//-------------------------------------------------------------------------------------------------

// ServiceType defines the available types of Kubernetes service.
// +kubebuilder:validation:Enum=LoadBalancer;NodePort
type ServiceType string

// SubnetMask defines an IPv4 range in the form <ip>/<bits>.
// +kubebuilder:validation:Pattern="^(?:(?:25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9])\\.){3}(?:25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9])\\/((3[0-2])|([1-2][0-9])|[1-9])$"
type SubnetMask string

// IPv4Address defines an IPv4 address.
// +kubebuilder:validation:Format=ipv4
type IPv4Address string

// Hmac defines a message digest algorithm.
// +kubebuilder:validation:Enum=SHA-384
type Hmac string

// Cipher defines the TLS cipher to use.
// +kubebuilder:validation:Enum=AES-256-GCM
type Cipher string

const (
	// ServiceTypeLoadBalancer uses an external load balancer as entrypoint.
	ServiceTypeLoadBalancer ServiceType = "LoadBalancer"
	// ServiceTypeNodePort uses a port in the range 30000-32767 to expose the service.
	ServiceTypeNodePort ServiceType = "NodePort"

	// HmacSHA384 defines the SHA-384 message digest algorithm.
	HmacSHA384 Hmac = "SHA-384"

	// CipherAES256GCM defines the AES-256-GCM cipher.
	CipherAES256GCM Cipher = "AES-256-GCM"
)

//-------------------------------------------------------------------------------------------------

// OvpnServerSpec defines the configuration of an OVPN server.
type OvpnServerSpec struct {
	// The network configuration of the VPN server.
	Network OvpnServerAddress `json:"network"`
	// The traffic configuration of the VPN server.
	Traffic OvpnTrafficConfig `json:"traffic,omitempty"`
	// The security configuration of the VPN server.
	Security OvpnSecurityConfig `json:"security,omitempty"`
	// The secrets used by the server.
	Secrets OvpnServerSecrets `json:"secrets,omitempty"`
	// The deployment configuration of the VPN server.
	Deployment OvpnServerDeployment `json:"deployment,omitempty"`
	// The service configuration for the VPN server.
	Service OvpnServerService `json:"service,omitempty"`
}

// OvpnServerAddress describes how the OVPN server may be reached.
type OvpnServerAddress struct {
	// The host where the server is reachable at. Will also be used as the common name of the
	// server certificate.
	Host string `json:"host"`
	// The protocol used for the OVPN server.
	// +kubebuilder:default=UDP
	// +kubebuilder:validation:Enum=TCP;UDP
	Protocol corev1.Protocol `json:"protocol,omitempty"`
}

// OvpnTrafficConfig defines the configuration of how traffic flows through the VPN.
type OvpnTrafficConfig struct {
	// Whether all traffic should be routed through the VPN.
	// +kubebuilder:default=false
	RedirectAll bool `json:"redirectAll,omitempty"`
	// Defines a list of (target) IP ranges for which traffic is routed through the VPN. Ignored if
	// `redirectAll` is set.
	Routes []SubnetMask `json:"routes,omitempty"`
	// Defines a list of nameservers to use for name resolution.
	// +kubebuilder:default={"8.8.4.4","8.8.8.8"}
	Nameservers []IPv4Address `json:"nameservers,omitempty"`
}

// OvpnSecurityConfig encapsulates security configuration of the OVPN server.
type OvpnSecurityConfig struct {
	// The message digest algorithm to use.
	// +kubebuilder:default=SHA-384
	Hmac Hmac `json:"hmac,omitempty"`
	// The TLS cipher to use.
	// +kubebuilder:default=AES-256-GCM
	Cipher Cipher `json:"cipher,omitempty"`
	// The number of bits to use for the Diffie Hellman parameters.
	// +kubebuilder:default=2048
	// +kubebuilder:validation:Enum=1024;2048;4096
	DiffieHellmanBits int `json:"diffieHellmanBits,omitempty"`
	// The configuration of the PKI.
	PKI OvpnPkiConfig `json:"pki,omitempty"`
	// The configuration for the server certificates.
	Server OvpnServerCertificateConfig `json:"server,omitempty"`
	// The default configuration for the client certificates.
	Clients OvpnClientCertificateConfig `json:"clients,omitempty"`
}

// OvpnPkiConfig describes the how the PKI of the OVPN server should be constructed.
type OvpnPkiConfig struct {
	OvpnPKICertificateConfig `json:",inline"`
	// The configuration for the distinguished name.
	DN OvpnPkiDnConfig `json:"dn,omitempty"`
}

// OvpnPkiDnConfig describes the configuration of the distinguished name.
type OvpnPkiDnConfig struct {
	// The common name for the PKI.
	CommonName string `json:"commonName,omitempty"`
	// The name of the organization.
	Organization string `json:"organization,omitempty"`
	// The unit within the defined organization.
	OrganizationalUnit string `json:"organizationalUnit,omitempty"`
	// The country code.
	Country string `json:"country,omitempty"`
	// The location of the organization within the country.
	Locality string `json:"locality,omitempty"`
}

// OvpnServerSecrets describes the secrets that are stored in the cluster for an OVPN server.
type OvpnServerSecrets struct {
	// The name for the secret to use for shared secrets (DH params and TLS auth). The default is
	// `<servername>-shared-secrets`.
	SharedSecretName string `json:"sharedSecretName,omitempty"`
	// The name of the secret to use for the certificate used by the server. Defaults to
	// `<servername>-server-certificate`.
	ServerCertificateName string `json:"serverCertificateName,omitempty"`
	// The name of the secret containing the CRL. Defaults to `<servername>-crl`.
	CrlName string `json:"crlName,omitempty"`
}

// OvpnServerDeployment describes the deployment configuration of the server.
type OvpnServerDeployment struct {
	// The name of the deployment. Defaults to the name of the server.
	Name string `json:"name,omitempty"`
	// Custom annotations to set on the deployment.
	Annotations map[string]string `json:"annotations,omitempty"`
	// Custom annotations to set on the pod.
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`
	// The name of the configmap to be used for the OpenVPN config. Defaults to
	// `<servername>-config`.
	OvpnConfigMapName string `json:"ovpnConfigMapName,omitempty"`
	// The name of the configmap to carry the OpenVPN setup. Defaults to `<servername>-entrypoint`.
	EntrypointConfigMapName string `json:"entrypointConfigMapName,omitempty"`
}

// OvpnServerService describes the service configuration of the OVPN server.
type OvpnServerService struct {
	// The name of the service. Defaults to the name of the server.
	Name string `json:"name,omitempty"`
	// Custom annotations to set on the service.
	Annotations map[string]string `json:"annotations,omitempty"`
	// The port that the server should be running on. For `serviceType` set to `NodePort`, this
	// value must be in the range [30000, 32767].
	// +kubebuilder:default=1194
	Port uint16 `json:"port,omitempty"`
	// The type for the Kubernetes servce.
	// +kubebuilder:default=LoadBalancer
	// +kubebuilder:validation:Enum=LoadBalancer;NodePort
	ServiceType corev1.ServiceType `json:"serviceType,omitempty"`
}

//-------------------------------------------------------------------------------------------------

// OvpnServerStatus describes the status of an OVPN server.
type OvpnServerStatus struct {
}
