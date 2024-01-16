package builtInFunctions

type disabledBlockchainHook struct {
}

// CurrentRound returns 0 as this is a disabled handler
func (d *disabledBlockchainHook) CurrentRound() uint64 {
	return 0
}

// IsInterfaceNil returns true if underlying object is nil
func (d *disabledBlockchainHook) IsInterfaceNil() bool {
	return d == nil
}
