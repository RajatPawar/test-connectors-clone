package coda

import (
	"fmt"

	"github.com/amp-labs/connectors/common/interpreter"
)

var errorFormats = interpreter.NewFormatSwitch( // nolint:gochecknoglobals
	[]interpreter.FormatTemplate{
		{
			MustKeys: []string{"statusCode", "message"},
			Template: func() interpreter.ErrorDescriptor { return &ResponseError{} },
		},
	}...,
)

// ResponseError is the error envelope documented for every non-2xx response.
// https://coda.io/developers/apis/v1
type ResponseError struct {
	StatusCode    int    `json:"statusCode"`
	StatusMessage string `json:"statusMessage"`
	Message       string `json:"message"`
}

func (r ResponseError) CombineErr(base error) error {
	if len(r.Message) == 0 {
		return base
	}

	return fmt.Errorf("%w: %v", base, r.Message)
}
