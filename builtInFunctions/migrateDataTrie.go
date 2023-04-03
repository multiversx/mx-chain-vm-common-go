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
	trieLoadPerNode, trieStorePerNode := mdt.getGasCostForDataTrieLoadAndStore()

	err := checkArgumentsForMigrateDataTrie(acntDst, vmInput, trieLoadPerNode, trieStorePerNode)
	if err != nil {
		return nil, err
	}

	dtm := dataTrieMigrator.NewDataTrieMigrator(vmInput.GasProvided, trieLoadPerNode, trieStorePerNode)

	oldVersion := core.NotSpecified
	newVersion := core.AutoBalanceEnabled

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

func (mdt *migrateDataTrie) getGasCostForDataTrieLoadAndStore() (uint64, uint64) {
	mdt.mutExecution.RLock()
	builtInCost := mdt.builtInCost
	mdt.mutExecution.RUnlock()

	return builtInCost.TrieLoadPerNode, builtInCost.TrieStorePerNode
}

// IsInterfaceNil returns true if there is no value under the interface
func (mdt *migrateDataTrie) IsInterfaceNil() bool {
	return mdt == nil
}

func checkArgumentsForMigrateDataTrie(
	acntDst vmcommon.UserAccountHandler,
	input *vmcommon.ContractCallInput,
	trieLoadPerNode uint64,
	trieStorePerNode uint64,
) error {
	if input == nil {
		return ErrNilVmInput
	}
	if input.GasProvided < trieLoadPerNode+trieStorePerNode {
		return fmt.Errorf("not enough gas, gas provided: %d, trie load cost: %d, trie store cost: %d", input.GasProvided, trieLoadPerNode, trieStorePerNode)
	}
	if input.CallValue.Cmp(zero) != 0 {
		return ErrBuiltInFunctionCalledWithValue
	}
	if check.IfNil(acntDst) {
		return ErrNilSCDestAccount
	}

	return nil
}
