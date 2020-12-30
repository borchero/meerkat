package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&OvpnClient{}, &OvpnClientList{})
}

// OvpnClient defines the schema for an OVPN client.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type OvpnClient struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OvpnClientSpec   `json:"spec"`
	Status OvpnClientStatus `json:"status,omitempty"`
}

// OvpnClientList defines the schema for a list of OVPN clients.
// +kubebuilder:object:root=true
type OvpnClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []OvpnClient `json:"items"`
}

//-------------------------------------------------------------------------------------------------

// OvpnClientSpec describes an OVPN client.
type OvpnClientSpec struct {
	// The name of the OvpnServer the client is associated with. The server must be in the same
	// namespace as the client.
	ServerName string `json:"serverName"`
	// The common name of the user. Typically a unique identifier such as the email address.
	CommonName string `json:"commonName"`
	// The certificate configuration.
	Certificate OvpnClientCertificate `json:"certificate,omitempty"`
}

// OvpnClientCertificate describe the configuration of a OVPN client certificate.
type OvpnClientCertificate struct {
	OvpnCertificateConfig `json:",inline"`
	// The name of the secret used to store the OVPN certificate. Defaults to the name of the
	// client.
	SecretName string `json:"secretName,omitempty"`
}

//-------------------------------------------------------------------------------------------------

// OvpnClientStatus describes the status of an OVPN client.
type OvpnClientStatus struct {
}
