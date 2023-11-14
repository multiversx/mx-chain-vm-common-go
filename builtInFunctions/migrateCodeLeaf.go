package builtInFunctions

import (
	"fmt"
	"sync"

	"github.com/multiversx/mx-chain-core-go/core/check"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

const noOfArgsMigrateCodeLeaf = 0

type migrateCodeLeaf struct {
	baseActiveHandler
	accounts     vmcommon.AccountsAdapter
	gasCost      uint64
	mutExecution sync.RWMutex
}

// NewMigrateCodeLeafFunc creates a new removeCodeLeaf built-in function component
func NewMigrateCodeLeafFunc(
	gasCost uint64,
	enableEpochsHandler vmcommon.EnableEpochsHandler,
	accounts vmcommon.AccountsAdapter,
) (*migrateCodeLeaf, error) {
	if check.IfNil(enableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}
	if check.IfNil(accounts) {
		return nil, ErrNilAccountsAdapter
	}

	mdt := &migrateCodeLeaf{
		gasCost:  gasCost,
		accounts: accounts,
	}

	mdt.baseActiveHandler.activeHandler = func() bool {
		return enableEpochsHandler.IsFlagEnabled(RemoveCodeLeafFlag)
	}
	return mdt, nil
}

// ProcessBuiltinFunction will remove trie code leaf corresponding to specified codeHash
func (rcl *migrateCodeLeaf) ProcessBuiltinFunction(
	_, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	if vmInput == nil {
		return nil, ErrNilVmInput
	}
	if len(vmInput.Arguments) != noOfArgsMigrateCodeLeaf {
		return nil, fmt.Errorf("%w, expected %d, got %d ", ErrInvalidNumberOfArguments, noOfArgsMigrateCodeLeaf, len(vmInput.Arguments))
	}
	if vmInput.CallValue.Cmp(zero) != 0 {
		return nil, ErrBuiltInFunctionCalledWithValue
	}
	if check.IfNil(acntDst) {
		return nil, ErrNilSCDestAccount
	}

	err := rcl.accounts.MigrateCodeLeaf(acntDst.AddressBytes())
	if err != nil {
		return nil, err
	}

	rcl.mutExecution.RLock()
	gasRemaining := vmInput.GasProvided - rcl.gasCost
	rcl.mutExecution.RUnlock()

	return &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: gasRemaining,
	}, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (rcl *migrateCodeLeaf) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	rcl.mutExecution.Lock()
	rcl.gasCost = gasCost.BuiltInCost.MigrateCodeLeaf
	rcl.mutExecution.Unlock()
}

// IsInterfaceNil returns true if there is no value under the interface
func (rcl *migrateCodeLeaf) IsInterfaceNil() bool {
	return rcl == nil
}
