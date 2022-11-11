package builtInFunctions

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-vm-common/mock"
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

	enableEpochsHandler := mock.EnableEpochsHandlerStub{}
	handler = &baseActiveHandler{
		activeHandler: enableEpochsHandler.IsSCDeployFlagEnabled,
	}
	assert.False(t, handler.IsActive())

	enableEpochsHandler.IsSCDeployFlagEnabledField = true
	assert.True(t, handler.IsActive())
}

func TestBaseAlwaysActiveHandler_IsActive(t *testing.T) {
	t.Parallel()

	handler := baseAlwaysActiveHandler{}
	assert.False(t, check.IfNil(handler))
	assert.True(t, handler.IsActive())
}
