package crypto

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

// GenerateDhParams generates Diffie-Hellman parameters of the given size and returns the generated
// parameters upon success. This method may take multiple minutes to run.
func GenerateDhParams(bits int) ([]byte, error) {
	file, err := ioutil.TempFile("", "dh-params-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %s", err)
	}
	defer os.Remove(file.Name())

	cmd := exec.Command("openssl", "dhparam", "-out", file.Name(), fmt.Sprintf("%d", bits))
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed generating dh params: %s", err)
	}

	contents, err := ioutil.ReadFile(file.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read generated dh params: %s", err)
	}
	return contents, nil
}
