package vmcommon

import (
	"math/big"

	"github.com/ElrondNetwork/elrond-go-core/data/vm"
)

// VMInput contains the common fields between the 2 types of SC call.
type VMInput struct {
	// CallerAddr is the public key of the wallet initiating the transaction, "from".
	CallerAddr []byte

	// Arguments are the call parameters to the smart contract function call
	// For contract creation, these are the parameters to the @init function.
	// For contract call, these are the parameters to the function referenced in ContractCallInput.Function.
	// If the number of arguments does not match the function arity,
	// the transaction will return FunctionWrongSignature ReturnCode.
	Arguments [][]byte

	// CallValue is the eGLD value (amount of tokens) transferred by the transaction.
	// Before reaching the VM this value is subtracted from sender balance (CallerAddr)
	// and to added to the smart contract balance.
	// It is often, but not always zero in SC calls.
	CallValue *big.Int

	// CallType is the type of SmartContract call
	// Based on this value, the VM is informed of whether the call is direct,
	// asynchronous, or asynchronous callback.
	CallType vm.CallType

	// GasPrice multiplied by the gas burned by the transaction yields the transaction fee.
	// A larger GasPrice will incentivize block proposers to include the transaction in a block sooner,
	// but will cost the sender more.
	// The total fee should be GasPrice x (GasProvided - VMOutput.GasRemaining - VMOutput.GasRefund).
	// Note: the order of operations on the sender balance is:
	// 1. subtract GasPrice x GasProvided
	// 2. call VM, which will subtract CallValue if enough funds remain
	// 3. reimburse GasPrice x (VMOutput.GasRemaining + VMOutput.GasRefund)
	GasPrice uint64

	// GasProvided is the maximum gas allowed for the smart contract execution.
	// If the transaction consumes more gas than this value, it will immediately terminate
	// and return OutOfGas ReturnCode.
	// The sender will not be charged based on GasProvided, only on the gas burned,
	// so it doesn't cost the sender more to have a higher gas limit.
	GasProvided uint64

	// GasLocked is the amount of gas that must be kept unused during the current
	// call, because it will be used later for a callback. This field is only
	// used during asynchronous calls.
	GasLocked uint64

	// OriginalTxHash
	OriginalTxHash []byte

	// CurrentTxHash
	CurrentTxHash []byte

	// PrevTxHash
	PrevTxHash []byte

	// ESDTTransfers
	ESDTTransfers []*ESDTTransfer

	// ReturnCallAfterError
	ReturnCallAfterError bool
}

// ESDTTransfer defines the structure for and ESDT / NFT transfer
type ESDTTransfer struct {
	// ESDTValue is the value (amount of tokens) transferred by the transaction.
	// Before reaching the VM this value is subtracted from sender balance (CallerAddr)
	// and to added to the smart contract balance.
	// It is often, but not always zero in SC calls.
	ESDTValue *big.Int

	// ESDTTokenName is the name of the token which was transferred by the transaction to the SC
	ESDTTokenName []byte

	// ESDTTokenType is the type of the transferred token
	ESDTTokenType uint32

	// ESDTTokenNonce is the nonce for the given NFT token
	ESDTTokenNonce uint64
}

// ContractCreateInput VM input when creating a new contract.
// Here we have no RecipientAddr because
// the address (PK) of the created account will be provided by the vmcommon.
// We also do not need to specify a Function field,
// because on creation `init` is always called.
type ContractCreateInput struct {
	VMInput

	// ContractCode is the code of the contract being created, assembled into a byte array.
	ContractCode []byte

	// ContractCodeMetadata is the code metadata of the contract being created.
	ContractCodeMetadata []byte
}

// ContractCallInput VM input when calling a function from an existing contract
type ContractCallInput struct {
	VMInput

	// RecipientAddr is the smart contract public key, "to".
	RecipientAddr []byte

	// Function is the name of the smart contract function that will be called.
	// The function must be public
	Function string

	// AllowInitFunction specifies whether calling the initialization method of
	// the smart contract is allowed or not
	AllowInitFunction bool
}

// ParsedESDTTransfers defines the struct for the parsed esdt transfers
type ParsedESDTTransfers struct {
	ESDTTransfers []*ESDTTransfer
	RcvAddr       []byte
	CallFunction  string
	CallArgs      [][]byte
}
