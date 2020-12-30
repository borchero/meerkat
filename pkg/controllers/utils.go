package controllers

// Config describes global configuration for all reconcilers.
type Config struct {
	// The reference to the image to use for the OVPN server.
	Image string
	// The base path to use within Vault for mounting PKIs for the OVPN servers.
	PKIPath string `split_words:"true"`
}
