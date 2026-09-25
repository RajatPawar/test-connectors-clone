// Command schemagen turns a provider's OpenAPI spec into the connector's static object metadata
// (schemas.json), from a committed config instead of a hand-written per-provider script.
//
//	go run ./scripts/openapi/schemagen -config providers/acme/metadata/schemagen.json
//
// Reads are extracted with api3, exactly as the per-provider scripts under scripts/openapi do; every
// knob those scripts hard-coded is a config field (see Config). On top of that, objects listed under
// `write` get field writability from the spec's request bodies: a field the create/update body
// accepts is readOnly=false, a field only ever returned is readOnly=true. Fields the spec itself
// marks readOnly are readOnly=true everywhere. Output is the V2 field format.
//
// It prints a JSON report (objects written, and every endpoint api3 could not extract with the
// reason) and exits non-zero only when the config or spec is unusable.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/invopop/yaml"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/staticschema"
	"github.com/amp-labs/connectors/tools/fileconv"
	"github.com/amp-labs/connectors/tools/fileconv/api3"
	"github.com/amp-labs/connectors/tools/scrapper"
)

type problem struct {
	Object string `json:"object"`
	Path   string `json:"path,omitempty"`
	Error  string `json:"error"`
}

type report struct {
	Output   string    `json:"output"`
	Objects  []string  `json:"objects"`
	Written  []string  `json:"written,omitempty"`
	Problems []problem `json:"problems,omitempty"`
}

func main() {
	configPath := flag.String("config", "", "path to schemagen.json")
	flag.Parse()

	if err := run(*configPath); err != nil {
		fmt.Fprintln(os.Stderr, "schemagen:", err) //nolint:forbidigo
		os.Exit(1)
	}
}

func run(configPath string) error {
	if configPath == "" {
		return fmt.Errorf("-config is required")
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}

	doc, err := loadSpec(cfg.Spec)
	if err != nil {
		return err
	}

	rep := &report{Output: cfg.Output + "/" + staticschema.SchemasFile}
	objects := map[string]*object{}

	if err := readObjects(cfg, doc, objects, rep); err != nil {
		return err
	}

	if err := writeObjects(cfg, doc, objects, rep); err != nil {
		return err
	}

	if err := applyOverrides(cfg, objects); err != nil {
		return err
	}

	if err := checkRequired(cfg, objects, rep); err != nil {
		return err
	}

	metadata := staticschema.NewMetadata[staticschema.FieldMetadataMapV2]()

	for _, name := range sortedKeys(objects) {
		obj := objects[name]
		metadata.Add(common.ModuleID(cfg.Module), name, obj.displayName, obj.path, obj.responseKey,
			obj.fields, nil, nil)
		rep.Objects = append(rep.Objects, name)
	}

	writer := scrapper.NewWriter[staticschema.FieldMetadataMapV2](fileconv.NewPath(cfg.Output))
	if cfg.HoistCommonPath {
		err = writer.SaveSchemas(metadata)
	} else {
		err = writer.FlushSchemas(metadata)
	}

	if err != nil {
		return err
	}

	out, _ := json.MarshalIndent(rep, "", "  ")
	fmt.Println(string(out)) //nolint:forbidigo

	return nil
}

type object struct {
	displayName string
	path        string
	responseKey string
	fields      staticschema.FieldMetadataMapV2
	// props are the spec's declarations for the fields, for readOnly and type decisions.
	props map[string]property
}

func readObjects(cfg *Config, doc *openapi3.T, objects map[string]*object, rep *report) error {
	rc := cfg.Read
	opts := []api3.Option{api3.WithMediaType(cfg.MediaType)}

	processors, err := displayProcessors(rc.DisplayProcessors)
	if err != nil {
		return err
	}

	opts = append(opts, api3.WithDisplayNamePostProcessors(processors...))

	if rc.AutoSelectArray {
		opts = append(opts, api3.WithArrayItemAutoSelection())
	}

	if len(rc.Flatten) > 0 {
		opts = append(opts, api3.WithPropertyFlattening(func(_, field string) bool {
			return contains(rc.Flatten, field)
		}))
	}

	if rc.OnlyOptionalQueryParams {
		opts = append(opts, api3.WithParameterFilterGetMethod(api3.OnlyOptionalQueryParameters))
	}

	explorer := api3.NewExplorer[any](doc, opts...)

	var matcher api3.AndPathMatcher
	if len(rc.Allow) > 0 {
		matcher = append(matcher, api3.NewAllowPathStrategy(rc.Allow))
	}

	if len(rc.Deny) > 0 {
		matcher = append(matcher, api3.NewDenyPathStrategy(rc.Deny))
	}

	if !rc.AllowIDPaths {
		matcher = append(matcher, api3.IDPathIgnorer{})
	}

	if len(matcher) == 0 {
		matcher = append(matcher, api3.DefaultPathMatcher{})
	}

	schemas, err := explorer.ReadObjects(rc.Method, matcher, rc.Objects, rc.DisplayNames, responseKeyLocator(rc.ResponseKey))
	if err != nil {
		return fmt.Errorf("reading objects: %w", err)
	}

	for _, schema := range schemas {
		if schema.Problem != nil {
			rep.Problems = append(rep.Problems, problem{
				Object: schema.ObjectName, Path: schema.URLPath, Error: schema.Problem.Error(),
			})

			continue
		}

		props := itemProperties(doc, rc.Method, schema.URLPath, schema.ResponseKey, cfg.MediaType, rc.Flatten)
		obj := &object{
			displayName: schema.DisplayName,
			path:        strings.TrimPrefix(schema.URLPath, rc.StripPathPrefix),
			responseKey: schema.ResponseKey,
			fields:      staticschema.FieldMetadataMapV2{},
			props:       props,
		}

		for _, field := range schema.Fields {
			p, ok := props[field.Name]
			if !ok {
				p = property{Type: field.Type, Enum: field.ValueOptions}
			}

			obj.props[field.Name] = p // every extracted field has a declaration the write pass can use
			obj.fields[field.Name] = fieldMetadata(field.Name, p, specReadOnly(p))
		}

		for name := range obj.props {
			if _, extracted := obj.fields[name]; !extracted {
				delete(obj.props, name) // declared on the schema but not extracted by api3 (e.g. flattened parent)
			}
		}

		objects[schema.ObjectName] = obj
	}

	return nil
}

// writeObjects decides writability for every object under `write`, from its create/update bodies.
func writeObjects(cfg *Config, doc *openapi3.T, objects map[string]*object, rep *report) error {
	for _, name := range sortedKeys(cfg.Write) {
		wc := cfg.Write[name]
		accepted := map[string]property{}

		var createPath string

		for kind, ref := range map[string]string{"create": wc.Create, "update": wc.Update} {
			if ref == "" {
				continue
			}

			method, path, err := operation(ref)
			if err != nil {
				return fmt.Errorf("write.%s.%s: %w", name, kind, err)
			}

			op := findOperation(doc, method, path)
			if op == nil {
				return fmt.Errorf("write.%s.%s: %s is not in the spec", name, kind, ref)
			}

			body, err := descend(requestSchema(op, cfg.MediaType), wc.BodyPath)
			if err != nil {
				return fmt.Errorf("write.%s.bodyPath: %s request body: %w", name, ref, err)
			}

			for field, p := range properties(body) {
				if !p.ReadOnly {
					accepted[field] = p
				}
			}

			if kind == "create" {
				createPath = path
			}
		}

		obj, isRead := objects[name]
		if !isRead {
			// Written but not read: the create's response describes the record.
			obj = &object{
				displayName: name,
				path:        strings.TrimPrefix(createPath, cfg.Read.StripPathPrefix),
				fields:      staticschema.FieldMetadataMapV2{},
				props:       map[string]property{},
			}

			if method, path, err := operation(wc.Create); err == nil {
				resp, err := descend(successSchema(findOperation(doc, method, path), cfg.MediaType), wc.ResponsePath)
				if err != nil {
					return fmt.Errorf("write.%s.responsePath: %s response: %w", name, wc.Create, err)
				}

				for field, p := range properties(resp) {
					obj.props[field] = p
				}
			}

			objects[name] = obj
		}

		for field := range obj.fields {
			delete(obj.fields, field)
		}

		for field, p := range obj.props {
			_, writable := accepted[field]
			obj.fields[field] = fieldMetadata(field, p, boolPtr(p.ReadOnly || !writable))
		}

		for field, p := range accepted {
			if _, known := obj.fields[field]; !known {
				obj.fields[field] = fieldMetadata(field, p, boolPtr(false)) // accepted, never returned
			}
		}

		rep.Written = append(rep.Written, name)
	}

	return nil
}

// itemProperties are the declared properties of an object's records: the list operation's
// response, the array under its response key (or the response itself when it is an array), then
// that array's items — including properties lifted out of flattened sub-objects.
func itemProperties(
	doc *openapi3.T, method, path, responseKey, mediaType string, flatten []string,
) map[string]property {
	out := map[string]property{}
	items := recordSchema(successSchema(findOperation(doc, method, path), mediaType), responseKey, 0)

	if items == nil || items.Value == nil {
		return out
	}

	for name, p := range properties(items) {
		out[name] = p
	}

	for _, field := range flatten {
		if nested := findProperty(items, field, 0); nested != nil {
			for name, p := range properties(nested) {
				out[name] = p
			}
		}
	}

	return out
}

func recordSchema(resp *openapi3.SchemaRef, responseKey string, depth int) *openapi3.SchemaRef {
	if resp == nil || resp.Value == nil || depth > maxSchemaDepth {
		return nil
	}

	if responseKey != "" {
		resp = findProperty(resp, responseKey, 0)
		if resp == nil || resp.Value == nil {
			return nil
		}
	}

	if resp.Value.Items != nil {
		return resp.Value.Items
	}

	return nil
}

// findProperty looks a property up through allOf/oneOf/anyOf.
func findProperty(ref *openapi3.SchemaRef, name string, depth int) *openapi3.SchemaRef {
	if ref == nil || ref.Value == nil || depth > maxSchemaDepth {
		return nil
	}

	if p, ok := ref.Value.Properties[name]; ok {
		return p
	}

	for _, group := range [][]*openapi3.SchemaRef{ref.Value.AllOf, ref.Value.OneOf, ref.Value.AnyOf} {
		for _, part := range group {
			if p := findProperty(part, name, depth+1); p != nil {
				return p
			}
		}
	}

	return nil
}

// descend follows a dotted path of properties (through allOf/oneOf/anyOf) into a schema. An empty
// path is the schema itself.
func descend(ref *openapi3.SchemaRef, path string) (*openapi3.SchemaRef, error) {
	if path == "" {
		return ref, nil
	}

	for _, key := range strings.Split(path, ".") {
		next := findProperty(ref, key, 0)
		if next == nil {
			return nil, fmt.Errorf("no property %q (path %q)", key, path)
		}

		ref = next
	}

	return ref, nil
}

// applyOverrides drops excluded fields and applies fieldOverrides. An override or exclusion that
// names an object or field the output does not have is an error — it is a typo, or a config that
// no longer matches the spec, and silently ignoring it would hide exactly that.
func applyOverrides(cfg *Config, objects map[string]*object) error {
	for objectName, fields := range cfg.Read.ExcludeFields {
		targets := []string{objectName}
		if objectName == "*" {
			targets = sortedKeys(objects)
		} else if _, ok := objects[objectName]; !ok {
			return fmt.Errorf("read.excludeFields: no object %q in the output", objectName)
		}

		for _, target := range targets {
			for _, field := range fields {
				delete(objects[target].fields, field)
			}
		}
	}

	for objectName, fields := range cfg.FieldOverrides {
		obj, ok := objects[objectName]
		if !ok {
			return fmt.Errorf("fieldOverrides: no object %q in the output", objectName)
		}

		for field, override := range fields {
			fm, ok := obj.fields[field]
			if !ok {
				return fmt.Errorf("fieldOverrides.%s: no field %q in the output", objectName, field)
			}

			if override.ReadOnly != nil {
				fm.ReadOnly = override.ReadOnly
			}

			if override.ValueType != "" {
				fm.ValueType = common.ValueType(override.ValueType)
			}

			if override.DisplayName != "" {
				fm.DisplayName = override.DisplayName
			}

			obj.fields[field] = fm
		}
	}

	return nil
}

func checkRequired(cfg *Config, objects map[string]*object, rep *report) error {
	var missing []string

	for _, name := range cfg.Require {
		if _, ok := objects[name]; ok {
			continue
		}

		why := "no list endpoint in read (check read.allow/deny/objects) and no write entry"

		for _, p := range rep.Problems {
			if p.Object == name {
				why = p.Path + ": " + p.Error

				break
			}
		}

		missing = append(missing, fmt.Sprintf("%s (%s)", name, why))
	}

	if len(missing) > 0 {
		return fmt.Errorf("required objects not generated: %s", strings.Join(missing, "; "))
	}

	return nil
}

func findOperation(doc *openapi3.T, method, path string) *openapi3.Operation {
	if doc.Paths == nil {
		return nil
	}

	item := doc.Paths.Find(path)
	if item == nil {
		return nil
	}

	return item.GetOperation(method)
}

func requestSchema(op *openapi3.Operation, mediaType string) *openapi3.SchemaRef {
	if op == nil || op.RequestBody == nil || op.RequestBody.Value == nil {
		return nil
	}

	return mediaSchema(op.RequestBody.Value.Content, mediaType)
}

// successSchema is the first 2xx response body — creates often answer 201 or 202, which api3
// (200 only) never looks at.
func successSchema(op *openapi3.Operation, mediaType string) *openapi3.SchemaRef {
	if op == nil || op.Responses == nil {
		return nil
	}

	for _, code := range []string{"200", "201", "202", "2XX", "default"} {
		if resp := op.Responses.Value(code); resp != nil && resp.Value != nil {
			if schema := mediaSchema(resp.Value.Content, mediaType); schema != nil {
				return schema
			}
		}
	}

	return nil
}

func mediaSchema(content openapi3.Content, mediaType string) *openapi3.SchemaRef {
	if media := content.Get(mediaType); media != nil {
		return media.Schema
	}

	return nil
}

func responseKeyLocator(rk ResponseKeyConfig) api3.ObjectArrayLocator {
	return func(objectName, fieldName string) bool {
		key, ok := rk.Objects[objectName]
		if !ok {
			key = rk.Default
		}

		switch key {
		case "@identical":
			return fieldName == objectName
		case "":
			return false
		default:
			return fieldName == key
		}
	}
}

func displayProcessors(names []string) ([]api3.DisplayNameProcessor, error) {
	known := map[string]api3.DisplayNameProcessor{
		"camelCaseToSpaces": api3.CamelCaseToSpaceSeparated,
		"capitalize":        api3.CapitalizeFirstLetterEveryWord,
		"slashesToSpaces":   api3.SlashesToSpaceSeparated,
		"pluralize":         api3.Pluralize,
	}

	out := make([]api3.DisplayNameProcessor, 0, len(names))

	for _, name := range names {
		p, ok := known[name]
		if !ok {
			return nil, fmt.Errorf("unknown display processor %q", name)
		}

		out = append(out, p)
	}

	return out, nil
}

func loadSpec(location string) (*openapi3.T, error) {
	data, err := readLocation(location)
	if err != nil {
		return nil, err
	}

	var version struct {
		Swagger string `json:"swagger" yaml:"swagger"`
	}

	_ = yaml.Unmarshal(data, &version)

	if strings.HasPrefix(version.Swagger, "2") {
		// invopop/yaml honours the json tags on openapi2.T (yaml.v3 does not); it reads JSON too.
		var v2 openapi2.T
		if err := yaml.Unmarshal(data, &v2); err != nil {
			return nil, fmt.Errorf("swagger 2 spec: %w", err)
		}

		return openapi2conv.ToV3(&v2)
	}

	loader := openapi3.NewLoader()

	return loader.LoadFromData(data)
}

func readLocation(location string) ([]byte, error) {
	if !strings.HasPrefix(location, "http://") && !strings.HasPrefix(location, "https://") {
		return os.ReadFile(location) //nolint:gosec
	}

	client := &http.Client{Timeout: 2 * time.Minute}

	resp, err := client.Get(location) //nolint:noctx
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", location, resp.Status)
	}

	return io.ReadAll(resp.Body)
}

// specReadOnly is only the spec's own word: true when it says readOnly, unknown (nil) otherwise.
func specReadOnly(p property) *bool {
	if p.ReadOnly {
		return boolPtr(true)
	}

	return nil
}

func boolPtr(b bool) *bool { return &b }

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}

	return false
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}
