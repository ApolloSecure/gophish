package api

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

var openAPIVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

func TestPublishedOpenAPIContract(t *testing.T) {
	contractPath := filepath.Join("..", "..", "docs", "openapi.json")
	data, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("read published OpenAPI contract: %v", err)
	}

	document := map[string]interface{}{}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("parse published OpenAPI contract: %v", err)
	}
	if document["openapi"] != "3.0.3" {
		t.Fatalf("OpenAPI version = %v, want 3.0.3", document["openapi"])
	}
	info := openAPIObject(t, document, "info")
	version, _ := info["version"].(string)
	if !openAPIVersionPattern.MatchString(version) {
		t.Fatalf("API contract version %q is not semantic versioning", version)
	}

	paths := openAPIObject(t, document, "paths")
	registeredPaths := registeredAPIPaths(t)
	for path, value := range paths {
		if _, ok := registeredPaths[path]; !ok {
			t.Errorf("documented path %q is not registered by the API server", path)
		}
		pathItem, ok := value.(map[string]interface{})
		if !ok {
			t.Errorf("path %q is not an object", path)
			continue
		}
		for method, operationValue := range pathItem {
			operation, ok := operationValue.(map[string]interface{})
			if !ok {
				t.Errorf("operation %s %s is not an object", strings.ToUpper(method), path)
				continue
			}
			responses, ok := operation["responses"].(map[string]interface{})
			if !ok || len(responses) == 0 {
				t.Errorf("operation %s %s has no responses", strings.ToUpper(method), path)
			}
		}
	}
	validateOpenAPIReferences(t, document, document)
}

func registeredAPIPaths(t *testing.T) map[string]struct{} {
	t.Helper()
	server := NewServer()
	router, ok := server.handler.(*mux.Router)
	if !ok {
		t.Fatalf("API handler type = %T, want *mux.Router", server.handler)
	}
	paths := map[string]struct{}{}
	if err := router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		template, err := route.GetPathTemplate()
		if err != nil {
			return nil
		}
		template = regexp.MustCompile(`\{([^}:]+):[^}]+\}`).ReplaceAllString(template, `{$1}`)
		paths[template] = struct{}{}
		return nil
	}); err != nil {
		t.Fatalf("walk API routes: %v", err)
	}
	return paths
}

func openAPIObject(t *testing.T, parent map[string]interface{}, key string) map[string]interface{} {
	t.Helper()
	value, ok := parent[key].(map[string]interface{})
	if !ok {
		t.Fatalf("OpenAPI %s is not an object", key)
	}
	return value
}

func validateOpenAPIReferences(t *testing.T, document map[string]interface{}, value interface{}) {
	t.Helper()
	switch typed := value.(type) {
	case map[string]interface{}:
		for key, child := range typed {
			if key == "$ref" {
				reference, ok := child.(string)
				if !ok || !strings.HasPrefix(reference, "#/") {
					t.Errorf("unsupported OpenAPI reference %v", child)
					continue
				}
				if _, err := resolveOpenAPIReference(document, reference); err != nil {
					t.Errorf("invalid OpenAPI reference %q: %v", reference, err)
				}
				continue
			}
			validateOpenAPIReferences(t, document, child)
		}
	case []interface{}:
		for _, child := range typed {
			validateOpenAPIReferences(t, document, child)
		}
	}
}

func resolveOpenAPIReference(document map[string]interface{}, reference string) (interface{}, error) {
	var current interface{} = document
	for _, rawPart := range strings.Split(strings.TrimPrefix(reference, "#/"), "/") {
		part := strings.ReplaceAll(strings.ReplaceAll(rawPart, "~1", "/"), "~0", "~")
		object, ok := current.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("%q does not resolve through an object", part)
		}
		current, ok = object[part]
		if !ok {
			return nil, fmt.Errorf("component %q does not exist", part)
		}
	}
	return current, nil
}
