package gimage

import (
	"bytes"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestGetSwaggerMatchesOpenAPIYAML(t *testing.T) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile("../../openapi.yaml")
	if err != nil {
		t.Fatalf("failed to load openapi.yaml: %v", err)
	}
	jsonBytes, err := doc.MarshalJSON()
	if err != nil {
		t.Fatalf("failed to marshal openapi.yaml to JSON: %v", err)
	}

	embeddedBytes, err := rawSpec()
	if err != nil {
		t.Fatalf("failed to decode embedded spec: %v", err)
	}

	if !bytes.Equal(bytes.TrimSpace(jsonBytes), bytes.TrimSpace(embeddedBytes)) {
		t.Fatalf("embedded spec in spec.gen.go is out of sync with openapi.yaml")
	}

	swagger, err := GetSwagger()
	if err != nil {
		t.Fatalf("GetSwagger failed: %v", err)
	}
	if swagger == nil || swagger.Info == nil {
		t.Fatal("expected non-nil swagger spec")
	}
}
