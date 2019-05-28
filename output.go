package vminterface

import (
	"math/big"
)

// StorageUpdate data pertaining changes in an account storage.
// Note: current implementation will also return unmodified storage entries.
type StorageUpdate struct {
	Offset []byte
	Data   []byte
}

// OutputAccount account state after contract execution.
// Note: current implementation will also return unmodified accounts.
type OutputAccount struct {
	Address        []byte
	Nonce          *big.Int
	Balance        *big.Int
	StorageUpdates []*StorageUpdate
	Code           []byte
}

// LogEntry is the contract execution log
type LogEntry struct {
	Address []byte
	Topics  []*big.Int
	Data    []byte
}

// VMOutput is the return data and final account state after a SC execution
type VMOutput struct {
	ReturnData      []*big.Int
	ReturnCode      ReturnCode
	GasRemaining    *big.Int
	GasRefund       *big.Int
	Error           bool
	OutputAccounts  []*OutputAccount
	DeletedAccounts [][]byte
	TouchedAccounts [][]byte
	Logs            []*LogEntry
}
