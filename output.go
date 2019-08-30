package vmcommon

import (
	"math/big"
)

// StorageUpdate represents a change in the account storage (insert, update or delete)
// Note: current implementation might also return unmodified storage entries.
type StorageUpdate struct {
	// Offset is the storage key.
	// The VM treats this as a big.Int.
	Offset []byte

	// Data is the new storage value.
	// The VM treats this as a big.Int.
	// Zero indicates missing data for the key (or even a missing key),
	// therefore a value of zero here indicates that
	// the storage map entry with the given key can be deleted.
	Data []byte
}

// OutputAccount shows the state of an account after contract execution.
// It can be an existing account or a new account created by the transaction.
// Note: the current implementation might also return unmodified accounts.
type OutputAccount struct {
	// Address is the public key of the account.
	Address []byte

	// Nonce is the new account nonce.
	Nonce uint64

	// Balance is the account balance after running a SC.
	// Only used for some tests now, please ignore. Might be removed in the future.
	Balance *big.Int

	// BalanceDelta is by how much the balance should change following the SC execution.
	// A negative value indicates that balance should decrease.
	BalanceDelta *big.Int

	// StorageUpdates is a list of key-value pairs
	// that should be updated in the account storage.
	// Please note that it is likely that not all existing
	// account storage keys show up here.
	StorageUpdates []*StorageUpdate

	// Code is the assembled code of a smart contract account.
	// This field will be populated when a new SC must be created after the transaction.
	Code []byte
}

// LogEntry represents an entry in the contract execution log.
// TODO: document all fields.
type LogEntry struct {
	Address []byte
	Topics  []*big.Int
	Data    []byte
}

// VMOutput is the return data and final account state after a SC execution.
type VMOutput struct {
	// ReturnData is the function call returned result.
	// This value does not influence the account state in any way.
	// The value should be accessible in a UI.
	// ReturnData is part of the transaction receipt.
	ReturnData []*big.Int

	// ReturnCode is the function call error code.
	// If it is not `Ok`, the transaction failed in some way - gas is, however, consumed anyway.
	// This value does not influence the account state in any way.
	// The value should be accessible to a UI.
	// ReturnCode is part of the transaction receipt.
	ReturnCode ReturnCode

	// GasRemaining = VMInput.GasProvided - gas used.
	// It is necessary to compute how much to charge the sender for the transaction.
	GasRemaining *big.Int

	// GasRefund is how much gas the sender earned during the transaction.
	// Certain operations, like freeing up storage, actually return gas instead of consuming it.
	// Based on GasRefund, the sender could in principle be rewarded instead of taxed.
	// TODO: decide if we are going to support this.
	GasRefund *big.Int

	// OutputAccounts contains data about all acounts changed as a result of the transaction.
	// This data tells the network how to update the account data.
	// It can contain new accounts or existing changed accounts.
	// Note: the current implementation might also retrieve accounts that were not changed.
	OutputAccounts []*OutputAccount

	// DeletedAccounts is a list of public keys of accounts that need to be deleted
	// as a result of the transaction.
	DeletedAccounts [][]byte

	// TouchedAccounts is a list of public keys of accounts that were somehow involved in the VM execution.
	// TODO: investigate what we need to to about these.
	TouchedAccounts [][]byte

	// Logs is a list of event data logged by the VM.
	// Smart contracts can choose to log certain events programatically.
	// There are 3 main use cases for events and logs:
	// 1. smart contract return values for the user interface;
	// 2. asynchronous triggers with data;
	// 3. a cheaper form of storage (e.g. storing historical data that can be rendered by the frontend).
	// The logs should be accessible to the UI.
	// The logs are part of the transaction receipt.
	Logs []*LogEntry
}
