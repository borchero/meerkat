package v1alpha1

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObjectRefSharedSecrets returns the reference to the shared secret.
func (s OvpnServer) ObjectRefSharedSecrets() metav1.ObjectMeta {
	ref := metav1.ObjectMeta{
		Name:      s.Spec.Secrets.SharedSecretName,
		Namespace: s.Namespace,
	}
	if ref.Name == "" {
		ref.Name = fmt.Sprintf("%s-shared-secret", s.Name)
	}
	return ref
}

// ObjectRefCrlSecret returns a reference to the secret containing the CRL.
func (s *OvpnServer) ObjectRefCrlSecret() metav1.ObjectMeta {
	ref := metav1.ObjectMeta{
		Name:      s.Spec.Secrets.CrlName,
		Namespace: s.Namespace,
	}
	if ref.Name == "" {
		ref.Name = fmt.Sprintf("%s-crl", s.Name)
	}
	return ref
}

// ObjectRefServerCertificateSecret returns a reference to the secret containing the server
// certificate.
func (s *OvpnServer) ObjectRefServerCertificateSecret() metav1.ObjectMeta {
	ref := metav1.ObjectMeta{
		Name:      s.Spec.Secrets.ServerCertificateName,
		Namespace: s.Namespace,
	}
	if ref.Name == "" {
		ref.Name = fmt.Sprintf("%s-server-certificate", s.Name)
	}
	return ref
}

// ObjectRefOvpnConfigMap returns a reference to the server configmap.
func (s *OvpnServer) ObjectRefOvpnConfigMap() metav1.ObjectMeta {
	ref := metav1.ObjectMeta{
		Name:      s.Spec.Deployment.OvpnConfigMapName,
		Namespace: s.Namespace,
	}
	if ref.Name == "" {
		ref.Name = fmt.Sprintf("%s-config", s.Name)
	}
	return ref
}

// ObjectRefEntrypointConfigMap returns a reference to the server entrypoint configmap.
func (s *OvpnServer) ObjectRefEntrypointConfigMap() metav1.ObjectMeta {
	ref := metav1.ObjectMeta{
		Name:      s.Spec.Deployment.EntrypointConfigMapName,
		Namespace: s.Namespace,
	}
	if ref.Name == "" {
		ref.Name = fmt.Sprintf("%s-entrypoint", s.Name)
	}
	return ref
}

// ObjectRefDeployment returns a reference to the deployment.
func (s *OvpnServer) ObjectRefDeployment() metav1.ObjectMeta {
	ref := metav1.ObjectMeta{
		Name:      s.Spec.Deployment.Name,
		Namespace: s.Namespace,
	}
	if ref.Name == "" {
		ref.Name = s.Name
	}
	return ref
}

// ObjectRefService returns a reference to the service exposing the VPN.
func (s *OvpnServer) ObjectRefService() metav1.ObjectMeta {
	ref := metav1.ObjectMeta{
		Name:      s.Spec.Service.Name,
		Namespace: s.Namespace,
	}
	if ref.Name == "" {
		ref.Name = s.Name
	}
	return ref
}

//-------------------------------------------------------------------------------------------------

// DefaultedProtocol returns the provided protocol or UDP if none is provided.
func (a OvpnServerAddress) DefaultedProtocol() corev1.Protocol {
	if a.Protocol == "" {
		return corev1.ProtocolUDP
	}
	return a.Protocol
}

// DefaultedNameservers returns the provided nameservers or standard Google nameservers otherwise.
func (c OvpnTrafficConfig) DefaultedNameservers() []string {
	if c.Nameservers == nil || len(c.Nameservers) == 0 {
		return []string{"8.8.4.4", "8.8.8.8"}
	}
	result := []string{}
	for _, ip := range c.Nameservers {
		result = append(result, string(ip))
	}
	return result
}

// DefaultedHmac returns the provided Hmac or SHA-384.
func (c OvpnSecurityConfig) DefaultedHmac() Hmac {
	if c.Hmac == "" {
		return HmacSHA384
	}
	return c.Hmac
}

// DefaultedCipher returns the provided cipher or AES-256-GCM.
func (c OvpnSecurityConfig) DefaultedCipher() Cipher {
	if c.Cipher == "" {
		return CipherAES256GCM
	}
	return c.Cipher
}

// DefaultedCommonName returns a default PKI common name if it is not defined.
func (c OvpnPkiDnConfig) DefaultedCommonName() string {
	if c.CommonName == "" {
		return "ovpn-pki"
	}
	return c.CommonName
}

// DefaultedPort returns the port of the service.
func (s OvpnServerService) DefaultedPort() uint16 {
	if s.Port == 0 {
		return 1194
	}
	return s.Port
}

// DefaultedServiceType returns the service type of the service.
func (s OvpnServerService) DefaultedServiceType() corev1.ServiceType {
	if s.ServiceType == "" {
		return corev1.ServiceTypeLoadBalancer
	}
	return s.ServiceType
}
