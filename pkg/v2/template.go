package v2

import (
	"bytes"
	"text/template"
)

func RenderTemplate(templateStr string, fields any) (string, error) {
	// use go templates to generate the path
	tmpl := template.Must(template.New("template").Parse(templateStr))
	var result bytes.Buffer

	err := tmpl.Execute(&result, fields)
	if err != nil {
		return "", err
	}

	return result.String(), nil
}
