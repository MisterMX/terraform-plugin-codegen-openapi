package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	xpv1 "github.com/crossplane/crossplane/apis/apiextensions/v1"
	"github.com/pkg/errors"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/hashicorp/terraform-plugin-codegen-openapi/internal/config"
)

// loadOpenAPISchemaFromXRD from a file. The file may contain a single YAML or
// JSON document.
func loadOpenAPISchemaFromXRD(filename string) ([]byte, *config.Config, error) {
	raw, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, err
	}

	xrd := &xpv1.CompositeResourceDefinition{}
	if err := yaml.Unmarshal(raw, xrd); err != nil {
		return nil, nil, err
	}

	var version *xpv1.CompositeResourceDefinitionVersion
	for _, v := range xrd.Spec.Versions {
		if v.Referenceable {
			version = &v
			break
		}
	}
	if version == nil {
		return nil, nil, errors.New("no referencable version for XRD")
	}
	if version.Schema == nil {
		return nil, nil, errors.New("version has no schema")
	}

	schema := extv1.JSONSchemaProps{}
	if err := json.Unmarshal(version.Schema.OpenAPIV3Schema.Raw, &schema); err != nil {
		return nil, nil, errors.Wrap(err, "cannot parse schema")
	}

	// TODO: Terraform requires all computed properties to be set when filling
	//       the state. That does not work well with properties that are set
	//       leter.
	//       The status is left out for now until their is a reliable way in the
	//       provider to set the unset fields with default values.
	if _, hasStatus := schema.Properties["status"]; hasStatus {
		delete(schema.Properties, "status")
	}

	pathName := xrdOpenAPIPath(xrd, version)

	type pathNameMapVal struct {
		xrd     *xpv1.CompositeResourceDefinition
		version *xpv1.CompositeResourceDefinitionVersion
	}

	pathNameMap := map[string]pathNameMapVal{}

	doc := openAPIDocument{
		Openapi: "3.0.0",
		Info: openAPIDocumentInfo{
			Title:   "XRD openapi",
			Version: "v0.1.0",
		},
	}

	pathNameMap[pathName] = pathNameMapVal{
		xrd:     xrd,
		version: version,
	}

	doc.Paths = map[string]openAPIPath{
		// TODO: Add path for composite
		pathName: {
			Post: &openAPIPathOperation{
				Parameters: []openAPIPathOperationParameter{
					{
						Name:     "namespace",
						In:       "path",
						Required: true,
						Schema: extv1.JSONSchemaProps{
							Type: "string",
						},
					},
					{
						Name:     "name",
						In:       "path",
						Required: true,
						Schema: extv1.JSONSchemaProps{
							Type: "string",
						},
					},
				},
				RequestBody: &openAPIPathOperationRequestBody{
					Required: true,
					Content: map[string]openAPIMediaTypeObject{
						"application/json": {
							Schema: schema,
						},
					},
				},
				Responses: map[string]openAPIPathOperationResponse{
					"200": {
						Description: "Sucess",
						Content: map[string]openAPIMediaTypeObject{
							"application/json": {
								Schema: schema,
							},
						},
					},
				},
			},
		},
	}

	cfg := &config.Config{
		Provider: config.Provider{
			Name: "example_crossplane",
		},
		Resources: map[string]config.Resource{},
	}

	for pathName := range doc.Paths {
		source := pathNameMap[pathName]
		resourceName := strings.ReplaceAll(source.xrd.GetName(), ".", "_")

		cfg.Resources[resourceName] = config.Resource{
			Create: &config.OpenApiSpecLocation{
				Path:   pathName,
				Method: "POST",
			},
			Read: &config.OpenApiSpecLocation{
				Path:   pathName,
				Method: "GET",
			},
			Update: &config.OpenApiSpecLocation{
				Path:   pathName,
				Method: "PUT",
			},
			Delete: &config.OpenApiSpecLocation{
				Path:   pathName,
				Method: "DELETE",
			},
		}
	}

	docRaw, err := json.Marshal(&doc)
	return docRaw, cfg, errors.Wrap(err, "cannot marshal OpenAPI document")
}

type openAPIDocument struct {
	Openapi string                 `json:"openapi"`
	Info    openAPIDocumentInfo    `json:"info"`
	Paths   map[string]openAPIPath `json:"paths"`
}

type openAPIDocumentInfo struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type openAPIPath struct {
	Post *openAPIPathOperation `json:"post,omitempty"`
}

type openAPIPathOperation struct {
	Parameters  []openAPIPathOperationParameter         `json:"parameters"`
	RequestBody *openAPIPathOperationRequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]openAPIPathOperationResponse `json:"responses"`
}

type openAPIPathOperationParameter struct {
	Name     string                `json:"name"`
	In       string                `json:"in"`
	Required bool                  `json:"required"`
	Schema   extv1.JSONSchemaProps `json:"schema"`
}

type openAPIPathOperationRequestBody struct {
	Content  map[string]openAPIMediaTypeObject `json:"content"`
	Required bool                              `json:"required"`
}

type openAPIPathOperationResponse struct {
	Description string                            `json:"description"`
	Content     map[string]openAPIMediaTypeObject `json:"content"`
}

type openAPIMediaTypeObject struct {
	Schema extv1.JSONSchemaProps `json:"schema"`
}

func xrdOpenAPIPath(xrd *xpv1.CompositeResourceDefinition, version *xpv1.CompositeResourceDefinitionVersion) string {
	return fmt.Sprintf(
		"/apis/%s/%s/%s/{namespace}/{name}",
		xrd.Spec.Group,
		version.Name,
		xrd.Spec.ClaimNames.Plural,
	)
}
