package crypto

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
)

// PKI provides a proxy to a Vault instance to manage a PKI.
type PKI struct {
	client *vaultapi.Client
	path   string
}

// PKICertificate describes a certificate obtained from a PKI.
type PKICertificate struct {
	Serial        string
	Certificate   string
	PrivateKey    string
	CACertificate string
	Expiration    time.Time
}

// PKICrl represents a certificate revocation list includign non-expired revoked certificates.
type PKICrl struct {
	Certificate string
}

// PKIConfig describes the configuration of a PKI root certificates. Fields that are not set
// explicitly are not added to the root certificate. Common name, rsa bits and validity must be
// set.
type PKIConfig struct {
	CommonName         string
	Validity           time.Duration
	RSABits            int
	Organization       string
	OrganizationalUnit string
	Country            string
	Locality           string
}

// PKIRoleConfig describes the configuration of a role.
type PKIRoleConfig struct {
	DefaultValidity time.Duration
	RSABits         int
	Server          bool
}

// NewPKI returns a new PKI at the given path. Possibly, the PKI is not yet initialized.
func NewPKI(vault *vaultapi.Client, path string) *PKI {
	return &PKI{client: vault, path: path}
}

// EnsureEnabled makes sure that the PKI is enabled at the given path.
func (pki *PKI) EnsureEnabled() error {
	mounts, err := pki.client.Sys().ListMounts()
	if err != nil {
		return fmt.Errorf("failed to list existing mount paths: %s", err)
	}

	// If the mounts contain the path, it is already enabled
	if _, ok := mounts[pki.path+"/"]; ok {
		return nil
	}

	// Otherwise, we create it
	input := &vaultapi.MountInput{
		Type: "pki",
		Config: vaultapi.MountConfigInput{
			DefaultLeaseTTL: "2592000",   // 30 days
			MaxLeaseTTL:     "315360000", // 10 years
		},
	}
	if err := pki.client.Sys().Mount(pki.path, input); err != nil {
		return fmt.Errorf("failed to create mount for PKI: %s", err)
	}

	// Also, we need to configure the CRL
	path := fmt.Sprintf("%s/config/crl", pki.path)
	content := map[string]interface{}{
		"expiry":  "72h",
		"disable": false,
	}
	if _, err := pki.client.Logical().Write(path, content); err != nil {
		return fmt.Errorf("failed to set CRL configuration: %s", err)
	}
	return nil
}

// DisableIfEnabled disables the engine backing the PKI if it exists.
func (pki *PKI) DisableIfEnabled() error {
	if err := pki.client.Sys().Unmount(pki.path); err != nil {
		return fmt.Errorf("failed to disable PKI: %s", err)
	}
	return nil
}

// GenerateRootIfRequired generates the internal private key and certificate of the PKI or does
// nothing if it already exists.
func (pki *PKI) GenerateRootIfRequired(config PKIConfig) error {
	path := fmt.Sprintf("%s/root/generate/internal", pki.path)
	contents := map[string]interface{}{
		"common_name":          config.CommonName,
		"key_type":             "rsa",
		"key_bits":             config.RSABits,
		"ttl":                  fmt.Sprintf("%ds", int(config.Validity.Seconds())),
		"exclude_cn_from_sans": true,
	}
	if config.Organization != "" {
		contents["organization"] = config.Organization
	}
	if config.OrganizationalUnit != "" {
		contents["ou"] = config.OrganizationalUnit
	}
	if config.Country != "" {
		contents["country"] = config.Country
	}
	if config.Locality != "" {
		contents["locality"] = config.Locality
	}
	if _, err := pki.client.Logical().Write(path, contents); err != nil {
		return fmt.Errorf("failed to verify and possibly create root key: %s", err)
	}
	return nil
}

// ConfigureRole configures the role with the given name. If the role doesn't exist yet, it is
// created, otherwise it is updated.
func (pki *PKI) ConfigureRole(name string, config PKIRoleConfig) error {
	var extensions []string
	if config.Server {
		extensions = []string{"TLS Web Server Authentication"}
	} else {
		extensions = []string{"TLS Web Client Authentication"}
	}

	path := fmt.Sprintf("%s/roles/%s", pki.path, name)
	contents := map[string]interface{}{
		"key_type":            "rsa",
		"key_bits":            config.RSABits,
		"ttl":                 fmt.Sprintf("%ds", int(config.DefaultValidity.Seconds())),
		"max_ttl":             "87600h",
		"allow_any_name":      true,
		"server_flag":         config.Server,
		"client_flag":         !config.Server,
		"generate_lease":      false,
		"not_before_duration": "15m",
		"key_usage":           []string{"DigitalSignature", "KeyAgreement", "KeyEncipherment"},
		"ext_key_usage":       extensions,
	}
	if _, err := pki.client.Logical().Write(path, contents); err != nil {
		return fmt.Errorf("failed to update role configuration: %s", err)
	}
	return nil
}

// Generate generates a new certificate for the provided role with the given common name. If the
// validity is greater than 0, it replaces the default validity.
func (pki *PKI) Generate(role, commonName string, validity time.Duration) (PKICertificate, error) {
	path := fmt.Sprintf("%s/issue/%s", pki.path, role)
	contents := map[string]interface{}{
		"common_name": commonName,
		"format":      "pem",
	}
	if validity > 0 {
		contents["ttl"] = fmt.Sprintf("%ds", int(validity.Seconds()))
	}
	result, err := pki.client.Logical().Write(path, contents)
	if err != nil {
		return PKICertificate{}, fmt.Errorf("failed to generate certificate: %s", err)
	}
	expiration, err := result.Data["expiration"].(json.Number).Int64()
	if err != nil {
		return PKICertificate{}, fmt.Errorf("invalid expiration date: %s", err)
	}

	return PKICertificate{
		Serial:        result.Data["serial_number"].(string),
		Certificate:   result.Data["certificate"].(string),
		PrivateKey:    result.Data["private_key"].(string),
		CACertificate: result.Data["issuing_ca"].(string),
		Expiration:    time.Unix(expiration, 0),
	}, nil
}

// Revoke revokes the certificate with the given serial.
func (pki *PKI) Revoke(serial string) error {
	path := fmt.Sprintf("%s/revoke", pki.path)
	contents := map[string]interface{}{
		"serial_number": serial,
	}
	if _, err := pki.client.Logical().Write(path, contents); err != nil {
		return fmt.Errorf("failed to revoke certificate: %s", err)
	}
	return nil
}

// GetCRL returns the revocation list for this PKI. The CRL is automatically rotated if its
// expiration date is within the next 24 hours.
func (pki *PKI) GetCRL() (PKICrl, error) {
	// First, we get the certificate and check whether it expires soon
	crl, err := pki.readCRL()
	if err != nil {
		return PKICrl{}, err
	}
	expiration, err := pki.crlExpiration(crl)
	if err != nil {
		return PKICrl{}, err
	}
	if expiration.Sub(time.Now()) >= 24*time.Hour {
		return PKICrl{Certificate: crl}, nil
	}

	// Otherwise, we rotate it
	rotationPath := fmt.Sprintf("%s/crl/rotate", pki.path)
	if _, err := pki.client.Logical().Read(rotationPath); err != nil {
		return PKICrl{}, fmt.Errorf("failed to rotate CRL: %s", err)
	}

	// And then we can request it again
	crl, err = pki.readCRL()
	if err != nil {
		return PKICrl{}, err
	}
	return PKICrl{Certificate: crl}, nil
}

func (pki *PKI) readCRL() (string, error) {
	path := fmt.Sprintf("%s/cert/crl", pki.path)
	result, err := pki.client.Logical().Read(path)
	if err != nil {
		return "", fmt.Errorf("failed to read CRL: %s", err)
	}
	return result.Data["certificate"].(string), nil
}

func (pki *PKI) crlExpiration(crl string) (time.Time, error) {
	cert, err := x509.ParseCRL([]byte(crl))
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse CRL: %s", err)
	}
	return cert.TBSCertList.NextUpdate, nil
}
