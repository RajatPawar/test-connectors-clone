package clarify

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/datautils"
)

// listObjectMetadata describes person, company, deal, meeting and task from
// GET /v1/workspaces/{workspace}/schemas, fetched once per call: each object's JSON Schema is the entry
// whose id ends in /entities/{object}, so workspace custom fields are included.
// https://developer.clarify.ai/docs/api-reference/schemas/getSchemas
//
// user is not a schema entity read through /resources but the fixed shape of GET /users, so its fields
// are static (see userFields).
func (c *Connector) listObjectMetadata(
	ctx context.Context, objectNames []string,
) (*common.ListObjectMetadataResult, error) {
	result := common.NewListObjectMetadataResult()

	var (
		schemas    *schemasResponse
		schemasErr error
	)

	for _, objectName := range datautils.NewSetFromList(objectNames).List() {
		switch {
		case objectName == objectNameUser:
			result.Result[objectName] = *common.NewObjectMetadata("Users", userFields())
		case !supportedObjectsSet.Has(objectName):
			result.AppendError(objectName, common.ErrObjectNotSupported)
		default:
			if schemas == nil && schemasErr == nil {
				schemas, schemasErr = c.fetchSchemas(ctx)
			}

			if schemasErr != nil {
				result.AppendError(objectName, schemasErr)

				continue
			}

			entity, ok := schemas.entity(objectName)
			if !ok {
				result.AppendError(objectName,
					fmt.Errorf("%w: no schema for entity %q", common.ErrMissingExpectedValues, objectName))

				continue
			}

			result.Result[objectName] = *entity.objectMetadata()
		}
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
	Properties           map[string]property  `json:"properties"`
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

	metadata := common.NewObjectMetadata(displayName, systemFields())
	required := datautils.NewSetFromList(s.Required)

	// Schema properties take precedence over the system fields of the same name.
	for name, prop := range s.Properties {
		metadata.AddFieldMetadata(name, prop.fieldMetadata(name, required.Has(name)))
	}

	return metadata
}

// systemFields are on every record: the JSON:API id at the root, and the `_`-prefixed attributes
// shown in the resources example and the schemas guide. They are listed even when a schema omits them.
// https://developer.clarify.ai/docs/api-reference/resources/getResources
// https://developer.clarify.ai/docs/guides/schemas-and-relationships#user-identifiers
func systemFields() common.FieldsMetadata {
	return common.FieldsMetadata{
		"id":          readOnlyField("ID", common.ValueTypeString, "string"),
		"_id":         readOnlyField("Record ID", common.ValueTypeString, "uuid"),
		"_created_at": readOnlyField("Created at", common.ValueTypeDateTime, "date-time"),
		"_updated_at": readOnlyField("Updated at", common.ValueTypeDateTime, "date-time"),
		"_created_by": readOnlyField("Created by", common.ValueTypeString, "string"),
		"_updated_by": readOnlyField("Updated by", common.ValueTypeString, "string"),
	}
}

// userFields are the attributes of a GET /users record. `is_service_account` is not in the reference
// example but is returned by the API.
// https://developer.clarify.ai/docs/api-reference/users/getUsers
func userFields() common.FieldsMetadata {
	return common.FieldsMetadata{
		"id":                 readOnlyField("ID", common.ValueTypeString, "string"),
		"_id":                readOnlyField("User ID", common.ValueTypeString, "string"),
		"name":               readOnlyField("Name", common.ValueTypeOther, "object"),
		"roles":              readOnlyField("Roles", common.ValueTypeOther, "object"),
		"email":              readOnlyField("Email", common.ValueTypeString, "string"),
		"profile_picture":    readOnlyField("Profile picture", common.ValueTypeString, "string"),
		"is_service_account": readOnlyField("Is service account", common.ValueTypeBoolean, "boolean"),
		"last_active_at":     readOnlyField("Last active at", common.ValueTypeDateTime, "date-time"),
		"_created_at":        readOnlyField("Created at", common.ValueTypeDateTime, "date-time"),
		"_updated_at":        readOnlyField("Updated at", common.ValueTypeDateTime, "date-time"),
	}
}

func readOnlyField(displayName string, valueType common.ValueType, providerType string) common.FieldMetadata {
	return common.FieldMetadata{
		DisplayName:  displayName,
		ValueType:    valueType,
		ProviderType: providerType,
		ReadOnly:     new(true),
	}
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

// providerType is the JSON Schema format when there is one (date-time, uuid, markdown...), else the type.
func (p property) providerType() string {
	if p.Format != "" {
		return p.Format
	}

	return p.Type.name()
}

// valueType maps a JSON Schema property to an Ampersand value type.
// A relationship is a record id with xClarifyRelationship. A collection field is an object holding an
// `items` array (https://developer.clarify.ai/docs/getting-started/key-concepts#collection-fields).
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
		return p.multiSelect()
	case "object":
		if items, ok := p.Properties["items"]; ok {
			return items.multiSelect()
		}

		return common.ValueTypeOther, nil
	default:
		return common.ValueTypeOther, nil
	}
}

// multiSelect types an array whose items are an enum; any other array is other.
func (p property) multiSelect() (common.ValueType, []common.FieldValue) {
	if p.Items != nil && len(p.Items.Enum) != 0 {
		return common.ValueTypeMultiSelect, enumValues(p.Items.Enum)
	}

	return common.ValueTypeOther, nil
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
