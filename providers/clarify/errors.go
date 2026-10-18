package clarify

import (
	"fmt"
	"strings"

	"github.com/amp-labs/connectors/common/interpreter"
)

// Errors follow JSON:API: a top-level `errors` array. `status` is a string, `title` is present on
// validation errors only, and `detail` may be null on unexpected 5xx errors.
// https://developer.clarify.ai/docs/api-basics/errors
var errorFormats = interpreter.NewFormatSwitch( // nolint:gochecknoglobals
	interpreter.FormatTemplate{
		MustKeys: []string{"errors"},
		Template: func() interpreter.ErrorDescriptor { return &ResponseError{} },
	},
)

type ResponseError struct {
	Errors []ErrorDetails `json:"errors"`
}

type ErrorDetails struct {
	Status string `json:"status"`
	Title  string `json:"title,omitempty"`
	Detail string `json:"detail,omitempty"`
}

func (r ResponseError) CombineErr(base error) error {
	messages := make([]string, 0, len(r.Errors))

	for _, details := range r.Errors {
		if message := details.message(); message != "" {
			messages = append(messages, message)
		}
	}

	if len(messages) == 0 {
		return base
	}

	return fmt.Errorf("%w: %s", base, strings.Join(messages, ", "))
}

func (d ErrorDetails) message() string {
	switch {
	case d.Title == "":
		return d.Detail
	case d.Detail == "":
		return d.Title
	default:
		return d.Title + ": " + d.Detail
	}
}
