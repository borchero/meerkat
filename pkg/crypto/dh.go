package crypto

import (
	"fmt"
	"os/exec"
)

// GenerateDhParams generates Diffie-Hellman parameters of the given size and returns the generated
// parameters upon success. This method may take multiple minutes to run.
func GenerateDhParams(bits int) ([]byte, error) {
	cmd := exec.Command("openssl", "dhparam", "-out", "/dev/stdout", fmt.Sprintf("%d", bits))
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed generating dh params: %s", err)
	}
	return out, nil
}
