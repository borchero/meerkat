package ovpn

// ConfigValues describes the set of values required to render the OVPN config file.
type ConfigValues struct {
	Files       ConfigFiles
	Routes      []ConfigRoute
	Nameservers []string
	RedirectAll bool
	Protocol    string
	Security    ConfigSecurity
}

// ConfigFiles describes the set of file paths required for the OVPN config file.
type ConfigFiles struct {
	TLSServerCrt string
	TLSServerKey string
	TLSCaCrt     string
	DHParams     string
	TLSAuth      string
	CRL          string
}

// ConfigRoute describes a route for the OVPN config file, consisting of IP and subnet mask.
type ConfigRoute struct {
	IP   string
	Mask string
}

// ConfigSecurity describe the security configuration for the OVPN config file.
type ConfigSecurity struct {
	Hmac   string
	Cipher string
}

// GetConfig returns a OVPN config for the given files and configuration.
func GetConfig(values ConfigValues) (string, error) {
	return renderTemplate("config", "/cmd/templates/openvpn.conf.tpl", values)
}
