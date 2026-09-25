package main

import (
	"github.com/getkin/kin-openapi/openapi3"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/staticschema"
)

const maxSchemaDepth = 8

// property is one field as the spec declares it, after allOf/oneOf/anyOf are merged.
type property struct {
	Type     string
	Format   string
	Enum     []string
	ItemEnum []string
	ReadOnly bool
}

// properties merges a schema's properties across allOf/oneOf/anyOf (the way api3 does for
// reads), keeping the flags api3 drops: format, readOnly, and array item enums.
func properties(ref *openapi3.SchemaRef) map[string]property {
	out := map[string]property{}
	collect(ref, out, 0)

	return out
}

func collect(ref *openapi3.SchemaRef, out map[string]property, depth int) {
	if ref == nil || ref.Value == nil || depth > maxSchemaDepth {
		return
	}

	schema := ref.Value
	for _, group := range [][]*openapi3.SchemaRef{schema.AllOf, schema.OneOf, schema.AnyOf} {
		for _, part := range group {
			collect(part, out, depth+1)
		}
	}

	for name, prop := range schema.Properties {
		if prop == nil || prop.Value == nil {
			continue
		}

		p := describe(prop)
		if prev, ok := out[name]; ok && prev.Type != "" && p.Type == "" {
			p.Type, p.Format = prev.Type, prev.Format
		}

		out[name] = p
	}
}

func describe(ref *openapi3.SchemaRef) property {
	v := ref.Value
	p := property{Format: v.Format, ReadOnly: v.ReadOnly, Enum: enumStrings(v.Enum)}

	if v.Type != nil && len(*v.Type) > 0 {
		for _, t := range *v.Type {
			if t != "null" {
				p.Type = t

				break
			}
		}
	}

	// A nullable wrapper (anyOf: [X, null]) carries its type on the non-null branch.
	if p.Type == "" {
		for _, part := range append(append([]*openapi3.SchemaRef{}, v.AnyOf...), v.OneOf...) {
			if part != nil && part.Value != nil && part.Value.Type != nil && !part.Value.Type.Is("null") {
				inner := describe(part)
				inner.ReadOnly = inner.ReadOnly || p.ReadOnly

				return inner
			}
		}
	}

	if p.Type == "array" && v.Items != nil && v.Items.Value != nil {
		p.ItemEnum = enumStrings(v.Items.Value.Enum)
	}

	return p
}

func enumStrings(values []any) []string {
	out := make([]string, 0, len(values))

	for _, value := range values {
		if s, ok := value.(string); ok {
			out = append(out, s)
		}
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

// valueType maps a spec type to Ampersand's ValueType. Unlike the older shared converter it keeps
// number (float), date and date-time instead of flattening them to other/string.
func valueType(p property) common.ValueType {
	switch p.Type {
	case "integer":
		return common.ValueTypeInt
	case "number":
		return common.ValueTypeFloat
	case "boolean":
		return common.ValueTypeBoolean
	case "string":
		switch {
		case len(p.Enum) > 0:
			return common.ValueTypeSingleSelect
		case p.Format == "date":
			return common.ValueTypeDate
		case p.Format == "date-time":
			return common.ValueTypeDateTime
		default:
			return common.ValueTypeString
		}
	case "array":
		if len(p.ItemEnum) > 0 {
			return common.ValueTypeMultiSelect
		}

		return common.ValueTypeOther
	default:
		return common.ValueTypeOther
	}
}

func fieldMetadata(name string, p property, readOnly *bool) staticschema.FieldMetadata {
	values := p.Enum
	if len(values) == 0 {
		values = p.ItemEnum
	}

	var options staticschema.FieldValues
	for _, value := range values {
		options = append(options, staticschema.FieldValue{Value: value, DisplayValue: value})
	}

	providerType := p.Type
	if providerType == "" {
		providerType = "object"
	}

	return staticschema.FieldMetadata{
		DisplayName:  name,
		ValueType:    valueType(p),
		ProviderType: providerType,
		ReadOnly:     readOnly,
		Values:       options,
	}
}
