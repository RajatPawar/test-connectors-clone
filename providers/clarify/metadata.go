package clarify

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/datautils"
)

// listObjectMetadata describes objects from GET /v1/workspaces/{workspace}/schemas, fetched once per call.
// Each object's JSON Schema is the entry whose id ends in /entities/{object}.
// https://developer.clarify.ai/docs/api-reference/schemas/getSchemas
func (c *Connector) listObjectMetadata(
	ctx context.Context, objectNames []string,
) (*common.ListObjectMetadataResult, error) {
	result := common.NewListObjectMetadataResult()

	schemas, err := c.fetchSchemas(ctx)
	if err != nil {
		return nil, err
	}

	for _, objectName := range datautils.NewSetFromList(objectNames).List() {
		if !supportedObjectsSet.Has(objectName) {
			result.AppendError(objectName, common.ErrObjectNotSupported)

			continue
		}

		entity, ok := schemas.entity(objectName)
		if !ok {
			result.AppendError(objectName, common.ErrObjectNotSupported)

			continue
		}

		result.Result[objectName] = *entity.objectMetadata()
	}

	return result, nil
}

var supportedObjectsSet = datautils.NewSetFromList(supportedObjects) // nolint:gochecknoglobals

func (c *Connector) fetchSchemas(ctx context.Context) (*schemasResponse, error) {
	schemasURL, err := c.workspaceURL("schemas")
	if err != nil {
		return nil, err
	}

	response, err := c.JSONHTTPClient().Get(ctx, schemasURL.String())
	if err != nil {
		return nil, err
	}

	schemas, err := common.UnmarshalJSON[schemasResponse](response)
	if err != nil {
		return nil, err
	}

	if schemas == nil {
		return nil, common.ErrEmptyJSONHTTPResponse
	}

	return schemas, nil
}

type schemasResponse struct {
	Data []schemaResource `json:"data"`
}

type schemaResource struct {
	ID         string     `json:"id"`
	Attributes jsonSchema `json:"attributes"`
}

type jsonSchema struct {
	Title         string              `json:"title"`
	XClarifyLabel *clarifyLabel       `json:"xClarifyLabel"`
	Properties    map[string]property `json:"properties"`
	Required      []string            `json:"required"`
}

type clarifyLabel struct {
	Singular string `json:"singular"`
	Plural   string `json:"plural"`
}

type property struct {
	Type                 schemaType           `json:"type"`
	Title                string               `json:"title"`
	Format               string               `json:"format"`
	Enum                 []any                `json:"enum"`
	Items                *property            `json:"items"`
	ReadOnly             *bool                `json:"readOnly"`
	XClarifyRelationship *clarifyRelationship `json:"xClarifyRelationship"`
}

type clarifyRelationship struct {
	Kind   string `json:"kind"`
	Entity string `json:"entity"`
	Field  string `json:"field"`
}

// schemaType is the JSON Schema `type`: a single name or a list such as ["string", "null"].
type schemaType []string

func (t *schemaType) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*t = schemaType{single}

		return nil
	}

	var list []string
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}

	*t = list

	return nil
}

// name is the first non-null type.
func (t schemaType) name() string {
	for _, name := range t {
		if name != "null" {
			return name
		}
	}

	return ""
}

func (r *schemasResponse) entity(objectName string) (*jsonSchema, bool) {
	suffix := "/entities/" + objectName

	for index := range r.Data {
		if strings.HasSuffix(r.Data[index].ID, suffix) {
			return &r.Data[index].Attributes, true
		}
	}

	return nil, false
}

func (s *jsonSchema) objectMetadata() *common.ObjectMetadata {
	displayName := s.Title
	if s.XClarifyLabel != nil && s.XClarifyLabel.Plural != "" {
		displayName = s.XClarifyLabel.Plural
	}

	metadata := common.NewObjectMetadata(displayName, common.FieldsMetadata{
		// The JSON:API resource id, at the root of every record.
		"id": {
			DisplayName:  "ID",
			ValueType:    common.ValueTypeString,
			ProviderType: "string",
			ReadOnly:     new(true),
		},
	})

	required := datautils.NewSetFromList(s.Required)

	for name, prop := range s.Properties {
		metadata.AddFieldMetadata(name, prop.fieldMetadata(name, required.Has(name)))
	}

	return metadata
}

func (p property) fieldMetadata(name string, required bool) common.FieldMetadata {
	displayName := p.Title
	if displayName == "" {
		displayName = name
	}

	valueType, values := p.valueType()

	field := common.FieldMetadata{
		DisplayName:  displayName,
		ValueType:    valueType,
		ProviderType: p.providerType(),
		ReadOnly:     p.ReadOnly,
		IsRequired:   new(required),
		Values:       values,
	}

	if p.XClarifyRelationship != nil && p.XClarifyRelationship.Entity != "" {
		field.ReferenceTo = []string{p.XClarifyRelationship.Entity}
	}

	return field
}

func (p property) providerType() string {
	if p.Format != "" {
		return p.Type.name() + ":" + p.Format
	}

	return p.Type.name()
}

// valueType maps a JSON Schema property to an Ampersand value type.
// Relationship ids are strings with xClarifyRelationship; collection fields are objects holding `items`.
// https://developer.clarify.ai/docs/getting-started/key-concepts#collection-fields
func (p property) valueType() (common.ValueType, []common.FieldValue) { // nolint:cyclop
	if p.XClarifyRelationship != nil {
		return common.ValueTypeReference, nil
	}

	if len(p.Enum) != 0 {
		return common.ValueTypeSingleSelect, enumValues(p.Enum)
	}

	switch p.Type.name() {
	case "string":
		switch p.Format {
		case "date":
			return common.ValueTypeDate, nil
		case "date-time":
			return common.ValueTypeDateTime, nil
		default:
			return common.ValueTypeString, nil
		}
	case "integer":
		return common.ValueTypeInt, nil
	case "number":
		return common.ValueTypeFloat, nil
	case "boolean":
		return common.ValueTypeBoolean, nil
	case "array":
		if p.Items != nil && len(p.Items.Enum) != 0 {
			return common.ValueTypeMultiSelect, enumValues(p.Items.Enum)
		}

		return common.ValueTypeOther, nil
	default:
		return common.ValueTypeOther, nil
	}
}

func enumValues(enum []any) []common.FieldValue {
	values := make([]common.FieldValue, 0, len(enum))

	for _, option := range enum {
		text, ok := option.(string)
		if !ok {
			continue
		}

		values = append(values, common.FieldValue{Value: text, DisplayValue: text})
	}

	return values
}
