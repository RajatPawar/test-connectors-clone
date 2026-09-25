package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/staticschema"
)

func generate(t *testing.T, cfg map[string]any) map[string]staticschema.Object[staticschema.FieldMetadataMapV2, any] {
	t.Helper()

	dir := t.TempDir()
	cfg["spec"] = "testdata/spec.yaml"
	cfg["output"] = dir

	data, _ := json.Marshal(cfg)
	configPath := filepath.Join(dir, "schemagen.json")

	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := run(configPath); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, staticschema.SchemasFile))
	if err != nil {
		t.Fatal(err)
	}

	var out staticschema.Metadata[staticschema.FieldMetadataMapV2, any]
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}

	return out.Modules[common.ModuleRoot].Objects
}

func readOnly(t *testing.T, f staticschema.FieldMetadata) bool {
	t.Helper()

	if f.ReadOnly == nil {
		t.Fatalf("readOnly unset on %+v", f)
	}

	return *f.ReadOnly
}

func TestWritabilityComesFromTheRequestBody(t *testing.T) {
	t.Parallel()

	objects := generate(t, map[string]any{
		"read":  map[string]any{"responseKey": map[string]any{"default": "items"}},
		"write": map[string]any{"widgets": map[string]any{"create": "POST /widgets"}},
	})

	fields := objects["widgets"].Fields
	if readOnly(t, fields["name"]) || readOnly(t, fields["price"]) || readOnly(t, fields["secret"]) {
		t.Fatalf("fields the create accepts must be writable: %+v", fields)
	}

	if !readOnly(t, fields["id"]) || !readOnly(t, fields["createdAt"]) || !readOnly(t, fields["etag"]) {
		t.Fatalf("fields only returned must be readOnly: %+v", fields)
	}

	if fields["price"].ValueType != common.ValueTypeFloat || fields["createdAt"].ValueType != common.ValueTypeDateTime {
		t.Fatalf("number and date-time must keep their types: %+v", fields)
	}
}

func TestReadOnlyIsUnknownWithoutAWriteSection(t *testing.T) {
	t.Parallel()

	fields := generate(t, map[string]any{
		"read": map[string]any{"responseKey": map[string]any{"default": "items"}},
	})["widgets"].Fields

	if fields["name"].ReadOnly != nil {
		t.Fatal("no write section → writability unknown (nil), not guessed")
	}

	if !readOnly(t, fields["etag"]) {
		t.Fatal("the spec's own readOnly still applies")
	}
}

func TestAWriteOnlyObjectIsDescribedByItsCreate(t *testing.T) {
	t.Parallel()

	objects := generate(t, map[string]any{
		"read":  map[string]any{"responseKey": map[string]any{"default": "items"}},
		"write": map[string]any{"gadgets": map[string]any{"create": "POST /gadgets"}},
	})

	gadget, ok := objects["gadgets"]
	if !ok || gadget.URLPath != "/gadgets" {
		t.Fatalf("gadgets missing: %+v", objects)
	}

	if !readOnly(t, gadget.Fields["id"]) || readOnly(t, gadget.Fields["label"]) {
		t.Fatalf("unexpected writability: %+v", gadget.Fields)
	}
}

func TestConfigErrorsAreErrors(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	for name, cfg := range map[string]string{
		"unknown knob":     `{"spec":"testdata/spec.yaml","output":"` + dir + `","read":{"respnseKey":{}}}`,
		"missing op":       `{"spec":"testdata/spec.yaml","output":"` + dir + `","write":{"widgets":{"create":"PUT /widgets"}}}`,
		"bad op format":    `{"spec":"testdata/spec.yaml","output":"` + dir + `","write":{"widgets":{"create":"/widgets"}}}`,
		"unknown procesor": `{"spec":"testdata/spec.yaml","output":"` + dir + `","read":{"displayProcessors":["shout"]}}`,
	} {
		path := filepath.Join(dir, name+".json")
		if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
			t.Fatal(err)
		}

		if err := run(path); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestWrappedBodiesAndOverrides(t *testing.T) {
	t.Parallel()

	objects := generate(t, map[string]any{
		"read": map[string]any{
			"responseKey":   map[string]any{"default": "items"},
			"excludeFields": map[string]any{"*": []string{"etag"}},
		},
		"write": map[string]any{"widgets": map[string]any{"create": "POST /widgets"}},
		"fieldOverrides": map[string]any{
			"widgets": map[string]any{"name": map[string]any{"readOnly": true, "displayName": "Widget name"}},
		},
	})

	fields := objects["widgets"].Fields
	if _, ok := fields["etag"]; ok {
		t.Fatal("excluded field kept")
	}

	if !readOnly(t, fields["name"]) || fields["name"].DisplayName != "Widget name" {
		t.Fatalf("override not applied: %+v", fields["name"])
	}
}

func TestRequiredObjectsMustBeGenerated(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "c.json")
	cfg := `{"spec":"testdata/spec.yaml","output":"` + dir + `","require":["widgets","gizmos"],` +
		`"read":{"responseKey":{"default":"items"}}}`

	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}

	err := run(path)
	if err == nil || !strings.Contains(err.Error(), "gizmos") || strings.Contains(err.Error(), "widgets (") {
		t.Fatalf("expected only gizmos to be reported missing, got %v", err)
	}
}

func TestBadPathsAndOverridesAreErrors(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	for name, cfg := range map[string]string{
		"bodyPath":   `"write":{"widgets":{"create":"POST /widgets","bodyPath":"data.nope"}}`,
		"override":   `"fieldOverrides":{"widgets":{"nmae":{"readOnly":true}}}`,
		"no object":  `"fieldOverrides":{"gizmos":{"id":{"readOnly":true}}}`,
		"no exclude": `"read":{"responseKey":{"default":"items"},"excludeFields":{"gizmos":["id"]}}`,
	} {
		if !strings.HasPrefix(cfg, `"read"`) {
			cfg = `"read":{"responseKey":{"default":"items"}},` + cfg
		}

		path := filepath.Join(dir, name+".json")
		body := `{"spec":"testdata/spec.yaml","output":"` + dir + `",` + cfg + `}`

		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}

		if err := run(path); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}
