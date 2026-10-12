package lightfield

import (
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common/interpreter"
)

// Errors are wrapped in an "error" object, except the documented 429 body which is flat.
// https://docs.lightfield.app/using-the-api/errors
// https://docs.lightfield.app/using-the-api/rate-limits
var errorFormats = interpreter.NewFormatSwitch( // nolint:gochecknoglobals
	[]interpreter.FormatTemplate{
		{
			MustKeys: []string{"error"},
			Template: func() interpreter.ErrorDescriptor { return &ResponseError{} },
		},
		{
			MustKeys: []string{"message"},
			Template: func() interpreter.ErrorDescriptor { return &ErrorDetails{} },
		},
	}...,
)

type ResponseError struct {
	Error ErrorDetails `json:"error"`
}

func (r ResponseError) CombineErr(base error) error {
	return r.Error.CombineErr(base)
}

type ErrorDetails struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
	Param   string `json:"param,omitempty"`
}

func (e ErrorDetails) CombineErr(base error) error {
	if e.Message == "" {
		return base
	}

	details := make([]string, 0, 3) // nolint:mnd
	for _, value := range []string{e.Type, e.Code, e.Param} {
		if value != "" {
			details = append(details, value)
		}
	}

	if len(details) == 0 {
		return fmt.Errorf("%w: %s", base, e.Message)
	}

	return fmt.Errorf("%w: %s (%s)", base, e.Message, strings.Join(details, ", "))
}
