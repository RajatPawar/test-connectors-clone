package lightfield

import (
	"context"
	"maps"
	"net/http"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/naming"
	"github.com/amp-labs/connectors/common/urlbuilder"
)

// listObjectMetadata describes accounts, contacts and opportunities from their definitions endpoint,
// which includes custom fields and select options. Members have no definitions endpoint,
// so their schema is the one documented for the member object.
func (c *Connector) listObjectMetadata(
	ctx context.Context, objectNames []string,
) (*common.ListObjectMetadataResult, error) {
	result := &common.ListObjectMetadataResult{
		Result: make(map[string]common.ObjectMetadata),
		Errors: make(map[string]error),
	}

	described := make([]string, 0, len(objectNames))

	for _, objectName := range objectNames {
		switch {
		case objectName == objectMembers:
			result.Result[objectName] = *membersMetadata()
		case objectsWithDefinitions.Has(objectName):
			described = append(described, objectName)
		default:
			result.Errors[objectName] = common.ErrObjectNotSupported
		}
	}

	if len(described) == 0 {
		return result, nil
	}

	definitions, err := c.definitions.ListObjectMetadata(ctx, described)
	if err != nil {
		return nil, err
	}

	maps.Copy(result.Result, definitions.Result)
	maps.Copy(result.Errors, definitions.Errors)

	return result, nil
}

// https://docs.lightfield.app/api/resources/account/methods/definitions
func (c *Connector) buildDefinitionsRequest(ctx context.Context, objectName string) (*http.Request, error) {
	url, err := urlbuilder.New(c.ProviderInfo().BaseURL, apiVersion, objectName, "definitions")
	if err != nil {
		return nil, err
	}

	return newGetRequest(ctx, url.String())
}

func (c *Connector) parseDefinitionsResponse(
	ctx context.Context,
	objectName string,
	request *http.Request,
	response *common.JSONHTTPResponse,
) (*common.ObjectMetadata, error) {
	definitions, err := common.UnmarshalJSON[definitionsResponse](response)
	if err != nil {
		return nil, err
	}

	if definitions == nil {
		return nil, common.ErrEmptyJSONHTTPResponse
	}

	fields := make(common.FieldsMetadata)

	for key, relationship := range definitions.RelationshipDefinitions {
		fields[key] = relationship.fieldMetadata()
	}

	for key, field := range definitions.FieldDefinitions {
		fields[key] = field.fieldMetadata()
	}

	maps.Copy(fields, envelopeFields())
	fields["externalId"] = common.FieldMetadata{
		DisplayName:  "External ID",
		ValueType:    common.ValueTypeString,
		ProviderType: "string",
	}

	return common.NewObjectMetadata(naming.CapitalizeFirstLetterEveryWord(objectName), fields), nil
}

type definitionsResponse struct {
	FieldDefinitions        map[string]fieldDefinition        `json:"fieldDefinitions"`
	RelationshipDefinitions map[string]relationshipDefinition `json:"relationshipDefinitions"`
}

// https://docs.lightfield.app/using-the-api/fields-and-relationships#field-definition-object
type fieldDefinition struct {
	ID                *string           `json:"id"`
	Label             string            `json:"label"`
	ValueType         string            `json:"valueType"`
	System            *bool             `json:"system"`
	ReadOnly          *bool             `json:"readOnly"`
	TypeConfiguration typeConfiguration `json:"typeConfiguration"`
}

type typeConfiguration struct {
	MultipleValues *bool          `json:"multipleValues"`
	Options        []selectOption `json:"options"`
}

type selectOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// https://docs.lightfield.app/using-the-api/fields-and-relationships#relationship-definition-object
type relationshipDefinition struct {
	ID          *string `json:"id"`
	Label       string  `json:"label"`
	Cardinality string  `json:"cardinality"`
	ObjectType  string  `json:"objectType"`
	System      *bool   `json:"system"`
}

func (d fieldDefinition) fieldMetadata() common.FieldMetadata {
	valueType := fieldValueType(d.ValueType, d.TypeConfiguration.MultipleValues)

	var values []common.FieldValue

	if valueType == common.ValueTypeSingleSelect || valueType == common.ValueTypeMultiSelect {
		values = make([]common.FieldValue, len(d.TypeConfiguration.Options))
		for index, option := range d.TypeConfiguration.Options {
			values[index] = common.FieldValue{Value: option.ID, DisplayValue: option.Label}
		}
	}

	return common.FieldMetadata{
		DisplayName:  d.Label,
		ValueType:    valueType,
		ProviderType: d.ValueType,
		ReadOnly:     d.ReadOnly,
		IsCustom:     isCustom(d.System),
		FieldId:      d.ID,
		Values:       values,
	}
}

func (d relationshipDefinition) fieldMetadata() common.FieldMetadata {
	var referenceTo []string
	if object, ok := relatedObjects[d.ObjectType]; ok {
		referenceTo = []string{object}
	}

	return common.FieldMetadata{
		DisplayName:  d.Label,
		ValueType:    common.ValueTypeReference,
		ProviderType: d.Cardinality,
		IsCustom:     isCustom(d.System),
		FieldId:      d.ID,
		ReferenceTo:  referenceTo,
	}
}

// relatedObjects maps a relationship's objectType to the object this connector reads.
var relatedObjects = map[string]string{ // nolint:gochecknoglobals
	"account":     objectAccounts,
	"contact":     objectContacts,
	"opportunity": objectOpportunities,
	"member":      objectMembers,
}

// fieldValueType maps a Lightfield valueType to an Ampersand ValueType.
// EMAIL and URL values are arrays unless the field is configured for a single value,
// and TELEPHONE is always an array.
// https://docs.lightfield.app/using-the-api/fields-and-relationships#field-value-types
func fieldValueType(providerType string, multipleValues *bool) common.ValueType {
	switch providerType {
	case "TEXT", "MARKDOWN", "READONLY_MARKDOWN", "HTML", "SOCIAL_HANDLE":
		return common.ValueTypeString
	case "EMAIL", "URL":
		if multipleValues != nil && !*multipleValues {
			return common.ValueTypeString
		}

		return common.ValueTypeOther
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
	default: // ADDRESS, FULL_NAME, TELEPHONE
		return common.ValueTypeOther
	}
}

func isCustom(system *bool) *bool {
	if system == nil {
		return nil
	}

	custom := !*system

	return &custom
}

// envelopeFields are the record properties outside `fields` and `relationships`.
// https://docs.lightfield.app/api/resources/account/methods/list
func envelopeFields() common.FieldsMetadata {
	return common.FieldsMetadata{
		"id": {
			DisplayName:  "ID",
			ValueType:    common.ValueTypeString,
			ProviderType: "string",
		},
		"createdAt": {
			DisplayName:  "Created At",
			ValueType:    common.ValueTypeDateTime,
			ProviderType: "string",
		},
		"updatedAt": {
			DisplayName:  "Updated At",
			ValueType:    common.ValueTypeDateTime,
			ProviderType: "string",
		},
		"httpLink": {
			DisplayName:  "HTTP Link",
			ValueType:    common.ValueTypeString,
			ProviderType: "string",
		},
	}
}

// https://docs.lightfield.app/api/resources/member/methods/list
func membersMetadata() *common.ObjectMetadata {
	fields := envelopeFields()

	fields["$email"] = common.FieldMetadata{
		DisplayName:  "Email",
		ValueType:    common.ValueTypeString,
		ProviderType: "EMAIL",
	}
	fields["$name"] = common.FieldMetadata{
		DisplayName:  "Name",
		ValueType:    common.ValueTypeOther,
		ProviderType: "FULL_NAME",
	}
	fields["$profileImage"] = common.FieldMetadata{
		DisplayName:  "Profile Image",
		ValueType:    common.ValueTypeString,
		ProviderType: "URL",
	}
	fields["$role"] = common.FieldMetadata{
		DisplayName:  "Role",
		ValueType:    common.ValueTypeString,
		ProviderType: "TEXT",
	}
	fields["$user"] = common.FieldMetadata{
		DisplayName:  "User",
		ValueType:    common.ValueTypeReference,
		ProviderType: "HAS_ONE",
	}

	return common.NewObjectMetadata("Members", fields)
}
