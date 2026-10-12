package lightfield

import (
	"context"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/providers/lightfield/internal/metadata"
)

// ListObjectMetadata starts from the static catalog generated from the OpenAPI spec, which holds the
// top-level record properties (and the fixed member fields), then adds each object's standard and
// custom fields and relationships from its definitions endpoint.
// https://docs.lightfield.app/using-the-api/fields-and-relationships#discovering-definitions
func (c *Connector) ListObjectMetadata(
	ctx context.Context,
	objectNames []string,
) (*common.ListObjectMetadataResult, error) {
	selection, err := metadata.Catalog.Select(objectNames)
	if err != nil {
		return nil, err
	}

	result := selection.Result()

	for objectName, objectMetadata := range result.Result {
		if !objectsWithDefinitions.Has(objectName) {
			continue
		}

		definitions, err := c.fetchDefinitions(ctx, objectName)
		if err != nil {
			delete(result.Result, objectName)
			result.AppendError(objectName, err)

			continue
		}

		definitions.addFields(&objectMetadata)
		result.Result[objectName] = objectMetadata
	}

	return result, nil
}

func (c *Connector) fetchDefinitions(ctx context.Context, objectName string) (*definitionsResponse, error) {
	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, apiVersion, objectName, "definitions")
	if err != nil {
		return nil, err
	}

	resp, err := c.JSONHTTPClient().Get(ctx, url.String(), common.Header{
		Key:   versionHeaderName,
		Value: versionHeaderValue,
	})
	if err != nil {
		return nil, err
	}

	return common.UnmarshalJSON[definitionsResponse](resp)
}

// https://docs.lightfield.app/api/resources/account/methods/definitions
type definitionsResponse struct {
	FieldDefinitions        map[string]fieldDefinition        `json:"fieldDefinitions"`
	RelationshipDefinitions map[string]relationshipDefinition `json:"relationshipDefinitions"`
}

type fieldDefinition struct {
	Label             string            `json:"label"`
	ValueType         string            `json:"valueType"`
	ReadOnly          bool              `json:"readOnly"`
	TypeConfiguration typeConfiguration `json:"typeConfiguration"`
}

type typeConfiguration struct {
	Options []selectOption `json:"options"`
}

type selectOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type relationshipDefinition struct {
	Label       string `json:"label"`
	Cardinality string `json:"cardinality"`
	ObjectType  string `json:"objectType"`
}

// addFields adds definitions under the keys Read returns them by (see flattenRecord),
// without replacing a top-level property of the same name.
func (d definitionsResponse) addFields(objectMetadata *common.ObjectMetadata) {
	for key, definition := range d.FieldDefinitions {
		if _, exists := objectMetadata.Fields[key]; !exists {
			objectMetadata.AddFieldMetadata(key, definition.fieldMetadata(key))
		}
	}

	for key, definition := range d.RelationshipDefinitions {
		if _, exists := objectMetadata.Fields[key]; !exists {
			objectMetadata.AddFieldMetadata(key, definition.fieldMetadata(key))
		}
	}
}

func (f fieldDefinition) fieldMetadata(key string) common.FieldMetadata {
	valueType, values := f.valueType()
	readOnly := f.ReadOnly || f.ValueType == "READONLY_MARKDOWN"
	isCustom := isCustomKey(key)

	return common.FieldMetadata{
		DisplayName:  displayName(f.Label, key),
		ValueType:    valueType,
		ProviderType: f.ValueType,
		ReadOnly:     &readOnly,
		IsCustom:     &isCustom,
		Values:       values,
	}
}

// valueType maps a Lightfield field value type to an Ampersand value type.
// EMAIL and URL may hold a string or an array, TELEPHONE is an array, ADDRESS and FULL_NAME are objects.
// https://docs.lightfield.app/using-the-api/fields-and-relationships#field-value-types
func (f fieldDefinition) valueType() (common.ValueType, []common.FieldValue) {
	switch f.ValueType {
	case "TEXT", "SOCIAL_HANDLE", "MARKDOWN", "READONLY_MARKDOWN":
		return common.ValueTypeString, nil
	case "NUMBER", "CURRENCY":
		return common.ValueTypeFloat, nil
	case "CHECKBOX":
		return common.ValueTypeBoolean, nil
	case "DATETIME":
		return common.ValueTypeDateTime, nil
	case "SINGLE_SELECT":
		if values := f.selectValues(); len(values) != 0 {
			return common.ValueTypeSingleSelect, values
		}

		return common.ValueTypeString, nil
	case "MULTI_SELECT":
		if values := f.selectValues(); len(values) != 0 {
			return common.ValueTypeMultiSelect, values
		}

		return common.ValueTypeOther, nil
	default:
		return common.ValueTypeOther, nil
	}
}

// Select values hold option IDs; labels come from the definitions.
// https://docs.lightfield.app/using-the-api/fields-and-relationships#select-options
func (f fieldDefinition) selectValues() []common.FieldValue {
	values := make([]common.FieldValue, 0, len(f.TypeConfiguration.Options))

	for _, option := range f.TypeConfiguration.Options {
		values = append(values, common.FieldValue{
			Value:        option.ID,
			DisplayValue: option.Label,
		})
	}

	return values
}

// relatedObjectNames maps a relationship's objectType to the object this connector reads it as.
var relatedObjectNames = map[string]string{ // nolint:gochecknoglobals
	"account":     objectNameAccounts,
	"contact":     objectNameContacts,
	"opportunity": objectNameOpportunities,
}

// Read returns a relationship as the array of related record IDs.
func (r relationshipDefinition) fieldMetadata(key string) common.FieldMetadata {
	readOnly := false
	isCustom := isCustomKey(key)

	field := common.FieldMetadata{
		DisplayName:  displayName(r.Label, key),
		ValueType:    common.ValueTypeOther,
		ProviderType: r.Cardinality,
		ReadOnly:     &readOnly,
		IsCustom:     &isCustom,
	}

	if objectName, ok := relatedObjectNames[r.ObjectType]; ok {
		field.ValueType = common.ValueTypeReference
		field.ReferenceTo = []string{objectName}
	}

	return field
}

// System fields and relationships are prefixed with "$"; custom ones use a bare slug.
// https://docs.lightfield.app/using-the-api/fields-and-relationships#field-keys
func isCustomKey(key string) bool {
	return !strings.HasPrefix(key, "$")
}

func displayName(label, key string) string {
	if label == "" {
		return key
	}

	return label
}
