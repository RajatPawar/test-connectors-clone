package lightfield

import (
	"fmt"

	"github.com/amp-labs/connectors/common/interpreter"
)

// Errors are wrapped in an "error" object.
// https://docs.lightfield.app/using-the-api/errors
var errorFormats = interpreter.NewFormatSwitch( // nolint:gochecknoglobals
	[]interpreter.FormatTemplate{
		{
			MustKeys: []string{"error"},
			Template: func() interpreter.ErrorDescriptor { return &ResponseError{} },
		},
	}...,
)

type ResponseError struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
		Code    string `json:"code"`
		Param   string `json:"param"`
	} `json:"error"`
}

func (r ResponseError) CombineErr(base error) error {
	message := r.Error.Message
	if r.Error.Code != "" {
		message = fmt.Sprintf("%s (%s)", message, r.Error.Code)
	}

	if message == "" {
		return base
	}

	return fmt.Errorf("%w: %s", base, message)
}
