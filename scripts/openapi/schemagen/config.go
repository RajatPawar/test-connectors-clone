package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Config is everything a per-provider OpenAPI script used to hard-code, as data. It is committed
// next to the output (providers/<name>/metadata/schemagen.json) so the schema can be regenerated
// — and checked — by anyone, from the same inputs.
type Config struct {
	// Spec is the OpenAPI document: a path relative to the repo root, or an http(s) URL.
	// Swagger 2.0 is detected and converted to OpenAPI 3.
	Spec string `json:"spec"`
	// Output is the directory schemas.json is written to, e.g. "providers/acme/metadata".
	Output string `json:"output"`
	// Module is the module id the objects belong to. Default "root".
	Module string `json:"module,omitempty"`
	// HoistCommonPath moves the longest common prefix of object paths into the module path
	// (the old SaveSchemas behaviour). Default false: object paths are written in full.
	HoistCommonPath bool `json:"hoistCommonPath,omitempty"`
	// MediaType of request and response bodies. Default "application/json".
	MediaType string `json:"mediaType,omitempty"`

	// Require lists the objects this connector needs (its read/write scope). A run that does not
	// produce every one of them FAILS, naming each missing object and why api3 could not extract
	// it — so "the output silently lacks my object" is an error, never a success to rerun blindly.
	Require []string `json:"require,omitempty"`

	Read  ReadConfig             `json:"read"`
	Write map[string]WriteConfig `json:"write,omitempty"`

	// FieldOverrides corrects what the spec gets wrong, per object then field, applied last:
	// {"contacts": {"id": {"readOnly": true}, "score": {"valueType": "float"}}}. Every override is
	// in the committed config, so a reviewer sees each one.
	FieldOverrides map[string]map[string]FieldOverride `json:"fieldOverrides,omitempty"`
}

// FieldOverride replaces one property of a generated field. Unset means keep the generated value.
type FieldOverride struct {
	ReadOnly    *bool  `json:"readOnly,omitempty"`
	ValueType   string `json:"valueType,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}

// ReadConfig selects the list endpoints objects are read from and how their records are found.
type ReadConfig struct {
	// Method of the list operation: "GET" (default) or "POST" (search-style reads).
	Method string `json:"method,omitempty"`
	// Allow / Deny are path rules: exact, "prefix*" or "*suffix". Allow, when set, is the whole list.
	Allow []string `json:"allow,omitempty"`
	Deny  []string `json:"deny,omitempty"`
	// AllowIDPaths keeps paths with {params}. Default false: only constant paths are objects.
	AllowIDPaths bool `json:"allowIdPaths,omitempty"`
	// Objects names an object for a path ("/v2/people": "contacts"). Default: the last segment.
	Objects map[string]string `json:"objects,omitempty"`
	// DisplayNames overrides an object's display name by object name.
	DisplayNames map[string]string `json:"displayNames,omitempty"`
	// DisplayProcessors format display names in order: camelCaseToSpaces, capitalize,
	// slashesToSpaces, pluralize. Default: camelCaseToSpaces, capitalize.
	DisplayProcessors []string `json:"displayProcessors,omitempty"`
	// ResponseKey says which response field holds the record array.
	ResponseKey ResponseKeyConfig `json:"responseKey,omitempty"`
	// AutoSelectArray picks the only array-of-objects property when the key rule finds none.
	AutoSelectArray bool `json:"autoSelectArray,omitempty"`
	// Flatten lifts a nested object's properties to the top level ("attributes" for JSON:API).
	Flatten []string `json:"flatten,omitempty"`
	// OnlyOptionalQueryParams drops list operations that require a query parameter.
	OnlyOptionalQueryParams bool `json:"onlyOptionalQueryParams,omitempty"`
	// StripPathPrefix removes a prefix from every object path (e.g. "/v1" when the catalog
	// BaseURL already ends in it).
	StripPathPrefix string `json:"stripPathPrefix,omitempty"`
	// ExcludeFields drops fields per object ("*" applies to every object):
	// {"*": ["links"], "contacts": ["_embedded"]}.
	ExcludeFields map[string][]string `json:"excludeFields,omitempty"`
}

// ResponseKeyConfig: Objects wins, then Default. Default is a field name, "@identical" (the field
// named like the object) or "" (a bare array, or AutoSelectArray).
type ResponseKeyConfig struct {
	Default string            `json:"default,omitempty"`
	Objects map[string]string `json:"objects,omitempty"`
}

// WriteConfig names the operations that write an object, as "METHOD /path". Their request bodies
// decide which fields are writable: a field only ever returned is readOnly, a field a create or
// update accepts is not. An object written but not read gets its fields from these operations.
type WriteConfig struct {
	Create string `json:"create,omitempty"`
	Update string `json:"update,omitempty"`
	// BodyPath is where the record's fields sit inside the request body, dotted, when the body
	// wraps them: "data.attributes" for JSON:API, "contact" for {"contact": {...}}. Default: the
	// body itself.
	BodyPath string `json:"bodyPath,omitempty"`
	// ResponsePath is the same for the create's response (used only for objects that are written
	// but not read): "data" for {"data": {...}}. Default: the response itself.
	ResponsePath string `json:"responsePath,omitempty"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		return nil, err
	}

	var cfg Config

	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields() // a misspelt knob must fail, not be silently ignored

	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("config %s: %w", path, err)
	}

	if cfg.Spec == "" || cfg.Output == "" {
		return nil, fmt.Errorf("config %s: spec and output are required", path)
	}

	if cfg.Module == "" {
		cfg.Module = "root"
	}

	if cfg.MediaType == "" {
		cfg.MediaType = "application/json"
	}

	if cfg.Read.Method == "" {
		cfg.Read.Method = "GET"
	}

	cfg.Read.Method = strings.ToUpper(cfg.Read.Method)

	if cfg.Read.DisplayProcessors == nil {
		cfg.Read.DisplayProcessors = []string{"camelCaseToSpaces", "capitalize"}
	}

	return &cfg, nil
}

// operation splits "PATCH /docs/{docId}" into its method and path.
func operation(ref string) (string, string, error) {
	method, path, ok := strings.Cut(strings.TrimSpace(ref), " ")
	if !ok || !strings.HasPrefix(strings.TrimSpace(path), "/") {
		return "", "", fmt.Errorf("operation %q must be \"METHOD /path\"", ref)
	}

	return strings.ToUpper(method), strings.TrimSpace(path), nil
}
