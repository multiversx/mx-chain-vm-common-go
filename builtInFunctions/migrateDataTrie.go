package builtInFunctions

import (
	"bytes"
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

	mdt.baseActiveHandler.activeHandler = func() bool {
		return enableEpochsHandler.IsFlagEnabled(MigrateDataTrieFlag)
	}

	return mdt, nil
}

// ProcessBuiltinFunction will migrate as many leaves as possible from the old version to the new version.
// This will stop when it runs out of gas.
func (mdt *migrateDataTrie) ProcessBuiltinFunction(
	_, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	dataTrieGasCost := mdt.getGasCostForDataTrieLoadAndStore()

	err := checkArgumentsForMigrateDataTrie(acntDst, vmInput, dataTrieGasCost)
	if err != nil {
		return nil, err
	}

	argsDataTrieMigrator := dataTrieMigrator.ArgsNewDataTrieMigrator{
		GasProvided:     vmInput.GasProvided,
		DataTrieGasCost: dataTrieGasCost,
	}
	dtm := dataTrieMigrator.NewDataTrieMigrator(argsDataTrieMigrator)

	argsMigrateDataTrie := vmcommon.ArgsMigrateDataTrieLeaves{
		OldVersion:   core.NotSpecified,
		NewVersion:   core.AutoBalanceEnabled,
		TrieMigrator: dtm,
	}

	shouldMigrateAcntDst := bytes.Equal(acntDst.AddressBytes(), vmcommon.SystemAccountAddress) || !vmcommon.IsSystemAccountAddress(acntDst.AddressBytes())
	if shouldMigrateAcntDst {
		err = acntDst.AccountDataHandler().MigrateDataTrieLeaves(argsMigrateDataTrie)
		if err != nil {
			return nil, err
		}
	} else {
		err = mdt.migrateSystemAccount(argsMigrateDataTrie)
		if err != nil {
			return nil, err
		}
	}

	vmOutput := &vmcommon.VMOutput{
		GasRemaining: dtm.GetGasRemaining(),
		ReturnCode:   vmcommon.Ok,
	}

	return vmOutput, nil
}

func (mdt *migrateDataTrie) migrateSystemAccount(argsMigrateDataTrie vmcommon.ArgsMigrateDataTrieLeaves) error {
	account, err := mdt.getExistingAccount(vmcommon.SystemAccountAddress)
	if err != nil {
		return err
	}

	err = account.AccountDataHandler().MigrateDataTrieLeaves(argsMigrateDataTrie)
	if err != nil {
		return err
	}

	return mdt.accounts.SaveAccount(account)
}

func (mdt *migrateDataTrie) getExistingAccount(address []byte) (vmcommon.UserAccountHandler, error) {
	account, err := mdt.accounts.GetExistingAccount(address)
	if err != nil {
		return nil, err
	}

	userAccount, ok := account.(vmcommon.UserAccountHandler)
	if !ok {
		return nil, ErrWrongTypeAssertion
	}

	return userAccount, nil
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

func (mdt *migrateDataTrie) getGasCostForDataTrieLoadAndStore() dataTrieMigrator.DataTrieGasCost {
	mdt.mutExecution.RLock()
	builtInCost := mdt.builtInCost
	mdt.mutExecution.RUnlock()

	dataTrieGasCost := dataTrieMigrator.DataTrieGasCost{
		TrieLoadPerNode:  builtInCost.TrieLoadPerNode,
		TrieStorePerNode: builtInCost.TrieStorePerNode,
	}
	return dataTrieGasCost
}

// IsInterfaceNil returns true if there is no value under the interface
func (mdt *migrateDataTrie) IsInterfaceNil() bool {
	return mdt == nil
}

func checkArgumentsForMigrateDataTrie(
	acntDst vmcommon.UserAccountHandler,
	input *vmcommon.ContractCallInput,
	cost dataTrieMigrator.DataTrieGasCost,
) error {
	if input == nil {
		return ErrNilVmInput
	}
	if input.GasProvided < cost.TrieLoadPerNode+cost.TrieStorePerNode {
		return fmt.Errorf("not enough gas, gas provided: %d, trie load cost: %d, trie store cost: %d", input.GasProvided, cost.TrieLoadPerNode, cost.TrieStorePerNode)
	}
	if input.CallValue.Cmp(zero) != 0 {
		return ErrBuiltInFunctionCalledWithValue
	}
	if len(input.Arguments) != 0 {
		return fmt.Errorf("no arguments must be given to migrate data trie: %w", ErrInvalidNumberOfArguments)
	}
	if check.IfNil(acntDst) {
		return fmt.Errorf("destination account must be in the same shard as the sender: %w", ErrNilSCDestAccount)
	}

	return nil
}
