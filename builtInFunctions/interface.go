package builtInFunctions

import vmcommon "github.com/multiversx/mx-chain-vm-common-go"

type withBlockDataHandler interface {
	SetBlockDataHandler(vmcommon.BlockDataHandler) error
	CurrentRound() (uint64, error)
	IsInterfaceNil() bool
}
