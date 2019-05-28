package vmcommon

import (
	"math/big"
)

// BlockchainHook is the interface for VM blockchain callbacks
type BlockchainHook interface {
	// An account with Balance = 0 and Nonce = 0 is considered to not exist
	AccountExists(address []byte) (bool, error)

	// Should retrieve the balance of an account
	GetBalance(address []byte) (*big.Int, error)

	// Should retrieve the nonce of an account
	GetNonce(address []byte) (*big.Int, error)

	// Get the storage value for a certain account and index.
	// Should return an empty byte array if the key is missing from the account storage
	GetStorageData(accountAddress []byte, index []byte) ([]byte, error)

	// Should return whether of not an account is SC.
	IsCodeEmpty(address []byte) (bool, error)

	// Should return the compiled and assembled SC code.
	// Empty byte array if the account is a wallet.
	GetCode(address []byte) ([]byte, error)

	// Should return the hash of the nth previous blockchain.
	// Offset specifies how many blocks we need to look back.
	GetBlockhash(offset *big.Int) ([]byte, error)
}
