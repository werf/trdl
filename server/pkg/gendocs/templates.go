package gendocs

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/hashicorp/vault/sdk/framework"
)

type MethodExampleTemplateData struct {
	Description string
	Path        string
}

type MethodResponseTemplateData struct {
	StatusCode  string
	Description string
}

type MethodParameterTemplateData struct {
	Name         string
	Type         string
	DefaultValue string // default value or <required> if requred
	IsUrlParam   bool
	Description  string
}

func NewMethodParameterTemplateDataFromSchema(paramName, description string, isUrlParam, isRequired bool, schemaDesc *framework.OASSchema) *MethodParameterTemplateData {
	parameter := &MethodParameterTemplateData{
		Name:        paramName,
		Description: description,
		IsUrlParam:  isUrlParam,
	}

	parameter.Type = "<unknown>"
	parameter.DefaultValue = "<optional>"

	if schemaDesc != nil {
		parameter.Type = schemaDesc.Type
		if schemaDesc.Default != nil {
			parameter.DefaultValue = fmt.Sprintf("%v", schemaDesc.Default)
		}
	}

	if isRequired {
		parameter.DefaultValue = "<required>"
	}

	return parameter
}

func NewMethodParameterTemplateData(paramDesc framework.OASParameter, isUrlParam bool) *MethodParameterTemplateData {
	return NewMethodParameterTemplateDataFromSchema(paramDesc.Name, paramDesc.Description, isUrlParam, paramDesc.Required, paramDesc.Schema)
}

type MethodTemplateData struct {
	Name        string
	Summary     string
	Description string
	Path        string
	Parameters  []*MethodParameterTemplateData
	Responses   []*MethodResponseTemplateData
	Examples    []*MethodExampleTemplateData
}

func NewMethodTemplateData(name, path string, urlParameters []framework.OASParameter, methodDesc *framework.OASOperation) *MethodTemplateData {
	method := &MethodTemplateData{
		Name: name,
		Path: path,
	}

	if methodDesc.Summary == "" {
		method.Summary = methodDesc.Description
	} else {
		method.Summary = methodDesc.Summary
		method.Description = methodDesc.Description
	}

	for _, paramDesc := range urlParameters {
		method.Parameters = append(method.Parameters, NewMethodParameterTemplateData(paramDesc, true))
	}

	for _, paramDesc := range methodDesc.Parameters {
		method.Parameters = append(method.Parameters, NewMethodParameterTemplateData(paramDesc, false))
	}

	if methodDesc.RequestBody != nil {
		// TODO: order
		for contentType, content := range methodDesc.RequestBody.Content {
			if contentType == "application/json" && content.Schema != nil && content.Schema.Type == "object" {
				// TODO: order
				for propName, propSchema := range content.Schema.Properties {
					isRequired := false
					for _, name := range content.Schema.Required {
						if name == propName {
							isRequired = true
						}
					}

					method.Parameters = append(method.Parameters, NewMethodParameterTemplateDataFromSchema(propName, propSchema.Description, false, isRequired, propSchema))
				}
			}
		}
	}

	// TODO: order
	for responseCode, responseDesc := range methodDesc.Responses {
		method.Responses = append(method.Responses, &MethodResponseTemplateData{
			StatusCode:  fmt.Sprintf("%d", responseCode),
			Description: responseDesc.Description,
		})
	}

	return method
}

type PathTemplateData struct {
	Name        string
	Description string
	Methods     []*MethodTemplateData
}

type BackendTemplateData struct {
	Description string
	Paths       []*PathTemplateData
}

func NewBackendTemplateData(backendDoc *framework.OASDocument, frameworkBackendRef *framework.Backend) (*BackendTemplateData, error) {
	backendTemplateData := &BackendTemplateData{}

	if frameworkBackendRef != nil {
		backendTemplateData.Description = strings.TrimSpace(frameworkBackendRef.Help)
	}

	for rawPathName, pathDesc := range backendDoc.Paths {
		pathName := FormatPathName(rawPathName)

		path := &PathTemplateData{
			Name:        pathName,
			Description: pathDesc.Description,
		}

		if path.Description == "" {
			return nil, fmt.Errorf("required path %q description", rawPathName)
		}

		if pathDesc.Get != nil {
			path.Methods = append(path.Methods, NewMethodTemplateData("GET", pathName, pathDesc.Parameters, pathDesc.Get))
		}

		if pathDesc.Post != nil {
			path.Methods = append(path.Methods, NewMethodTemplateData("POST", pathName, pathDesc.Parameters, pathDesc.Post))
		}

		if pathDesc.Delete != nil {
			path.Methods = append(path.Methods, NewMethodTemplateData("DELETE", pathName, pathDesc.Parameters, pathDesc.Delete))
		}

		backendTemplateData.Paths = append(backendTemplateData.Paths, path)
	}

	return backendTemplateData, nil
}

func FormatPathName(pathName string) string {
	pathParts := strings.Split(pathName, "/")

	var newPathParts []string

	for _, part := range pathParts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			newPart := ":" + strings.TrimSuffix(strings.TrimPrefix(part, "{"), "}")
			newPathParts = append(newPathParts, newPart)
		} else {
			newPathParts = append(newPathParts, part)
		}
	}

	return strings.Join(newPathParts, "/")
}

var (
	BackendOverviewTemplate = `### Description

{{ .Description }}

{{ if .Paths -}}
### Paths
{{   range .Paths }}
* ` + "`{{ .Name }}`" + `
{{   end }}
{{ end }}
`

	PathTemplate = `## ` + "`{{ .Name }}`" + `

{{ .Description }}

{{ range .Methods -}}
### ` + "{{ .Summary }}" + `

{{   if .Description -}}
{{ .Description }}
{{   end }}

| Method | Path |
|--------|------|
| ` + "`{{ .Name }}`" + ` | ` + "`{{ .Path }}`" + ` |

{{   if .Parameters -}}
#### Parameters

{{     range .Parameters -}}
* ` + "`{{ .Name }}`" + ` (` + "`{{ .Type }}{{ if .DefaultValue }}: {{ .DefaultValue }}{{ end }}`" + `{{ if .IsUrlParam }}, url param{{ end }}) — {{ .Description }}.
{{     end -}}
{{   end }}
{{   if .Responses -}}
#### Responses

{{     range .Responses -}}
* {{ .StatusCode }} — {{ .Description }}. 
{{     end -}}
{{   end }}
{{   if .Examples -}}
#### Examples

{{     range .Examples -}}
##### {{ .Description }}
{{       if eq .Method "GET" }}
    curl  --header \"X-Vault-Token: ...\" http://127.0.0.1:8200/v1/PLUGIN_MOOUNT/{{ .Path }} 
{{       else if eq .Method "POST" }}
    curl  --header \"X-Vault-Token: ...\" --request POST --data '{\"key\": \"value\", ...}' http://127.0.0.1:8200/v1/PLUGIN_MOUNT/{{ .Path }} 
{{       else if eq .Method "DELETE" }}
    curl  --header \"X-Vault-Token: ...\" --request DELETE http://127.0.0.1:8200/v1/PLUGIN_MOUNT/{{ .Path }} 
{{       end }}
{{     end }}
{{   end }}
{{ end }}
`
)

func ExecuteTemplate(tpl string, data interface{}) (string, error) {
	t, err := template.New("root").Parse(tpl)
	if err != nil {
		return "", fmt.Errorf("error parsing template: %s", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("error executing template: %s", err)
	}

	return strings.TrimSpace(buf.String()), nil
}
