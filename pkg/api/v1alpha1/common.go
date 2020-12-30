package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OvpnCertificateConfig describes common properties of certificate configurations.
type OvpnCertificateConfig struct {
	// The duration for which the certificate is valid. Defaults to 10 years for the root key,
	// 90 days for the server and 2 years for client.
	Validity metav1.Duration `json:"validity,omitempty"`
	// The number of bits to use for the root RSA key. Changing this value for existing keys (such
	// as the root key) has no effect.
	// +kubebuilder:default=4096
	// +kubebuilder:validation:Enum=2048;4096;8192
	RSABits int `json:"rsaBits,omitempty"`
}

// OvpnPKICertificateConfig describes the certificate configuration of a PKI.
type OvpnPKICertificateConfig struct {
	OvpnCertificateConfig `json:",inline"`
}

// OvpnServerCertificateConfig describes the certificate configuration of a server.
type OvpnServerCertificateConfig struct {
	OvpnCertificateConfig `json:",inline"`
}

// OvpnClientCertificateConfig describes the certificate configuration of a client.
type OvpnClientCertificateConfig struct {
	OvpnCertificateConfig `json:",inline"`
}
