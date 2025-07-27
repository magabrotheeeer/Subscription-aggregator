package response

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator"
)

type Response struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

const (
	StatusOK    = "OK"
	StatusError = "Error"
)

func OK() Response {
	return Response{
		Status: StatusOK,
	}
}

func Error(msg string) Response {
	return Response{
		Status: StatusError,
		Error:  msg,
	}
}

func ValidationError(errs validator.ValidationErrors) Response {
	var errsMsgs []string

	for _, err := range errs {
		switch err.ActualTag() {
		case "required":
			errsMsgs = append(errsMsgs, fmt.Sprintf("field %s is a required field", err.Field()))
		case "alphanum":
			errsMsgs = append(errsMsgs, fmt.Sprintf("field %s can contain only numbers and letters", err.Field()))
		case "numeric":
			errsMsgs = append(errsMsgs, fmt.Sprintf("field %s can contain only numbers", err.Field()))
		case "uuid":
			errsMsgs = append(errsMsgs, fmt.Sprintf("field %s can contain only uuid", err.Field()))
		case "datetime=01-2006":
			errsMsgs = append(errsMsgs, fmt.Sprintf("field %s can contain only date in format 01-2006", err.Field()))
		default:
			errsMsgs = append(errsMsgs, fmt.Sprintf("field %s is not a valid", err.Field()))
		}
	}
	return Response{
		Status: StatusError,
		Error:  strings.Join(errsMsgs, ", "),
	}
}
