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
	accounts         vmcommon.AccountsAdapter
	shardCoordinator vmcommon.Coordinator
	builtInCost      vmcommon.BuiltInCost
	mutExecution     sync.RWMutex
}

// NewMigrateDataTrieFunc creates a new migrateDataTrie built-in function component
func NewMigrateDataTrieFunc(
	builtInCost vmcommon.BuiltInCost,
	enableEpochsHandler vmcommon.EnableEpochsHandler,
	accounts vmcommon.AccountsAdapter,
	shardCoordinator vmcommon.Coordinator,
) (*migrateDataTrie, error) {
	if check.IfNil(enableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}
	if check.IfNil(accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if check.IfNil(shardCoordinator) {
		return nil, ErrNilShardCoordinator
	}

	mdt := &migrateDataTrie{
		builtInCost:      builtInCost,
		accounts:         accounts,
		shardCoordinator: shardCoordinator,
	}

	mdt.baseActiveHandler.activeHandler = enableEpochsHandler.IsMigrateDataTrieEnabled

	return mdt, nil
}

// ProcessBuiltinFunction will migrate as many leaves as possible from the old version to the new version.
// This will stop when it runs out of gas.
func (mdt *migrateDataTrie) ProcessBuiltinFunction(
	_, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	dataTrieGasCost := mdt.getGasCostForDataTrieLoadAndStore()

	err := checkArgumentsForMigrateDataTrie(vmInput, dataTrieGasCost)
	if err != nil {
		return nil, err
	}

	address := vmInput.Arguments[0]
	account, err := mdt.getAccountForMigration(vmInput.CallerAddr, address)
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
	err = account.AccountDataHandler().MigrateDataTrieLeaves(argsMigrateDataTrie)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{
		GasRemaining: dtm.GetGasRemaining(),
		ReturnCode:   vmcommon.Ok,
	}

	return vmOutput, nil
}

func (mdt *migrateDataTrie) getAccountForMigration(callerAddr []byte, address []byte) (vmcommon.UserAccountHandler, error) {
	if vmcommon.IsSystemAccountAddress(address) {
		systemSCAccount, err := mdt.loadAccount(vmcommon.SystemAccountAddress)
		if err != nil {
			return nil, err
		}

		return systemSCAccount, nil
	}

	if !mdt.shardCoordinator.SameShard(callerAddr, address) {
		return nil, fmt.Errorf("%w, address must be in the same shard as the sender; caller shard = %v, address shard = %v ", ErrOperationNotPermitted, mdt.shardCoordinator.ComputeId(callerAddr), mdt.shardCoordinator.ComputeId(address))
	}

	account, err := mdt.loadAccount(address)
	if err != nil {
		return nil, err
	}

	return account, nil
}

func (mdt *migrateDataTrie) loadAccount(address []byte) (vmcommon.UserAccountHandler, error) {
	account, err := mdt.accounts.LoadAccount(address)
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
	if len(input.Arguments) != 1 {
		return fmt.Errorf("one argument must be given to migrate data trie: %w", ErrInvalidNumberOfArguments)
	}
	senderIsNotReceiver := !bytes.Equal(input.CallerAddr, input.RecipientAddr)
	if senderIsNotReceiver {
		return ErrOperationNotPermitted
	}

	return nil
}
