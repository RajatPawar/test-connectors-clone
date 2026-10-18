package clarify

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/naming"
)

// listObjectMetadata describes the built-in objects from GET /v1/workspaces/{workspace}/schemas, which
// returns every schema of the workspace (built-in and custom fields included) in one cursor-paginated list.
// It is fetched once per call, and only when a resource object is requested.
// GET /schemas does not describe users, so their fields come from the List users reference.
// https://developer.clarify.ai/docs/api-reference/schemas/getSchemas
func (c *Connector) listObjectMetadata(
	ctx context.Context, objectNames []string,
) (*common.ListObjectMetadataResult, error) {
	result := common.NewListObjectMetadataResult()

	var (
		schemas    map[string]entitySchema
		schemasErr error
	)

	for _, objectName := range objectNames {
		switch {
		case objectName == objectNameUsers:
			result.Result[objectName] = *common.NewObjectMetadata("Users", usersFields())
		case resourceObjects.Has(objectName):
			if schemas == nil && schemasErr == nil {
				schemas, schemasErr = c.fetchSchemas(ctx)
			}

			if schemasErr != nil {
				result.AppendError(objectName, schemasErr)

				continue
			}

			entity, ok := schemas[objectName]
			if !ok {
				result.AppendError(objectName, fmt.Errorf("%w: no schema for %s", common.ErrMissingMetadata, objectName))

				continue
			}

			result.Result[objectName] = *entity.objectMetadata(objectName)
		default:
			result.AppendError(objectName, common.ErrObjectNotSupported)
		}
	}

	return result, nil
}

// fetchSchemas returns the entity schemas keyed by object name, following links.next.
// Entities are identified by their schema URL, .../entities/{object}; the .../core/* entries are
// shared base definitions and are skipped.
// https://developer.clarify.ai/docs/guides/schemas-and-relationships#schema-fields
func (c *Connector) fetchSchemas(ctx context.Context) (map[string]entitySchema, error) {
	schemasURL, err := c.workspaceURL("schemas")
	if err != nil {
		return nil, err
	}

	schemas := make(map[string]entitySchema)
	next := schemasURL.String()

	for next != "" {
		response, err := c.JSONHTTPClient().Get(ctx, next)
		if err != nil {
			return nil, err
		}

		page, err := common.UnmarshalJSON[schemasResponse](response)
		if err != nil {
			return nil, err
		}

		if page == nil {
			return nil, common.ErrEmptyJSONHTTPResponse
		}

		for _, resource := range page.Data {
			if objectName, ok := entityName(resource.ID); ok {
				schemas[objectName] = resource.Attributes
			}
		}

		next = ""
		if page.Links.Next != nil {
			next = *page.Links.Next
		}
	}

	return schemas, nil
}

func entityName(schemaID string) (string, bool) {
	const marker = "/entities/"

	index := strings.LastIndex(schemaID, marker)
	if index < 0 {
		return "", false
	}

	return schemaID[index+len(marker):], true
}

type schemasResponse struct {
	Links struct {
		Next *string `json:"next"`
	} `json:"links"`
	Data []schemaResource `json:"data"`
}

type schemaResource struct {
	ID         string       `json:"id"`
	Attributes entitySchema `json:"attributes"`
}

type entitySchema struct {
	Title string `json:"title"`
	Label struct {
		Singular string `json:"singular"`
		Plural   string `json:"plural"`
	} `json:"xClarifyLabel"`
	Properties map[string]property `json:"properties"`
}

type property struct {
	Type         schemaType    `json:"type"`
	Title        string        `json:"title"`
	Format       string        `json:"format"`
	Enum         []any         `json:"enum"`
	Items        *property     `json:"items"`
	ReadOnly     *bool         `json:"readOnly"`
	Relationship *relationship `json:"xClarifyRelationship"`
}

type relationship struct {
	Kind   string `json:"kind"`
	Entity string `json:"entity"`
	Field  string `json:"field"`
}

// schemaType is a JSON Schema `type`: a single type, or a list such as ["string", "null"].
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

// main is the type without "null".
func (t schemaType) main() string {
	for _, name := range t {
		if name != "null" {
			return name
		}
	}

	return ""
}

func (e entitySchema) objectMetadata(objectName string) *common.ObjectMetadata {
	displayName := e.Label.Plural
	if displayName == "" {
		displayName = e.Title
	}

	if displayName == "" {
		displayName = naming.CapitalizeFirstLetterEveryWord(objectName)
	}

	fields := systemFields()

	for name, prop := range e.Properties {
		fields[name] = prop.fieldMetadata(name)
	}

	return common.NewObjectMetadata(displayName, fields)
}

func (p property) fieldMetadata(name string) common.FieldMetadata {
	valueType, values := p.valueType()

	field := common.FieldMetadata{
		DisplayName:  p.Title,
		ValueType:    valueType,
		ProviderType: p.providerType(),
		ReadOnly:     p.ReadOnly,
		Values:       values,
	}

	if field.DisplayName == "" {
		field.DisplayName = name
	}

	if p.Relationship != nil && p.Relationship.Entity != "" {
		field.ReferenceTo = []string{p.Relationship.Entity}
	}

	return field
}

func (p property) providerType() string {
	if p.Format != "" {
		return p.Type.main() + ":" + p.Format
	}

	return p.Type.main()
}

// valueType maps a JSON Schema property to an Ampersand value type.
// Collection fields such as email_addresses are objects holding an `items` array, and composite
// fields such as a person's name are objects, so both are reported as other.
// https://developer.clarify.ai/docs/getting-started/key-concepts#collection-fields
func (p property) valueType() (common.ValueType, []common.FieldValue) {
	if p.Relationship != nil {
		return common.ValueTypeReference, nil
	}

	switch p.Type.main() {
	case "string":
		if values := enumValues(p.Enum); len(values) != 0 {
			return common.ValueTypeSingleSelect, values
		}

		switch p.Format {
		case "date-time":
			return common.ValueTypeDateTime, nil
		case "date":
			return common.ValueTypeDate, nil
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
		if p.Items != nil {
			if values := enumValues(p.Items.Enum); len(values) != 0 {
				return common.ValueTypeMultiSelect, values
			}
		}

		return common.ValueTypeOther, nil
	default:
		return common.ValueTypeOther, nil
	}
}

// Enum values are case-sensitive and are both the stored value and its label. A nullable enum lists null.
func enumValues(enum []any) []common.FieldValue {
	values := make([]common.FieldValue, 0, len(enum))

	for _, value := range enum {
		if text, ok := value.(string); ok {
			values = append(values, common.FieldValue{Value: text, DisplayValue: text})
		}
	}

	return values
}

// systemFields are on every record: the JSON:API `id` and `type`, and the `_`-prefixed system fields.
// TODO: confirm against the live schemas response whether entity schemas already list the system fields.
// https://developer.clarify.ai/docs/api-reference/resources/getResources
// https://developer.clarify.ai/docs/guides/schemas-and-relationships#user-identifiers
func systemFields() common.FieldsMetadata {
	readOnly := true

	return common.FieldsMetadata{
		"id":          {DisplayName: "ID", ValueType: common.ValueTypeString, ProviderType: "string", ReadOnly: &readOnly},
		"type":        {DisplayName: "Type", ValueType: common.ValueTypeString, ProviderType: "string", ReadOnly: &readOnly},
		"_id":         {DisplayName: "ID", ValueType: common.ValueTypeString, ProviderType: "string", ReadOnly: &readOnly},
		"_created_at": {DisplayName: "Created At", ValueType: common.ValueTypeDateTime, ProviderType: "string:date-time", ReadOnly: &readOnly},
		"_updated_at": {DisplayName: "Updated At", ValueType: common.ValueTypeDateTime, ProviderType: "string:date-time", ReadOnly: &readOnly},
		"_created_by": {DisplayName: "Created By", ValueType: common.ValueTypeString, ProviderType: "string", ReadOnly: &readOnly},
		"_updated_by": {DisplayName: "Updated By", ValueType: common.ValueTypeString, ProviderType: "string", ReadOnly: &readOnly},
	}
}

// usersFields are the user attributes of the List users response; GET /schemas has no user schema.
// https://developer.clarify.ai/docs/api-reference/users/getUsers
func usersFields() common.FieldsMetadata {
	readOnly := true

	return common.FieldsMetadata{
		"id":              {DisplayName: "ID", ValueType: common.ValueTypeString, ProviderType: "string", ReadOnly: &readOnly},
		"type":            {DisplayName: "Type", ValueType: common.ValueTypeString, ProviderType: "string", ReadOnly: &readOnly},
		"_id":             {DisplayName: "ID", ValueType: common.ValueTypeString, ProviderType: "string", ReadOnly: &readOnly},
		"name":            {DisplayName: "Name", ValueType: common.ValueTypeOther, ProviderType: "object", ReadOnly: &readOnly},
		"roles":           {DisplayName: "Roles", ValueType: common.ValueTypeOther, ProviderType: "object", ReadOnly: &readOnly},
		"email":           {DisplayName: "Email", ValueType: common.ValueTypeString, ProviderType: "string", ReadOnly: &readOnly},
		"profile_picture": {DisplayName: "Profile Picture", ValueType: common.ValueTypeString, ProviderType: "string", ReadOnly: &readOnly},
		"last_active_at":  {DisplayName: "Last Active At", ValueType: common.ValueTypeDateTime, ProviderType: "string:date-time", ReadOnly: &readOnly},
		"_created_at":     {DisplayName: "Created At", ValueType: common.ValueTypeDateTime, ProviderType: "string:date-time", ReadOnly: &readOnly},
		"_updated_at":     {DisplayName: "Updated At", ValueType: common.ValueTypeDateTime, ProviderType: "string:date-time", ReadOnly: &readOnly},
	}
}
