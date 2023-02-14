package mock

import "github.com/multiversx/mx-chain-vm-common-go"

// ESDTRoleHandlerStub -
type ESDTRoleHandlerStub struct {
	CheckAllowedToExecuteCalled func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error
}

// CheckAllowedToExecute -
func (e *ESDTRoleHandlerStub) CheckAllowedToExecute(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
	if e.CheckAllowedToExecuteCalled != nil {
		return e.CheckAllowedToExecuteCalled(account, tokenID, action)
	}

	return nil
}

// IsInterfaceNil -
func (e *ESDTRoleHandlerStub) IsInterfaceNil() bool {
	return e == nil
}
