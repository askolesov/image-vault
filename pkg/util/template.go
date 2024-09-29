package v2

import (
	"bytes"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

func RenderTemplate(templateStr string, fields any) (string, error) {
	// use go templates to generate the path
	tmpl := template.New("template").Funcs(sprig.FuncMap())
	tmpl, err := tmpl.Parse(templateStr)
	if err != nil {
		return "", err
	}

	var result bytes.Buffer

	err = tmpl.Execute(&result, fields)
	if err != nil {
		return "", err
	}

	return result.String(), nil
}
