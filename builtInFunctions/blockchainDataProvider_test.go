package builtInFunctions

import (
	"testing"

	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/require"
)

func TestNewBlockchainDataProvider(t *testing.T) {
	t.Parallel()

	bdh := NewBlockchainDataProvider()
	require.NotNil(t, bdh)
	require.Equal(t, uint64(0), bdh.CurrentRound())
}

func TestBlockchainDataProvider_SetBlockchainHook(t *testing.T) {
	t.Parallel()

	bdh := NewBlockchainDataProvider()
	err := bdh.SetBlockchainHook(nil)
	require.Equal(t, ErrNilBlockchainHook, err)

	newDataHandler := &mock.BlockDataHandlerStub{}
	err = bdh.SetBlockchainHook(newDataHandler)
	require.Nil(t, err)
	require.Equal(t, newDataHandler, bdh.blockchainHook)
}

func TestBlockchainDataProvider_CurrentRound(t *testing.T) {
	t.Parallel()

	bdh := NewBlockchainDataProvider()
	newDataHandler := &mock.BlockDataHandlerStub{
		CurrentRoundCalled: func() uint64 {
			return 1
		},
	}
	bdh.blockchainHook = newDataHandler

	currentRound := bdh.CurrentRound()
	require.Equal(t, uint64(1), currentRound)
}

func TestBlockchainDataProvider_IsInterfaceNil(t *testing.T) {
	t.Parallel()

	var instance *blockchainDataProvider
	require.True(t, instance.IsInterfaceNil())

	instance = &blockchainDataProvider{}
	require.False(t, instance.IsInterfaceNil())
}
