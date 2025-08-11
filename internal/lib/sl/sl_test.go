package sl_test

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
)

func TestErr_ReturnsCorrectAttr(t *testing.T) {
	err := errors.New("something went wrong")
	attr := sl.Err(err)

	assert.Equal(t, "error", attr.Key)
	assert.Equal(t, slog.StringValue("something went wrong"), attr.Value)
}

func TestErr_NilError(t *testing.T) {
	assert.Panics(t, func() {
		_ = sl.Err(nil)
	})
}
