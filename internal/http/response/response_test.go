package response

import (
	"testing"

	"github.com/go-playground/validator"
	"github.com/stretchr/testify/assert"
)

func TestStatusOKWithData(t *testing.T) {
	data := map[string]string{"key": "value"}
	resp := StatusOKWithData(data)

	assert.Equal(t, StatusOK, resp.Status)
	assert.Empty(t, resp.Error)
	assert.Equal(t, data, resp.Data)
}

func TestError(t *testing.T) {
	msg := "something went wrong"
	resp := Error(msg)

	assert.Equal(t, StatusError, resp.Status)
	assert.Equal(t, msg, resp.Error)
	assert.Nil(t, resp.Data)
}

func TestValidationError(t *testing.T) {
	type TestStruct struct {
		Name string `validate:"required,alphanum"`
		Age  string `validate:"numeric"`
	}

	v := validator.New()
	ts := TestStruct{
		Name: "!!!",
		Age:  "twenty",
	}

	err := v.Struct(ts)
	assert.Error(t, err)

	validationErrors := err.(validator.ValidationErrors)
	resp := ValidationError(validationErrors)

	assert.Equal(t, StatusError, resp.Status)
	assert.NotEmpty(t, resp.Error)

	errMsg := resp.Error
	assert.Contains(t, errMsg, "field Name can contain only numbers and letters")
	assert.Contains(t, errMsg, "field Age can contain only numbers")
}

func TestValidationErrorRequired(t *testing.T) {
	type TestStruct struct {
		Name string `validate:"required"`
	}

	v := validator.New()
	ts := TestStruct{}

	err := v.Struct(ts)
	assert.Error(t, err)

	validationErrors := err.(validator.ValidationErrors)
	resp := ValidationError(validationErrors)

	assert.Equal(t, StatusError, resp.Status)
	assert.Contains(t, resp.Error, "field Name is a required field")
}
