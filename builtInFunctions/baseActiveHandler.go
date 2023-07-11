package builtInFunctions

type baseAlwaysActiveHandler struct {
}

// IsActive returns true as this built-in function is always active
func (b baseAlwaysActiveHandler) IsActive() bool {
	return trueHandler(epochZeroHandler())
}

// IsInterfaceNil always returns false
func (b baseAlwaysActiveHandler) IsInterfaceNil() bool {
	return false
}

type baseActiveHandler struct {
	activeHandler       func(epoch uint32) bool
	currentEpochHandler func() uint32
}

// IsActive returns true if function is active
func (b *baseActiveHandler) IsActive() bool {
	return b.activeHandler(b.currentEpochHandler())
}

// IsInterfaceNil returns true if there is no value under the interface
func (b *baseActiveHandler) IsInterfaceNil() bool {
	return b == nil
}
