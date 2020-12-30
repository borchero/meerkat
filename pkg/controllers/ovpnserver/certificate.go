package ovpnserver

import (
	api "github.com/borchero/meerkat-operator/pkg/api/v1alpha1"
	"github.com/borchero/meerkat-operator/pkg/crypto"
)

// PKIConfig returns the PKI configuration for the PKI.
func PKIConfig(server *api.OvpnServer) crypto.PKIConfig {
	return crypto.PKIConfig{
		CommonName:         server.Spec.Security.PKI.DN.DefaultedCommonName(),
		Validity:           server.Spec.Security.PKI.DefaultedValidity(),
		RSABits:            server.Spec.Security.PKI.DefaultedRSABits(),
		Organization:       server.Spec.Security.PKI.DN.Organization,
		OrganizationalUnit: server.Spec.Security.PKI.DN.OrganizationalUnit,
		Country:            server.Spec.Security.PKI.DN.Country,
		Locality:           server.Spec.Security.PKI.DN.Locality,
	}
}

// PKIServerConfig returns the PKI configuration for the server component.
func PKIServerConfig(server *api.OvpnServer) crypto.PKIRoleConfig {
	return crypto.PKIRoleConfig{
		DefaultValidity: server.Spec.Security.Server.DefaultedValidity(),
		RSABits:         server.Spec.Security.Server.DefaultedRSABits(),
		Server:          true,
	}
}

// PKIClientConfig returns the PKI configuration for the client component.
func PKIClientConfig(server *api.OvpnServer) crypto.PKIRoleConfig {
	return crypto.PKIRoleConfig{
		DefaultValidity: server.Spec.Security.Clients.DefaultedValidity(),
		RSABits:         server.Spec.Security.Clients.DefaultedRSABits(),
		Server:          true,
	}
}
