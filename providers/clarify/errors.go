package clarify

import (
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common/interpreter"
)

// Errors follow JSON:API: a top-level `errors` array of {status, title, detail, source}.
// `title` is only present on validation errors and `detail` may be null on 5xx.
// https://developer.clarify.ai/docs/api-basics/errors
var errorFormats = interpreter.NewFormatSwitch( // nolint:gochecknoglobals
	interpreter.FormatTemplate{
		MustKeys: []string{"errors"},
		Template: func() interpreter.ErrorDescriptor { return &ResponseError{} },
	},
)

type ResponseError struct {
	Errors []errorObject `json:"errors"`
}

type errorObject struct {
	Status string  `json:"status"`
	Title  string  `json:"title"`
	Detail *string `json:"detail"`
}

func (r ResponseError) CombineErr(base error) error {
	messages := make([]string, 0, len(r.Errors))

	for _, obj := range r.Errors {
		if message := obj.message(); message != "" {
			messages = append(messages, message)
		}
	}

	if len(messages) == 0 {
		return base
	}

	return fmt.Errorf("%w: %s", base, strings.Join(messages, "; "))
}

func (e errorObject) message() string {
	detail := ""
	if e.Detail != nil {
		detail = *e.Detail
	}

	switch {
	case e.Title != "" && detail != "":
		return e.Title + ": " + detail
	case detail != "":
		return detail
	default:
		return e.Title
	}
}
