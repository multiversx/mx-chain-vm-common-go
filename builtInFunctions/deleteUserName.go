package builtInFunctions

import (
	"sync"

	"github.com/multiversx/mx-chain-core-go/core/check"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

type deleteUserName struct {
	baseActiveHandler
	gasCost         uint64
	mapDnsAddresses map[string]struct{}
	mutExecution    sync.RWMutex
}

// NewDeleteUserNameFunc returns a delete username built in function implementation
func NewDeleteUserNameFunc(
	gasCost uint64,
	mapDnsAddresses map[string]struct{},
	enableEpochsHandler vmcommon.EnableEpochsHandler,
) (*deleteUserName, error) {
	if mapDnsAddresses == nil {
		return nil, ErrNilDnsAddresses
	}
	if check.IfNil(enableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}

	d := &deleteUserName{
		gasCost:         gasCost,
		mapDnsAddresses: make(map[string]struct{}, len(mapDnsAddresses)),
	}
	for key := range mapDnsAddresses {
		d.mapDnsAddresses[key] = struct{}{}
	}
	d.activeHandler = enableEpochsHandler.IsChangeUsernameEnabled

	return d, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (d *deleteUserName) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	d.mutExecution.Lock()
	d.gasCost = gasCost.BuiltInCost.SaveUserName
	d.mutExecution.Unlock()
}

// ProcessBuiltinFunction sets the username to the account if it is allowed
func (d *deleteUserName) ProcessBuiltinFunction(
	acntSnd, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	d.mutExecution.RLock()
	defer d.mutExecution.RUnlock()

	err := inputCheckForUserNameCall(acntSnd, vmInput, d.mapDnsAddresses, d.gasCost, 0)
	if err != nil {
		return nil, err
	}

	if check.IfNil(acntDst) {
		return createCrossShardUserNameCall(vmInput, vmInput.Function, vmInput.GasProvided-d.gasCost)
	}

	acntDst.SetUserName(nil)

	gasRemaining := vmInput.GasProvided
	if !check.IfNil(acntSnd) {
		gasRemaining = vmInput.GasProvided - d.gasCost
	}
	vmOutput := &vmcommon.VMOutput{
		GasRemaining: gasRemaining,
		ReturnCode:   vmcommon.Ok,
	}

	return vmOutput, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (d *deleteUserName) IsInterfaceNil() bool {
	return d == nil
}
