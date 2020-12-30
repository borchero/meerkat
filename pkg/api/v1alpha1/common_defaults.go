package v1alpha1

import "time"

// DefaultedRSABits returns the number of RSA bits with a default of 4096.
func (c OvpnCertificateConfig) DefaultedRSABits() int {
	if c.RSABits == 0 {
		return 4096
	}
	return c.RSABits
}

// DefaultedValidity returns the validity of the certificate with a default value of 10 years.
func (c OvpnPKICertificateConfig) DefaultedValidity() time.Duration {
	if c.OvpnCertificateConfig.Validity.Duration == 0 {
		return 87600 * time.Hour
	}
	return c.OvpnCertificateConfig.Validity.Duration
}

// DefaultedValidity returns the validity of the certificate with a default value of 90 days.
func (c OvpnServerCertificateConfig) DefaultedValidity() time.Duration {
	if c.OvpnCertificateConfig.Validity.Duration == 0 {
		return 2160 * time.Hour
	}
	return c.OvpnCertificateConfig.Validity.Duration
}

// DefaultedValidity returns the validity of the certificate with a default value of 2 years.
func (c OvpnClientCertificateConfig) DefaultedValidity() time.Duration {
	if c.OvpnCertificateConfig.Validity.Duration == 0 {
		return 17520 * time.Hour
	}
	return c.OvpnCertificateConfig.Validity.Duration
}
