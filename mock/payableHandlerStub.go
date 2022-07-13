package mock

import vmcommon "github.com/ElrondNetwork/elrond-vm-common"

// PayableHandlerStub -
type PayableHandlerStub struct {
	IsPayableCalled              func(address []byte) (bool, error)
	CheckPayableCalled           func(vmInput *vmcommon.ContractCallInput, dstAddress []byte, minArgs int) error
	DetermineIsSCCallAfterCalled func(vmInput *vmcommon.ContractCallInput, dstAddress []byte, mintArgs int) bool
}

// IsPayable -
func (p *PayableHandlerStub) IsPayable(_, address []byte) (bool, error) {
	if p.IsPayableCalled != nil {
		return p.IsPayableCalled(address)
	}
	return true, nil
}

// CheckPayable -
func (p *PayableHandlerStub) CheckPayable(vmInput *vmcommon.ContractCallInput, dstAddress []byte, minArgs int) error {
	if p.CheckPayableCalled != nil {
		return p.CheckPayableCalled(vmInput, dstAddress, minArgs)
	}
	return nil
}

// DetermineIsSCCallAfter -
func (p *PayableHandlerStub) DetermineIsSCCallAfter(vmInput *vmcommon.ContractCallInput, dstAddress []byte, minArgs int) bool {
	if p.DetermineIsSCCallAfterCalled != nil {
		return p.DetermineIsSCCallAfterCalled(vmInput, dstAddress, minArgs)
	}
	return false
}

// IsInterfaceNil -
func (p *PayableHandlerStub) IsInterfaceNil() bool {
	return p == nil
}
