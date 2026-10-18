package orb

import (
	"context"
	"slices"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/naming"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/amp-labs/connectors/internal/objectcatalog"
	"github.com/amp-labs/connectors/providers/orb/internal/metadata"
)

// ListObjectMetadata serves the static catalog. Orb has no describe endpoint, and its records
// have no user-defined fields beyond the free-form `metadata` map, which is listed as one field.
//
// The API reference in our docs has no response schema for subscriptions, so their catalog entry
// holds only the fields confirmed by a live response, and the rest are sampled from one live
// subscription: fields the catalog lacks are added with a type inferred from the sampled value.
// An account with no subscriptions gets the catalog fields alone.
func (c *Connector) ListObjectMetadata(
	ctx context.Context, objectNames []string,
) (*common.ListObjectMetadataResult, error) {
	selection, err := metadata.Catalog.Select(objectNames)
	if err != nil {
		return nil, err
	}

	if !slices.Contains(objectNames, objectNameSubscriptions) {
		return selection.Result(), nil
	}

	sampleErr := c.addSampledFields(ctx, selection, objectNameSubscriptions)

	result := selection.Result()

	if sampleErr != nil {
		delete(result.Result, objectNameSubscriptions)
		result.AppendError(objectNameSubscriptions, sampleErr)
	}

	return result, nil
}

func (c *Connector) addSampledFields(
	ctx context.Context, selection objectcatalog.Selection, objectName string,
) error {
	record, err := c.sampleRecord(ctx, objectName)
	if err != nil || record == nil {
		return err
	}

	known, err := metadata.Catalog.Object(objectName)
	if err != nil {
		return err
	}

	for field, value := range record {
		if _, ok := known.Fields[field]; ok {
			continue
		}

		selection.AddField(objectName, field, common.FieldMetadata{
			DisplayName: naming.CapitalizeFirstLetterEveryWord(strings.ReplaceAll(field, "_", " ")),
			ValueType:   common.InferValueTypeFromData(value),
		})
	}

	return nil
}

// sampleRecord returns the first record of the object's list endpoint, or nil when it is empty.
func (c *Connector) sampleRecord(ctx context.Context, objectName string) (map[string]any, error) {
	path, err := metadata.Catalog.ReadPath(objectName)
	if err != nil {
		return nil, err
	}

	sampleURL, err := urlbuilder.New(c.ProviderInfo().BaseURL, path)
	if err != nil {
		return nil, err
	}

	sampleURL.WithQueryParam("limit", "1")

	response, err := c.JSONHTTPClient().Get(ctx, sampleURL.String())
	if err != nil {
		return nil, err
	}

	body, ok := response.Body()
	if !ok {
		return nil, common.ErrEmptyJSONHTTPResponse
	}

	records, err := jsonquery.New(body).ArrayRequired("data")
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, nil // nolint:nilnil
	}

	return jsonquery.Convertor.ObjectToMap(records[0])
}
