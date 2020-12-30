package ovpn

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

func renderTemplate(name, templateString string, values interface{}) (string, error) {
	// And render it
	t, err := template.New(name).Funcs(sprig.TxtFuncMap()).Parse(templateString)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %s", err)
	}
	var result bytes.Buffer
	if err := t.Execute(&result, values); err != nil {
		return "", fmt.Errorf("failed to render template: %s", err)
	}
	return result.String(), nil
}
