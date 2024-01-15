package builtInFunctions

import (
	"github.com/multiversx/mx-chain-core-go/core/check"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

type blockDataHandler struct {
	handler vmcommon.BlockDataHandler
}

// NewBlockDataHandler returns the block data handler built-in function component
func NewBlockDataHandler() *blockDataHandler {
	return &blockDataHandler{}
}

// SetBlockDataHandler is called when block data handler is set
func (e *blockDataHandler) SetBlockDataHandler(blockDataHandler vmcommon.BlockDataHandler) error {
	if check.IfNil(blockDataHandler) {
		return ErrNilBlockDataHandler
	}

	e.handler = blockDataHandler
	return nil
}

// CurrentRound returns the current round
func (e *blockDataHandler) CurrentRound() (uint64, error) {
	if check.IfNil(e.handler) {
		return 0, ErrNilBlockDataHandler
	}

	return e.handler.CurrentRound(), nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (e *blockDataHandler) IsInterfaceNil() bool {
	return e == nil
}
