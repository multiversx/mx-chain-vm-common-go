package builtInFunctions

import (
	"bytes"
	"math/big"
	"sync"

	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data/vm"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

type claimDeveloperRewards struct {
	baseAlwaysActiveHandler
	gasCost      uint64
	mutExecution sync.RWMutex
}

// NewClaimDeveloperRewardsFunc returns a new developer rewards implementation
func NewClaimDeveloperRewardsFunc(gasCost uint64) *claimDeveloperRewards {
	return &claimDeveloperRewards{gasCost: gasCost}
}

// SetNewGasConfig is called whenever gas cost is changed
func (c *claimDeveloperRewards) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	c.mutExecution.Lock()
	c.gasCost = gasCost.BuiltInCost.ClaimDeveloperRewards
	c.mutExecution.Unlock()
}

// ProcessBuiltinFunction processes the protocol built-in smart contract function
func (c *claimDeveloperRewards) ProcessBuiltinFunction(
	acntSnd, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	c.mutExecution.RLock()
	defer c.mutExecution.RUnlock()

	if vmInput == nil {
		return nil, ErrNilVmInput
	}
	if vmInput.CallValue.Cmp(zero) != 0 {
		return nil, ErrBuiltInFunctionCalledWithValue
	}
	gasRemaining := computeGasRemaining(acntSnd, vmInput.GasProvided, c.gasCost, false)
	if check.IfNil(acntDst) {
		// The call is cross-shard, and we are at the sender shard.
		// Here, in the sender shard, only the gas is taken out.
		// Log entry does not need to be added.
		return &vmcommon.VMOutput{ReturnCode: vmcommon.Ok, GasRemaining: gasRemaining}, nil
	}

	if !bytes.Equal(vmInput.CallerAddr, acntDst.GetOwnerAddress()) {
		return nil, ErrOperationNotPermitted
	}
	if vmInput.GasProvided < c.gasCost {
		return nil, ErrNotEnoughGas
	}

	value, err := acntDst.ClaimDeveloperRewards(vmInput.CallerAddr)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{GasRemaining: gasRemaining, ReturnCode: vmcommon.Ok}
	outTransfer := vmcommon.OutputTransfer{
		Index:         1,
		Value:         big.NewInt(0).Set(value),
		GasLimit:      0,
		Data:          nil,
		CallType:      vm.DirectCall,
		SenderAddress: vmInput.CallerAddr,
	}
	if vmInput.CallType == vm.AsynchronousCall {
		outTransfer.GasLocked = vmInput.GasLocked
		outTransfer.GasLimit = vmOutput.GasRemaining
		outTransfer.CallType = vm.AsynchronousCallBack
		vmOutput.GasRemaining = 0
	}
	outputAcc := &vmcommon.OutputAccount{
		Address:         vmInput.CallerAddr,
		BalanceDelta:    big.NewInt(0),
		OutputTransfers: []vmcommon.OutputTransfer{outTransfer},
	}

	vmOutput.OutputAccounts = make(map[string]*vmcommon.OutputAccount)
	vmOutput.OutputAccounts[string(outputAcc.Address)] = outputAcc

	if check.IfNil(acntSnd) {
		// The call is cross-shard, and we are at the destination shard.
		addLogEntryForClaimDeveloperRewards(vmInput, vmOutput, value, vmInput.CallerAddr)
		return vmOutput, nil
	}

	err = acntSnd.AddToBalance(value)
	if err != nil {
		return nil, err
	}

	if vmcommon.IsSmartContractAddress(vmInput.CallerAddr) {
		vmOutput.OutputAccounts = make(map[string]*vmcommon.OutputAccount)
	}

	addLogEntryForClaimDeveloperRewards(vmInput, vmOutput, value, vmInput.CallerAddr)
	return vmOutput, nil
}

func addLogEntryForClaimDeveloperRewards(
	vmInput *vmcommon.ContractCallInput,
	vmOutput *vmcommon.VMOutput,
	value *big.Int,
	developerAddress []byte,
) {
	valueAsBytes := value.Bytes()

	logEntry := &vmcommon.LogEntry{
		Identifier: []byte(vmInput.Function),
		Address:    vmInput.RecipientAddr,
		Topics:     [][]byte{valueAsBytes, developerAddress},
		Data:       nil,
	}
	vmOutput.Logs = make([]*vmcommon.LogEntry, 1)
	vmOutput.Logs[0] = logEntry
}

// IsInterfaceNil returns true if underlying object is nil
func (c *claimDeveloperRewards) IsInterfaceNil() bool {
	return c == nil
}
