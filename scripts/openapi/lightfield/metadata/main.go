package main

import (
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/goutils"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api/filters"
	mapping "github.com/amp-labs/connectors/scripts/openapi/internal/api/map"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api/map/mappingfields"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api/output"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api/pipeline"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api/reducers"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api/spec"
	"github.com/amp-labs/connectors/scripts/openapi/lightfield/internal/files"
	"github.com/getkin/kin-openapi/openapi3"
)

func main() {
	extractor, err := files.OpenApiFile.Extractor()
	goutils.MustBeNil(err)

	getSchemas, err := extractor.ExtractListSchemas(api.GET,
		api.List.WithArrayLocator(api.ArrayLocationAtData),
		api.List.WithPropertyFlattener(func(objectName, fieldName string) bool {
			// Read promotes every entry of the "fields" and "relationships" maps to a top-level field.
			return fieldName == "fields" || fieldName == "relationships"
		}),
	)
	goutils.MustBeNil(err)

	// The objects this connector reads; every other list endpoint is out of scope.
	readObjects := filters.NewRules(
		"/v1/accounts",
		"/v1/contacts",
		"/v1/opportunities",
		"/v1/members",
	)

	pipe := pipeline.NewSchemaPipe(getSchemas).
		Filter(filters.KeepWithPath(filters.AndPathMatcher{
			filters.NoIdPath{}, filters.NewAllowPathStrategy(readObjects),
		})).
		Map(mapping.RemoveUrlPrefix("/v1")).
		Reduce(reducers.ShortestNameFromUrlWithFullFallback).
		Map(mapping.DisplayNameFromObjectName).
		Map(mapping.DisplayNameFormat(mapping.CamelCaseToSpaceSeparated, mapping.CapitalizeFirstLetterEveryWord)).
		Map(mapping.Fields(
			describeReadValue,
			mappingfields.DisplayNameFormat(fieldDisplayName),
		))

	output.PrintUnusedRules("readObjects", readObjects)
	goutils.MustBeNil(output.WriteMetadata(files.OutputDir, pipe))
	goutils.MustBeNil(output.WriteQueryParamStats(files.OutputDir, pipe))
	goutils.MustBeNil(output.WriteEndpoints(files.OutputDir, &pipe, nil, nil, nil))
}

// describeReadValue types each field by the value Read returns for it.
// Read unwraps a typed field ({value, valueType}) to its value and a relationship
// ({cardinality, objectType, values}) to its list of ids.
func describeReadValue(field spec.Field) spec.Field {
	switch field.Name {
	case "createdAt", "updatedAt":
		field.ValueType = common.ValueTypeDateTime

		return field
	}

	property := propertySchema(field)
	if property == nil {
		return field
	}

	if nullableString(property) {
		field.ValueType = common.ValueTypeString

		return field
	}

	if valueType, ok := singleEnum(property.Properties["valueType"]); ok {
		field.Type = valueType
		field.ValueType = common.ValueTypeOther

		if value := property.Properties["value"]; value != nil && nullableString(value.Value) {
			field.ValueType = common.ValueTypeString
		}

		return field
	}

	if cardinality, ok := singleEnum(property.Properties["cardinality"]); ok {
		field.Type = cardinality
		field.ValueType = common.ValueTypeReference
	}

	return field
}

// fieldDisplayName turns "$profileImage" into "Profile Image".
func fieldDisplayName(_, name string) string {
	return mapping.CapitalizeFirstLetterEveryWord(
		mapping.CamelCaseToSpaceSeparated(strings.TrimPrefix(name, "$")),
	)
}

func propertySchema(field spec.Field) *openapi3.Schema {
	if field.Origin.Schema == nil {
		return nil
	}

	ref := field.Origin.Schema.Properties[field.Name]
	if ref == nil {
		return nil
	}

	return ref.Value
}

// nullableString is a string, or anyOf string and null.
func nullableString(schema *openapi3.Schema) bool {
	if schema == nil {
		return false
	}

	if schema.Type != nil {
		return schema.Type.Is(openapi3.TypeString)
	}

	if len(schema.AnyOf) == 0 {
		return false
	}

	for _, option := range schema.AnyOf {
		if option.Value == nil || option.Value.Type == nil {
			return false
		}

		if !option.Value.Type.Is(openapi3.TypeString) && !option.Value.Type.Is(openapi3.TypeNull) {
			return false
		}
	}

	return true
}

func singleEnum(ref *openapi3.SchemaRef) (string, bool) {
	if ref == nil || ref.Value == nil || len(ref.Value.Enum) != 1 {
		return "", false
	}

	value, ok := ref.Value.Enum[0].(string)

	return value, ok
}
