package builtInFunctions

import (
	"testing"

	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/require"
)

func TestNewBlockDataHandler(t *testing.T) {
	t.Parallel()

	bdh := NewBlockDataHandler()
	require.NotNil(t, bdh)
	require.Nil(t, bdh.handler)
}

func TestBlockDataHandler_SetBlockDataHandler(t *testing.T) {
	t.Parallel()

	bdh := NewBlockDataHandler()
	err := bdh.SetBlockDataHandler(nil)
	require.Equal(t, ErrNilBlockDataHandler, err)

	newDataHandler := &mock.BlockDataHandlerStub{}
	err = bdh.SetBlockDataHandler(newDataHandler)
	require.Nil(t, err)
	require.Equal(t, newDataHandler, bdh.handler)
}

func TestBlockDataHandler_CurrentRound(t *testing.T) {
	t.Parallel()

	bdh := NewBlockDataHandler()
	_, err := bdh.CurrentRound()
	require.Equal(t, ErrNilBlockDataHandler, err)

	newDataHandler := &mock.BlockDataHandlerStub{
		CurrentRoundCalled: func() uint64 {
			return 1
		},
	}
	bdh.handler = newDataHandler

	currentRound, err := bdh.CurrentRound()
	require.Nil(t, err)
	require.Equal(t, uint64(1), currentRound)
}

func TestBlockDataHandler_IsInterfaceNil(t *testing.T) {
	t.Parallel()

	var instance *blockDataHandler
	require.True(t, instance.IsInterfaceNil())

	instance = &blockDataHandler{}
	require.False(t, instance.IsInterfaceNil())
}
