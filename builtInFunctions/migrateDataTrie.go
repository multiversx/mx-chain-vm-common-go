package builtInFunctions

import (
	"fmt"
	"sync"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/dataTrieMigrator"
)

type migrateDataTrie struct {
	baseActiveHandler
	accounts     vmcommon.AccountsAdapter
	builtInCost  vmcommon.BuiltInCost
	mutExecution sync.RWMutex
}

// NewMigrateDataTrieFunc creates a new migrateDataTrie built-in function component
func NewMigrateDataTrieFunc(
	builtInCost vmcommon.BuiltInCost,
	enableEpochsHandler vmcommon.EnableEpochsHandler,
	accounts vmcommon.AccountsAdapter,
) (*migrateDataTrie, error) {
	if check.IfNil(enableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}
	if check.IfNil(accounts) {
		return nil, ErrNilAccountsAdapter
	}

	mdt := &migrateDataTrie{
		builtInCost: builtInCost,
		accounts:    accounts,
	}

	mdt.baseActiveHandler.activeHandler = enableEpochsHandler.IsAutoBalanceDataTriesEnabled

	return mdt, nil
}

// ProcessBuiltinFunction will migrate as many leaves as possible from the old version to the new version.
// This will stop when it runs out of gas.
func (mdt *migrateDataTrie) ProcessBuiltinFunction(_, acntDst vmcommon.UserAccountHandler, vmInput *vmcommon.ContractCallInput) (*vmcommon.VMOutput, error) {
	err := checkArgumentsForMigrateDataTrie(acntDst, vmInput)
	if err != nil {
		return nil, err
	}

	mdt.mutExecution.RLock()
	dtm, err := dataTrieMigrator.NewDataTrieMigrator(vmInput.GasProvided, mdt.builtInCost)
	if err != nil {
		mdt.mutExecution.RUnlock()
		return nil, err
	}
	mdt.mutExecution.RUnlock()

	firstArgument := vmInput.Arguments[0]
	secondArgument := vmInput.Arguments[1]

	oldVersion := core.TrieNodeVersion(firstArgument[0])
	newVersion := core.TrieNodeVersion(secondArgument[0])

	err = acntDst.AccountDataHandler().MigrateDataTrieLeaves(oldVersion, newVersion, dtm)
	if err != nil {
		return nil, err
	}

	err = mdt.accounts.SaveAccount(acntDst)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{
		GasRemaining: dtm.GetGasRemaining(),
		ReturnCode:   vmcommon.Ok,
	}

	return vmOutput, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (mdt *migrateDataTrie) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	mdt.mutExecution.Lock()
	mdt.builtInCost = gasCost.BuiltInCost
	mdt.mutExecution.Unlock()
}

// IsInterfaceNil returns true if there is no value under the interface
func (mdt *migrateDataTrie) IsInterfaceNil() bool {
	return mdt == nil
}

func checkArgumentsForMigrateDataTrie(acntDst vmcommon.UserAccountHandler, input *vmcommon.ContractCallInput) error {
	if input == nil {
		return ErrNilVmInput
	}
	if len(input.Arguments) != 2 {
		return ErrInvalidArguments
	}
	// oldVersion and newVersion must be contained in an uint8 type, so they must be 1 byte long
	if len(input.Arguments[0]) != 1 {
		return fmt.Errorf("%w old version must be 1 byte long", ErrInvalidArguments)
	}
	if len(input.Arguments[1]) != 1 {
		return fmt.Errorf("%w new version must be 1 byte long", ErrInvalidArguments)
	}
	if input.CallValue.Cmp(zero) != 0 {
		return ErrBuiltInFunctionCalledWithValue
	}
	if check.IfNil(acntDst) {
		return ErrNilSCDestAccount
	}

	return nil
}
