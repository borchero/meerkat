package crypto

import (
	"fmt"
	"os/exec"
)

// GenerateTLSAuth generates an OpenVPN static key to be used.
func GenerateTLSAuth() ([]byte, error) {
	cmd := exec.Command("openvpn", "--genkey", "--secret", "/dev/stdout")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed generating tls auth: %s", err)
	}
	return out, nil
}
