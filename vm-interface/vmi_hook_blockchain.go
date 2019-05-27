package vminterface

import (
	"math/big"
)

// BlockchainHook ... interface for VM blockchain callbacks
type BlockchainHook interface {

	// an account with Balance = 0 and Nonce = 0 is considered to not exist
	AccountExists(address []byte) (bool, error)

	GetBalance(address []byte) (*big.Int, error)
	GetNonce(address []byte) (*big.Int, error)

	// the storage data is a key value pair per account, index is the key
	// should return an empty result if the key is missing from
	GetStorageData(accountAddress []byte, index []byte) ([]byte, error)

	IsCodeEmpty(address []byte) (bool, error)

	// this is the compiled and assembled code
	GetCode(address []byte) ([]byte, error)

	GetBlockhash(offset *big.Int) ([]byte, error)
}
