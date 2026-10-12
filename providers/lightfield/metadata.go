package lightfield

import (
	"context"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/providers/lightfield/internal/metadata"
)

// ListObjectMetadata starts from the static catalog, which holds the top-level record properties
// (and the fixed member fields), then adds every system and custom field and relationship
// returned by the object's definitions endpoint.
// https://docs.lightfield.app/using-the-api/fields-and-relationships#discovering-definitions
func (c *Connector) ListObjectMetadata(
	ctx context.Context,
	objectNames []string,
) (*common.ListObjectMetadataResult, error) {
	result, err := metadata.Catalog.SelectResult(objectNames)
	if err != nil {
		return nil, err
	}

	for objectName, object := range result.Result {
		if !objectsWithDefinitions.Has(objectName) {
			continue
		}

		definitions, err := c.fetchDefinitions(ctx, objectName)
		if err != nil {
			delete(result.Result, objectName)
			result.AppendError(objectName, err)

			continue
		}

		definitions.addTo(&object)
		result.Result[objectName] = object
	}

	return result, nil
}

func (c *Connector) fetchDefinitions(ctx context.Context, objectName string) (*definitionsResponse, error) {
	url, err := c.objectURL(objectName, "definitions")
	if err != nil {
		return nil, err
	}

	response, err := c.JSONHTTPClient().Get(ctx, url.String(), common.Header{
		Key:   apiVersionHeader,
		Value: apiVersion,
	})
	if err != nil {
		return nil, err
	}

	return common.UnmarshalJSON[definitionsResponse](response)
}

type definitionsResponse struct {
	FieldDefinitions        map[string]fieldDefinition        `json:"fieldDefinitions"`
	RelationshipDefinitions map[string]relationshipDefinition `json:"relationshipDefinitions"`
}

type fieldDefinition struct {
	ID                *string `json:"id"`
	Label             string  `json:"label"`
	ValueType         string  `json:"valueType"`
	ReadOnly          *bool   `json:"readOnly"`
	TypeConfiguration struct {
		Options []selectOption `json:"options"`
	} `json:"typeConfiguration"`
}

type selectOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type relationshipDefinition struct {
	ID          *string `json:"id"`
	Label       string  `json:"label"`
	Cardinality string  `json:"cardinality"`
	ObjectType  string  `json:"objectType"`
}

// Related object types that map onto objects this connector reads.
var objectTypeToObjectName = map[string]string{ // nolint:gochecknoglobals
	"account":     objectNameAccounts,
	"contact":     objectNameContacts,
	"opportunity": objectNameOpportunities,
}

// addTo mirrors flattenRecord: properties already present win over fields, fields over relationships.
func (d definitionsResponse) addTo(object *common.ObjectMetadata) {
	for key, definition := range d.FieldDefinitions {
		if _, taken := object.Fields[key]; taken {
			continue
		}

		object.AddFieldMetadata(key, common.FieldMetadata{
			DisplayName:  definition.Label,
			ValueType:    valueTypeOf(definition.ValueType),
			ProviderType: definition.ValueType,
			ReadOnly:     definition.ReadOnly,
			IsCustom:     isCustom(key),
			Values:       definition.values(),
			FieldId:      definition.ID,
		})
	}

	for key, definition := range d.RelationshipDefinitions {
		if _, taken := object.Fields[key]; taken {
			continue
		}

		var referenceTo []string
		if objectName, ok := objectTypeToObjectName[definition.ObjectType]; ok {
			referenceTo = []string{objectName}
		}

		object.AddFieldMetadata(key, common.FieldMetadata{
			DisplayName:  definition.Label,
			ValueType:    common.ValueTypeReference,
			ProviderType: definition.Cardinality,
			IsCustom:     isCustom(key),
			ReferenceTo:  referenceTo,
			FieldId:      definition.ID,
		})
	}
}

func (d fieldDefinition) values() []common.FieldValue {
	switch d.ValueType {
	case "SINGLE_SELECT", "MULTI_SELECT":
	default:
		return nil
	}

	values := make([]common.FieldValue, len(d.TypeConfiguration.Options))
	for index, option := range d.TypeConfiguration.Options {
		values[index] = common.FieldValue{
			Value:        option.ID,
			DisplayValue: option.Label,
		}
	}

	return values
}

// System keys start with "$"; custom fields and relationships use their bare slug.
// https://docs.lightfield.app/using-the-api/fields-and-relationships#field-keys
func isCustom(key string) *bool {
	return new(!strings.HasPrefix(key, "$"))
}

// valueTypeOf maps a Lightfield value type to the value Read returns for it.
// https://docs.lightfield.app/using-the-api/fields-and-relationships#field-value-types
func valueTypeOf(valueType string) common.ValueType {
	switch valueType {
	case "TEXT", "MARKDOWN", "READONLY_MARKDOWN", "HTML", "SOCIAL_HANDLE":
		return common.ValueTypeString
	case "NUMBER", "CURRENCY":
		return common.ValueTypeFloat
	case "CHECKBOX":
		return common.ValueTypeBoolean
	case "DATETIME":
		return common.ValueTypeDateTime
	case "SINGLE_SELECT":
		return common.ValueTypeSingleSelect
	case "MULTI_SELECT":
		return common.ValueTypeMultiSelect
	default:
		// EMAIL, URL and TELEPHONE may hold a list; ADDRESS and FULL_NAME are objects.
		return common.ValueTypeOther
	}
}
