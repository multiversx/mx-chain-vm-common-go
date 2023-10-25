package builtInFunctions

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

type changeOwnerAddress struct {
	baseAlwaysActiveHandler
	gasCost      uint64
	mutExecution sync.RWMutex

	enableEpochsHandler vmcommon.EnableEpochsHandler
}

// NewChangeOwnerAddressFunc create a new change owner built-in function
func NewChangeOwnerAddressFunc(gasCost uint64, enableEpochsHandler vmcommon.EnableEpochsHandler) (*changeOwnerAddress, error) {
	if check.IfNil(enableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}

	return &changeOwnerAddress{
		gasCost:             gasCost,
		enableEpochsHandler: enableEpochsHandler,
	}, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (c *changeOwnerAddress) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	c.mutExecution.Lock()
	c.gasCost = gasCost.BuiltInCost.ChangeOwnerAddress
	c.mutExecution.Unlock()
}

// ProcessBuiltinFunction processes simple protocol built-in function
func (c *changeOwnerAddress) ProcessBuiltinFunction(
	acntSnd, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	c.mutExecution.RLock()
	defer c.mutExecution.RUnlock()

	if vmInput == nil {
		return nil, ErrNilVmInput
	}
	if len(vmInput.Arguments) == 0 {
		return nil, ErrInvalidArguments
	}
	if vmInput.CallValue.Cmp(zero) != 0 {
		return nil, ErrBuiltInFunctionCalledWithValue
	}
	if len(vmInput.Arguments[0]) != len(vmInput.CallerAddr) {
		return nil, ErrInvalidAddressLength
	}
	if vmInput.GasProvided < c.gasCost {
		return nil, ErrNotEnoughGas
	}
	gasRemaining := computeGasRemaining(acntSnd, vmInput.GasProvided, c.gasCost)

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok, GasRemaining: gasRemaining}
	isCrossShardCallThroughASmartContract := check.IfNil(acntDst) && vmcommon.IsSmartContractAddress(vmInput.CallerAddr)
	if isCrossShardCallThroughASmartContract && c.enableEpochsHandler.IsChangeOwnerAddressCrossShardThroughSCEnabled() {
		addOutputTransferToVMOutput(
			1,
			vmInput.CallerAddr,
			core.BuiltInFunctionChangeOwnerAddress,
			vmInput.Arguments,
			vmInput.RecipientAddr,
			vmInput.GasLocked,
			vmInput.CallType,
			vmOutput)
	}

	if check.IfNil(acntDst) {
		// cross-shard call, in sender shard only the gas is taken out
		return vmOutput, nil
	}

	if !bytes.Equal(vmInput.CallerAddr, acntDst.GetOwnerAddress()) {
		return nil, fmt.Errorf("%w not the owner of the account", ErrOperationNotPermitted)
	}

	err := acntDst.ChangeOwnerAddress(vmInput.CallerAddr, vmInput.Arguments[0])
	if err != nil {
		return nil, err
	}

	logEntry := &vmcommon.LogEntry{
		Identifier: []byte(vmInput.Function),
		Address:    vmInput.RecipientAddr,
		Topics:     [][]byte{vmInput.Arguments[0]},
	}
	vmOutput.Logs = make([]*vmcommon.LogEntry, 0, 1)
	vmOutput.Logs = append(vmOutput.Logs, logEntry)

	return vmOutput, nil
}

func computeGasRemaining(snd vmcommon.UserAccountHandler, gasProvided uint64, gasToUse uint64) uint64 {
	if gasProvided < gasToUse {
		return 0
	}
	// in case of built in functions - gas is consumed in sender shard, returned already in sender shard
	// thus we must return with 0 here
	if check.IfNil(snd) {
		return 0
	}

	return gasProvided - gasToUse
}

// IsInterfaceNil returns true if underlying object in nil
func (c *changeOwnerAddress) IsInterfaceNil() bool {
	return c == nil
}
