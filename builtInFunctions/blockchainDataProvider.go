package builtInFunctions

import (
	"github.com/multiversx/mx-chain-core-go/core/check"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

type blockchainDataProvider struct {
	blockchainHook vmcommon.BlockchainDataHook
}

// NewBlockchainDataProvider returns a new blockchain data provider
func NewBlockchainDataProvider() *blockchainDataProvider {
	return &blockchainDataProvider{
		blockchainHook: &disabledBlockchainHook{},
	}
}

// SetBlockchainHook sets the given blockchain hook as the data provider
func (b *blockchainDataProvider) SetBlockchainHook(blockchainHook vmcommon.BlockchainDataHook) error {
	if check.IfNil(blockchainHook) {
		return ErrNilBlockchainHook
	}

	b.blockchainHook = blockchainHook
	return nil
}

// CurrentRound returns the current round
func (b *blockchainDataProvider) CurrentRound() uint64 {
	return b.blockchainHook.CurrentRound()
}

// IsInterfaceNil returns true if there is no value under the interface
func (b *blockchainDataProvider) IsInterfaceNil() bool {
	return b == nil
}
