package builtInFunctions

import vmcommon "github.com/multiversx/mx-chain-vm-common-go"

// disabledPayableHandler is a disabled payableCheck handler that implements PayableChecker interface but it is disabled
type disabledPayableHandler struct {
}

// CheckPayable returns error as this is a disabled payableCheck handler
func (d *disabledPayableHandler) CheckPayable(_ *vmcommon.ContractCallInput, _ []byte, _ int) error {
	return ErrAccountNotPayable
}

// DetermineIsSCCallAfter returns false as this is a disabled handler
func (d *disabledPayableHandler) DetermineIsSCCallAfter(_ *vmcommon.ContractCallInput, _ []byte, _ int) bool {
	return false
}

// IsInterfaceNil returns true if underlying object is nil
func (d *disabledPayableHandler) IsInterfaceNil() bool {
	return d == nil
}
