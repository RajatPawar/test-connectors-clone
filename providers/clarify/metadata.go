package clarify

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/datautils"
)

// ListObjectMetadata describes person, company, deal, meeting and task from the workspace's
// GET /v1/workspaces/{workspace}/schemas, fetched once per call. It returns every schema, built-in
// and custom, as JSON Schema; built-in objects are the `.../entities/{object}` entries.
// https://developer.clarify.ai/docs/guides/schemas-and-relationships#schema-fields
// Users have no schema there, so their fields come from the GET /users reference.
func (c *Connector) ListObjectMetadata(
	ctx context.Context, objectNames []string,
) (*common.ListObjectMetadataResult, error) {
	if len(objectNames) == 0 {
		return nil, common.ErrMissingObjects
	}

	result := common.NewListObjectMetadataResult()

	var (
		schemas    map[string]entitySchema
		schemasErr error
		fetched    bool
	)

	for _, objectName := range datautils.NewSetFromList(objectNames).List() {
		switch {
		case objectName == objectNameUsers:
			result.Result[objectName] = *usersMetadata()
		case resourceObjects.Has(objectName):
			if !fetched {
				schemas, schemasErr = c.fetchEntitySchemas(ctx)
				fetched = true
			}

			if schemasErr != nil {
				result.AppendError(objectName, schemasErr)

				continue
			}

			schema, ok := schemas[objectName]
			if !ok {
				result.AppendError(objectName,
					fmt.Errorf("%w: no schema for object %q", common.ErrMissingMetadata, objectName))

				continue
			}

			result.Result[objectName] = *schema.objectMetadata()
		default:
			result.AppendError(objectName, common.ErrObjectNotSupported)
		}
	}

	return result, nil
}

// fetchEntitySchemas returns the entity schemas keyed by object name, following `links.next`.
func (c *Connector) fetchEntitySchemas(ctx context.Context) (map[string]entitySchema, error) {
	schemasURL, err := urlbuilder.New(c.ProviderInfo().BaseURL, apiVersion, "workspaces", c.Workspace, "schemas")
	if err != nil {
		return nil, err
	}

	firstPageURL, err := schemasURL.ToURL()
	if err != nil {
		return nil, err
	}

	schemas := make(map[string]entitySchema)
	nextURL := firstPageURL.String()

	for nextURL != "" {
		response, err := c.JSONHTTPClient().Get(ctx, nextURL)
		if err != nil {
			return nil, err
		}

		page, err := common.UnmarshalJSON[schemasResponse](response)
		if err != nil {
			return nil, err
		}

		for _, item := range page.Data {
			if objectName, ok := entityName(item.ID); ok {
				schemas[objectName] = item.Attributes
			}
		}

		nextURL = ""
		if page.Links.Next != nil && *page.Links.Next != "" {
			nextURL, err = sameOriginLink(*page.Links.Next, firstPageURL)
			if err != nil {
				return nil, err
			}
		}
	}

	return schemas, nil
}

// entityName extracts `person` from `https://getclarify.ai/schemas/entities/person`.
// The `.../core/*` entries are base definitions, not objects.
func entityName(schemaID string) (string, bool) {
	_, name, found := strings.Cut(schemaID, "/schemas/entities/")
	if !found || name == "" || strings.Contains(name, "/") {
		return "", false
	}

	return name, true
}

// https://developer.clarify.ai/docs/api-reference/schemas/getSchemas
type schemasResponse struct {
	Links struct {
		Next *string `json:"next"`
	} `json:"links"`
	Data []struct {
		ID         string       `json:"id"`
		Attributes entitySchema `json:"attributes"`
	} `json:"data"`
}

type entitySchema struct {
	Title         string `json:"title"`
	XClarifyLabel struct {
		Singular string `json:"singular"`
		Plural   string `json:"plural"`
	} `json:"xClarifyLabel"`
	Properties map[string]schemaProperty `json:"properties"`
}

type schemaProperty struct {
	Type                 schemaType          `json:"type"`
	Title                string              `json:"title"`
	Format               string              `json:"format"`
	Enum                 []any               `json:"enum"`
	Items                *schemaProperty     `json:"items"`
	ReadOnly             *bool               `json:"readOnly"`
	XClarifyRelationship *schemaRelationship `json:"xClarifyRelationship"`
}

type schemaRelationship struct {
	Kind   string `json:"kind"`
	Entity string `json:"entity"`
	Field  string `json:"field"`
}

// schemaType is the JSON Schema `type`, either "string" or a list such as ["string", "null"].
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

// primary is the first type that is not "null".
func (t schemaType) primary() string {
	for _, name := range t {
		if name != "null" {
			return name
		}
	}

	return ""
}

func (s entitySchema) objectMetadata() *common.ObjectMetadata {
	metadata := common.NewObjectMetadata(s.displayName(), recordFields())

	for name, property := range s.Properties {
		if !property.isAttribute() {
			continue
		}

		metadata.AddFieldMetadata(name, property.fieldMetadata(name))
	}

	return metadata
}

// isAttribute tells whether Read returns the property among a record's attributes.
// A relationship is an attribute only as a single foreign-key id (e.g. deal.company_id, a string).
// The other side of a relationship, a list of related records such as the people on a deal,
// is not in `attributes`: live deal records carry company_id but no people, and an empty
// `relationships` object, as related records are only returned with `include`.
func (p schemaProperty) isAttribute() bool {
	return p.XClarifyRelationship == nil || p.Type.primary() == "string"
}

func (s entitySchema) displayName() string {
	if s.XClarifyLabel.Plural != "" {
		return s.XClarifyLabel.Plural
	}

	return s.Title
}

func (p schemaProperty) fieldMetadata(name string) common.FieldMetadata {
	displayName := p.Title
	if displayName == "" {
		displayName = name
	}

	field := common.FieldMetadata{
		DisplayName:  displayName,
		ProviderType: p.providerType(),
		ReadOnly:     p.ReadOnly,
	}

	if relationship := p.XClarifyRelationship; relationship != nil {
		field.ValueType = common.ValueTypeReference
		field.ProviderType = relationship.Kind

		if resourceObjects.Has(relationship.Entity) {
			field.ReferenceTo = []string{relationship.Entity}
		}

		return field
	}

	field.ValueType, field.Values = p.valueType()

	return field
}

func (p schemaProperty) providerType() string {
	if p.Format != "" {
		return p.Format
	}

	return p.Type.primary()
}

// valueType maps JSON Schema types to Ampersand ones. Collection fields such as
// email_addresses are objects with an `items` array, so they are reported as other.
// https://developer.clarify.ai/docs/getting-started/key-concepts#collection-fields
func (p schemaProperty) valueType() (common.ValueType, []common.FieldValue) {
	if values := enumValues(p.Enum); len(values) != 0 {
		return common.ValueTypeSingleSelect, values
	}

	switch p.Type.primary() {
	case "string":
		switch p.Format {
		case "date":
			return common.ValueTypeDate, nil
		case "date-time":
			return common.ValueTypeDateTime, nil
		default:
			return common.ValueTypeString, nil
		}
	case "number":
		return common.ValueTypeFloat, nil
	case "integer":
		return common.ValueTypeInt, nil
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

// enumValues keeps the string options; a nullable enum may also list null.
// Values are case-sensitive and are both the stored value and the label.
func enumValues(enum []any) []common.FieldValue {
	values := make([]common.FieldValue, 0, len(enum))

	for _, option := range enum {
		if text, ok := option.(string); ok {
			values = append(values, common.FieldValue{Value: text, DisplayValue: text})
		}
	}

	return values
}

// recordFields are present on every record Read returns, whether or not a schema lists them:
// the JSON:API `id` and `type`, and the `_`-prefixed system attributes, which are read-only.
// https://developer.clarify.ai/docs/api-reference/resources/getResources
// https://developer.clarify.ai/docs/guides/schemas-and-relationships#user-identifiers
// https://developer.clarify.ai/docs/api-basics/errors#422-unprocessable-entity
func recordFields() common.FieldsMetadata {
	return common.FieldsMetadata{
		"id":          systemField("ID", common.ValueTypeString, "string"),
		"type":        systemField("Type", common.ValueTypeString, "string"),
		"_id":         systemField("Record ID", common.ValueTypeString, "string"),
		"_created_at": systemField("Created At", common.ValueTypeDateTime, "date-time"),
		"_updated_at": systemField("Updated At", common.ValueTypeDateTime, "date-time"),
		"_created_by": systemField("Created By", common.ValueTypeString, "string"),
		"_updated_by": systemField("Updated By", common.ValueTypeString, "string"),
	}
}

// usersMetadata lists the user attributes from the GET /users reference response, plus
// `is_service_account`, which live responses carry and the reference omits.
// https://developer.clarify.ai/docs/api-reference/users/getUsers
func usersMetadata() *common.ObjectMetadata {
	return common.NewObjectMetadata("Users", common.FieldsMetadata{
		"id":              systemField("ID", common.ValueTypeString, "string"),
		"type":            systemField("Type", common.ValueTypeString, "string"),
		"_id":             systemField("Record ID", common.ValueTypeString, "string"),
		"_created_at":     systemField("Created At", common.ValueTypeDateTime, "date-time"),
		"_updated_at":     systemField("Updated At", common.ValueTypeDateTime, "date-time"),
		"name":            {DisplayName: "Name", ValueType: common.ValueTypeOther, ProviderType: "object"},
		"roles":           {DisplayName: "Roles", ValueType: common.ValueTypeOther, ProviderType: "object"},
		"email":           {DisplayName: "Email", ValueType: common.ValueTypeString, ProviderType: "string"},
		"profile_picture": {DisplayName: "Profile Picture", ValueType: common.ValueTypeString, ProviderType: "string"},
		"last_active_at":  {DisplayName: "Last Active At", ValueType: common.ValueTypeDateTime, ProviderType: "date-time"},
		"is_service_account": {
			DisplayName: "Is Service Account", ValueType: common.ValueTypeBoolean, ProviderType: "boolean",
		},
	})
}

func systemField(displayName string, valueType common.ValueType, providerType string) common.FieldMetadata {
	return common.FieldMetadata{
		DisplayName:  displayName,
		ValueType:    valueType,
		ProviderType: providerType,
		ReadOnly:     new(true),
	}
}
