package coda

import (
	"fmt"

	"github.com/amp-labs/connectors/common/interpreter"
)

var errorFormats = interpreter.NewFormatSwitch( // nolint:gochecknoglobals
	[]interpreter.FormatTemplate{
		{
			MustKeys: []string{"statusCode"},
			Template: func() interpreter.ErrorDescriptor { return &ResponseError{} },
		},
	}...,
)

// ResponseError is the error envelope shared by every Coda error response:
// {"statusCode": 400, "statusMessage": "Bad Request", "message": "..."}.
type ResponseError struct {
	StatusCode    int    `json:"statusCode"`
	StatusMessage string `json:"statusMessage"`
	Message       string `json:"message"`
}

func (r ResponseError) CombineErr(base error) error {
	if r.Message == "" {
		return base
	}

	return fmt.Errorf("%w: %s", base, r.Message)
}
