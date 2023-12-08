package builtInFunctions

type disabledBlockDataHandler struct {
}

// CurrentRound returns 0 as this is a disabled handler
func (d *disabledBlockDataHandler) CurrentRound() uint64 {
	return 0
}

// IsInterfaceNil returns true if underlying object is nil
func (d *disabledBlockDataHandler) IsInterfaceNil() bool {
	return d == nil
}
