package orb

import (
	"context"
	"fmt"
	"slices"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/naming"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/amp-labs/connectors/providers/orb/internal/metadata"
)

// ListObjectMetadata serves the static catalog transcribed from the docs.
// Subscriptions are the exception: the docs carry no response schema for them (the
// list-subscriptions and fetch-subscription pages are not in the published reference),
// so their fields are sampled from the first listed subscription.
func (c *Connector) ListObjectMetadata(
	ctx context.Context, objectNames []string,
) (*common.ListObjectMetadataResult, error) {
	result, err := metadata.Catalog.SelectResult(objectNames)
	if err != nil {
		return nil, err
	}

	if slices.Contains(objectNames, objectNameSubscriptions) {
		delete(result.Errors, objectNameSubscriptions)

		subscriptions, err := c.sampleObjectMetadata(ctx, objectNameSubscriptions)
		if err != nil {
			result.AppendError(objectNameSubscriptions, err)
		} else {
			result.Result[objectNameSubscriptions] = *subscriptions
		}
	}

	return result, nil
}

func (c *Connector) sampleObjectMetadata(ctx context.Context, objectName string) (*common.ObjectMetadata, error) {
	sampleURL, err := urlbuilder.New(c.ProviderInfo().BaseURL, apiVersion, objectName)
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
		return nil, fmt.Errorf("%w: no %s to sample fields from", common.ErrMissingExpectedValues, objectName)
	}

	record, err := jsonquery.Convertor.ObjectToMap(records[0])
	if err != nil {
		return nil, err
	}

	objectMetadata := common.NewObjectMetadata(
		naming.CapitalizeFirstLetterEveryWord(objectName), common.FieldsMetadata{},
	)

	for field, value := range record {
		objectMetadata.AddFieldMetadata(field, common.FieldMetadata{
			DisplayName: field,
			ValueType:   common.InferValueTypeFromData(value),
		})
	}

	return objectMetadata, nil
}
