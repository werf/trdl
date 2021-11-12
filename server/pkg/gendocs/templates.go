package gendocs

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
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
	Name               string
	Type               string
	DefaultValue       string
	RequiredOrOptional string
	Description        string
}

func NewMethodParameterTemplateDataFromSchema(paramName, description string, isUrlPattern, isRequired bool, schemaDesc *framework.OASSchema) *MethodParameterTemplateData {
	parameter := &MethodParameterTemplateData{
		Name:        paramName,
		Description: description,
	}

	if isRequired {
		parameter.RequiredOrOptional = "required"
	} else {
		parameter.RequiredOrOptional = "optional"
	}

	parameter.Type = "unknown type"

	if isUrlPattern {
		parameter.Type = "url pattern"
	} else if schemaDesc != nil {
		parameter.Type = schemaDesc.Type
		if schemaDesc.Default != nil {
			parameter.DefaultValue = fmt.Sprintf("%v", schemaDesc.Default)
		}
	}

	return parameter
}

func NewMethodParameterTemplateData(paramDesc framework.OASParameter, isUrlPattern bool) *MethodParameterTemplateData {
	return NewMethodParameterTemplateDataFromSchema(paramDesc.Name, paramDesc.Description, isUrlPattern, paramDesc.Required, paramDesc.Schema)
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
		keys := make([]string, 0, len(methodDesc.RequestBody.Content))
		for k := range methodDesc.RequestBody.Content {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, contentType := range keys {
			content := methodDesc.RequestBody.Content[contentType]

			if contentType == "application/json" && content.Schema != nil && content.Schema.Type == "object" {
				props := make([]string, 0, len(content.Schema.Properties))
				for k := range content.Schema.Properties {
					props = append(props, k)
				}
				sort.Strings(props)

				for _, propName := range props {
					propSchema := content.Schema.Properties[propName]

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

	keys := make([]int, 0, len(methodDesc.Responses))
	for k := range methodDesc.Responses {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, statusCode := range keys {
		respDesc := methodDesc.Responses[statusCode]
		method.Responses = append(method.Responses, &MethodResponseTemplateData{
			StatusCode:  fmt.Sprintf("%d", statusCode),
			Description: respDesc.Description,
		})
	}

	return method
}

type PathTemplateData struct {
	Name        string
	Link        string
	Description string
	Synopsis    string
	Methods     []*MethodTemplateData
}

type BackendTemplateData struct {
	Description string
	Paths       []*PathTemplateData
}

func NewBackendTemplateData(backendDoc *framework.OASDocument, frameworkBackendRef *framework.Backend, formatPathLink func(markdownPagePath string) string) (*BackendTemplateData, error) {
	backendTemplateData := &BackendTemplateData{}

	if frameworkBackendRef != nil {
		backendTemplateData.Description = strings.TrimSpace(frameworkBackendRef.Help)
	}

	keys := make([]string, 0, len(backendDoc.Paths))
	for k := range backendDoc.Paths {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, rawPathName := range keys {
		pathName := FormatPathName(rawPathName)
		pathDesc := backendDoc.Paths[rawPathName]

		// FIXME: pathDesc.Description is actually a HelpSynopsis, where is HelpDescription?

		descParts := strings.SplitN(pathDesc.Description, " ", 2)
		descParts[0] = strings.Title(descParts[0])
		description := strings.Join(descParts, " ")
		description = strings.TrimSuffix(description, ".") + "."

		path := &PathTemplateData{
			Name:        pathName,
			Synopsis:    strings.ToLower(pathDesc.Description),
			Description: description,
		}

		if formatPathLink != nil {
			markdownPath, err := FormatPathPatternAsFilesystemMarkdownPath(pathName)
			if err != nil {
				return nil, fmt.Errorf("unable to format path pattern %q as filesystem markdown path: %s", pathName, err)
			}

			path.Link = formatPathLink(markdownPath)
		}

		if path.Description == "" {
			return nil, fmt.Errorf("required path %q description", rawPathName)
		}

		if pathDesc.Post != nil {
			path.Methods = append(path.Methods, NewMethodTemplateData("POST", pathName, pathDesc.Parameters, pathDesc.Post))
		}

		if pathDesc.Get != nil {
			path.Methods = append(path.Methods, NewMethodTemplateData("GET", pathName, pathDesc.Parameters, pathDesc.Get))
		}

		if pathDesc.Delete != nil {
			path.Methods = append(path.Methods, NewMethodTemplateData("DELETE", pathName, pathDesc.Parameters, pathDesc.Delete))
		}

		backendTemplateData.Paths = append(backendTemplateData.Paths, path)
	}

	return backendTemplateData, nil
}

func FormatPathPatternAsFilesystemMarkdownPath(pattern string) (string, error) {
	// fmt.Printf("INPUT PATTERN: %q\n", pattern)
	if pattern == "/" {
		return "index.md", nil
	}

	parts := strings.Split(pattern, "/")
	var newParts []string

	for _, part := range parts {
		if strings.HasPrefix(part, ":") {
			newParts = append(newParts, strings.TrimPrefix(part, ":"))
		} else {
			newParts = append(newParts, part)
		}
	}

	return filepath.Join(newParts...) + ".md", nil
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
	BackendOverviewTemplate = `	{{ .Description }}

{{ if .Paths -}}
## Paths
{{   range .Paths }}
{{     if .Link -}}
* ` + "[`{{ .Name }}`]({{ .Link }})" + ` — {{ .Synopsis }}.
{{-     else -}}
* ` + "`{{ .Name }}`" + ` — {{ .Synopsis }}.
{{-     end }}
{{   end }}
{{ end }}
`

	PathTemplate = `{{ .Description }}

{{ range .Methods -}}
## ` + "{{ .Summary }}" + `

{{   if .Description -}}
{{ .Description }}
{{   end -}}

| Method | Path |
|--------|------|
| ` + "`{{ .Name }}`" + ` | ` + "`{{ .Path }}`" + ` |

{{   if .Parameters -}}
### Parameters

{{     range .Parameters -}}
* ` + "`{{ .Name }}`" + ` ({{ .Type }}, {{ .RequiredOrOptional }}{{ if .DefaultValue }}, default: ` + "`{{ .DefaultValue }}`" + `{{ end }}) — {{ .Description }}.
{{     end -}}
{{   end }}
{{   if .Responses -}}
### Responses

{{     range .Responses -}}
* {{ .StatusCode }} — {{ .Description }}. 
{{     end -}}
{{   end }}
{{   if .Examples -}}
### Examples

{{     range .Examples -}}
#### {{ .Description }}
{{       if eq .Method "GET" }}
    curl  --header \"X-Vault-Token: ...\" http://127.0.0.1:8200/v1/PLUGIN_MOOUNT/{{ .Path }} 
{{       else if eq .Method "POST" }}
    curl  --header \"X-Vault-Token: ...\" --request POST --data '{\"key\": \"value\", ...}' http://127.0.0.1:8200/v1/PLUGIN_MOUNT/{{ .Path }} 
{{       else if eq .Method "DELETE" }}
    curl  --header \"X-Vault-Token: ...\" --request DELETE http://127.0.0.1:8200/v1/PLUGIN_MOUNT/{{ .Path }} 
{{       end }}
{{     end -}}
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
