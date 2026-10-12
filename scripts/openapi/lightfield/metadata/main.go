package main

import (
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/goutils"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api/filters"
	mapping "github.com/amp-labs/connectors/scripts/openapi/internal/api/map"
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
			// The connector promotes every entry of "fields" and "relationships" to a top-level field.
			return fieldName == "fields" || fieldName == "relationships"
		}),
	)
	goutils.MustBeNil(err)

	// Only the objects the connector reads.
	readObjects := filters.NewRules(
		"/v1/accounts",
		"/v1/contacts",
		"/v1/opportunities",
		"/v1/members",
	)

	pipe := pipeline.NewSchemaPipe(getSchemas).
		Filter(filters.KeepWithPath(filters.NewAllowPathStrategy(readObjects))).
		Map(mapping.RemoveUrlPrefix("/v1")).
		Reduce(reducers.ShortestNameFromUrlWithFullFallback).
		Map(mapping.DisplayNameFromObjectName).
		Map(mapping.DisplayNameFormat(
			mapping.CamelCaseToSpaceSeparated,
			mapping.CapitalizeFirstLetterEveryWord,
		)).
		Map(mapping.Fields(typedValueField))

	output.PrintUnusedRules("readObjects", readObjects)
	goutils.MustBeNil(output.WriteMetadata(files.OutputDir, pipe))
	goutils.MustBeNil(output.WriteQueryParamStats(files.OutputDir, pipe))
	goutils.MustBeNil(output.WriteEndpoints(files.OutputDir, &pipe, nil, nil, nil))
}

// typedValueField describes an entry of "fields" by its inner value rather than by its
// {value, valueType} wrapper, since the connector reads only the inner value.
// The provider type becomes the Lightfield valueType (e.g. EMAIL, FULL_NAME).
func typedValueField(field spec.Field) spec.Field {
	if field.Origin.Schema == nil {
		return field
	}

	wrapper, ok := field.Origin.Schema.Properties[field.Name]
	if !ok || wrapper.Value == nil {
		return field
	}

	valueType, hasValueType := wrapper.Value.Properties["valueType"]
	value, hasValue := wrapper.Value.Properties["value"]

	if !hasValueType || !hasValue || valueType.Value == nil || len(valueType.Value.Enum) != 1 {
		return field
	}

	providerType, ok := valueType.Value.Enum[0].(string)
	if !ok {
		return field
	}

	field.Type = providerType
	field.ValueType = common.ValueTypeOther

	if isStringSchema(value) {
		field.ValueType = common.ValueTypeString
	}

	return field
}

// isStringSchema reports whether the schema is a string, optionally nullable.
func isStringSchema(ref *openapi3.SchemaRef) bool {
	if ref.Value == nil {
		return false
	}

	if ref.Value.Type != nil {
		return ref.Value.Type.Is(openapi3.TypeString)
	}

	if len(ref.Value.AnyOf) == 0 {
		return false
	}

	for _, option := range ref.Value.AnyOf {
		if option.Value == nil || option.Value.Type == nil {
			return false
		}

		if !option.Value.Type.Is(openapi3.TypeString) && !option.Value.Type.Is(openapi3.TypeNull) {
			return false
		}
	}

	return true
}
