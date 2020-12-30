package ovpn

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/markbates/pkger"
)

func renderTemplate(name, path string, values interface{}) (string, error) {
	f, err := pkger.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to load template: %s", err)
	}
	defer f.Close()

	buf, err := ioutil.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("failed to read template: %s", err)
	}

	// And render it
	t, err := template.New(name).Funcs(sprig.TxtFuncMap()).Parse(string(buf))
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %s", err)
	}
	var result bytes.Buffer
	if err := t.Execute(&result, values); err != nil {
		return "", fmt.Errorf("failed to render template: %s", err)
	}
	return result.String(), nil
}
