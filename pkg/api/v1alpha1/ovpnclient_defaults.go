package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// ObjectRefCertificateSecret returns a reference to the secret containing the OVPN certificate.
func (c *OvpnClient) ObjectRefCertificateSecret() metav1.ObjectMeta {
	ref := metav1.ObjectMeta{
		Name:      c.Spec.Certificate.SecretName,
		Namespace: c.Namespace,
	}
	if ref.Name == "" {
		ref.Name = c.Name
	}
	return ref
}
