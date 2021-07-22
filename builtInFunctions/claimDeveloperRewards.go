package builtInFunctions

import (
	"bytes"
	"math/big"
	"sync"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data/vm"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

type claimDeveloperRewards struct {
	baseAlwaysActive
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
	gasRemaining := computeGasRemaining(acntSnd, vmInput.GasProvided, c.gasCost)
	if check.IfNil(acntDst) {
		// cross-shard call, in sender shard only the gas is taken out
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
		BalanceDelta:    big.NewInt(0).Set(value),
		OutputTransfers: []vmcommon.OutputTransfer{outTransfer},
	}

	vmOutput.OutputAccounts = make(map[string]*vmcommon.OutputAccount)
	vmOutput.OutputAccounts[string(outputAcc.Address)] = outputAcc

	if check.IfNil(acntSnd) {
		return vmOutput, nil
	}

	err = acntSnd.AddToBalance(value)
	if err != nil {
		return nil, err
	}

	if vmcommon.IsSmartContractAddress(vmInput.CallerAddr) {
		vmOutput.OutputAccounts = make(map[string]*vmcommon.OutputAccount)
	}

	return vmOutput, nil
}

// IsInterfaceNil returns true if underlying object is nil
func (c *claimDeveloperRewards) IsInterfaceNil() bool {
	return c == nil
}
