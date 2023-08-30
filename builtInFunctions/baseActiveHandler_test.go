package builtInFunctions

import (
	"testing"

	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/stretchr/testify/assert"
)

func TestBaseActiveHandler_IsActive(t *testing.T) {
	t.Parallel()

	handler := &baseActiveHandler{
		activeHandler: trueHandler,
	}
	assert.False(t, check.IfNil(handler))
	assert.True(t, handler.IsActive())

	handler = &baseActiveHandler{
		activeHandler: falseHandler,
	}
	assert.False(t, handler.IsActive())
}

func TestBaseAlwaysActiveHandler_IsActive(t *testing.T) {
	t.Parallel()

	handler := baseAlwaysActiveHandler{}
	assert.False(t, check.IfNil(handler))
	assert.True(t, handler.IsActive())
}
