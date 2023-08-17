package builtInFunctions

import "github.com/multiversx/mx-chain-core-go/core"

type baseAlwaysActiveHandler struct {
}

// IsActive returns true as this built-in function is always active
func (b baseAlwaysActiveHandler) IsActive() bool {
	return trueHandler(placeholderFlag)
}

// IsInterfaceNil always returns false
func (b baseAlwaysActiveHandler) IsInterfaceNil() bool {
	return false
}

type baseActiveHandler struct {
	activeHandler func(flag core.EnableEpochFlag) bool
	flag          core.EnableEpochFlag
}

// IsActive returns true if function is active
func (b *baseActiveHandler) IsActive() bool {
	return b.activeHandler(b.flag)
}

// IsInterfaceNil returns true if there is no value under the interface
func (b *baseActiveHandler) IsInterfaceNil() bool {
	return b == nil
}
