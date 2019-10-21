package vmcommon

import (
	"math/big"
)

// BlockchainHook is the interface for VM blockchain callbacks
type BlockchainHook interface {
	// An account with Balance = 0 and Nonce = 0 is considered to not exist
	AccountExists(address []byte) (bool, error)

	// NewAddress yields the address of a new SC account, when one such account is created.
	// The result should only depend on the creator address and nonce.
	// Returning an empty address lets the VM decide what the new address should be.
	NewAddress(creatorAddress []byte, creatorNonce uint64, vmType []byte) ([]byte, error)

	// Should yield the balance of an account.
	// Should yield zero if account does not exist.
	GetBalance(address []byte) (*big.Int, error)

	// Should yield the nonce of an account.
	// Should yield zero if account does not exist.
	GetNonce(address []byte) (uint64, error)

	// Should yield the storage value for a certain account and index.
	// Should return an empty byte array if the key is missing from the account storage,
	// or if account does not exist.
	GetStorageData(accountAddress []byte, index []byte) ([]byte, error)

	// Should return whether of not an account is SC.
	IsCodeEmpty(address []byte) (bool, error)

	// Should return the compiled and assembled SC code.
	// Should yield an empty byte array if the account is a wallet.
	GetCode(address []byte) ([]byte, error)

	// Should return the hash of the nth previous blockchain.
	// Offset specifies how many blocks we need to look back.
	GetBlockhash(offset *big.Int) ([]byte, error)

	// LastNonce returns the nonce from from the last committed block
	LastNonce() uint64

	// LastRound returns the round from the last committed block
	LastRound() uint64

	// LastTimeStamp returns the timeStamp from the last committed block
	LastTimeStamp() uint64

	// LastRandomSeed returns the random seed from the last committed block
	LastRandomSeed() []byte

	// LastEpoch returns the epoch from the last committed block
	LastEpoch() uint32

	// GetStateRootHash returns the state root hash from the last committed block
	GetStateRootHash() []byte

	// CurrentNonce returns the nonce from the current block
	CurrentNonce() uint64

	// CurrentRound returns the round from the current block
	CurrentRound() uint64

	// CurrentTimeStamp return the timestamp from the current block
	CurrentTimeStamp() uint64

	// CurrentRandomSeed returns the random seed from the current header
	CurrentRandomSeed() []byte

	// CurrentEpoch returns the current epoch
	CurrentEpoch() uint32
}
