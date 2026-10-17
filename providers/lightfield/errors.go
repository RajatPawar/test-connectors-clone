package lightfield

import (
	"fmt"

	"github.com/amp-labs/connectors/common/interpreter"
)

// Errors are documented as {"error": {"type", "message", "code", "param"}},
// but the documented 429 body is the unwrapped {"type", "message"}.
// https://docs.lightfield.app/using-the-api/errors
// https://docs.lightfield.app/using-the-api/rate-limits
var errorFormats = interpreter.NewFormatSwitch( // nolint:gochecknoglobals
	[]interpreter.FormatTemplate{
		{
			MustKeys: []string{"error"},
			Template: func() interpreter.ErrorDescriptor { return &wrappedError{} },
		},
		{
			MustKeys: []string{"message"},
			Template: func() interpreter.ErrorDescriptor { return &errorDetail{} },
		},
	}...,
)

type wrappedError struct {
	Error errorDetail `json:"error"`
}

func (e wrappedError) CombineErr(base error) error {
	return e.Error.CombineErr(base)
}

type errorDetail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    string `json:"code"`
	Param   string `json:"param"`
}

func (e errorDetail) CombineErr(base error) error {
	if e.Message == "" {
		return base
	}

	details := e.Type
	if e.Code != "" {
		details += "/" + e.Code
	}

	if e.Param != "" {
		details += " (" + e.Param + ")"
	}

	return fmt.Errorf("%w: [%s] %s", base, details, e.Message)
}
