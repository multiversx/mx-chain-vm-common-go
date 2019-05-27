package vminterface

import (
	"math/big"
)

// StorageUpdate ... data pertaining changes in an account storage
// note: current implementation will also return unmodified storage entries
type StorageUpdate struct {
	Offset []byte
	Data   []byte
}

// OutputAccount ... account state after contract execution
// note: current implementation will also return unmodified accounts
type OutputAccount struct {
	Address        []byte
	Nonce          *big.Int
	Balance        *big.Int
	StorageUpdates []*StorageUpdate
	Code           string
}

// LogEntry ... contract execution log
type LogEntry struct {
	Address []byte
	Topics  []*big.Int
	Data    []byte
}

// VMOutput ...
type VMOutput struct {
	ReturnData       []*big.Int
	ReturnCode       *big.Int
	GasRemaining     *big.Int
	GasRefund        *big.Int
	Error            bool
	ModifiedAccounts []*OutputAccount
	DeletedAccounts  [][]byte
	TouchedAccounts  [][]byte
	Logs             []*LogEntry
}
