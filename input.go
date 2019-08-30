package vmcommon

import (
	"math/big"
)

// SCCallHeader contains data about the block in which the transaction resides.
type SCCallHeader struct {
	// Beneficiary is the block proposer.
	// It is referred to as "coinbase" in the test .json files
	// This value is accessible in the VM:
	// - in IELE, it is returned by `call @iele.beneficiary`
	Beneficiary *big.Int

	// Number refers to the block number
	// This value is accessible in the VM:
	// - in IELE, it is returned by `call @iele.number`
	Number *big.Int

	// GasLimit refers to the gas limit of a block
	GasLimit *big.Int

	// Timestamp indicates when the proposer proposed the block.
	// It should be somehow encoded in the blockchain, in order to make VM execution deterministic.
	// This value is accessible in the VM:
	// - in IELE, it is returned by `call @iele.timestamp`
	Timestamp *big.Int
}

// VMInput contains the common fields between the 2 types of SC call.
type VMInput struct {
	// CallerAddr is the public key of the wallet initiating the transaction, "from".
	CallerAddr []byte

	// Arguments are the call parameters to the smart contract function call
	// For contract creation, these are the parameters to the @init function.
	// For contract call, these are the parameters to the function referenced in ContractCallInput.Function.
	// If the number of arguments does not match the function arity,
	// the transaction will return FunctionWrongSignature ReturnCode.
	Arguments []*big.Int

	// CallValue is the value (amount of tokens) transferred by the transaction.
	// The VM knows to subtract this value from sender balance (CallerAddr)
	// and to add it to the smart contract balance.
	// It is often, but not always zero in SC calls.
	CallValue *big.Int

	// GasPrice multiplied by the gas burned by the transaction yields the transaction fee.
	// A larger GasPrice will incentivize block proposers to include the transaction in a block sooner,
	// but will cost the sender more.
	// The total fee should be GasPrice x (GasProvided - VMOutput.GasRemaining - VMOutput.GasRefund).
	// Note: the order of operations on the sender balance is:
	// 1. subtract GasPrice x GasProvided
	// 2. call VM, which will subtract CallValue if enough funds remain
	// 3. reimburse GasPrice x (VMOutput.GasRemaining + VMOutput.GasRefund)
	GasPrice *big.Int

	// GasProvided is the maximum gas allowed for the smart contract execution.
	// If the transaction consumes more gas than this value, it will immediately terminate
	// and return OutOfGas ReturnCode.
	// The sender will not be charged based on GasProvided, only on the gas burned,
	// so it doesn't cost the sender more to have a higher gas limit.
	GasProvided *big.Int

	// Header is the block header info.
	// The same object can be reused in all transaction inputs in a block.
	Header *SCCallHeader
}

// ContractCreateInput VM input when creating a new contract.
// Here we have no RecipientAddr because
// the address (PK) of the created account will be provided by the VM.
// We also do not need to specify a Function field,
// because on creation `init` is always called.
type ContractCreateInput struct {
	VMInput

	// NewContractAddress is the address of the new contract to be created.
	// An empty NewContractAddress will cause the hook to be called.
	// If the hook also returns an empty address, the VM will decide the new address.
	NewContractAddress []byte

	// ContractCode is the code of the contract being created, assembled into a byte array.
	// For Iele VM, to convert a .iele file to this assembled byte array, see
	// src/github.com/ElrondNetwork/elrond-vm/iele/compiler/compiler.AssembleIeleCode
	ContractCode []byte
}

// ContractCallInput VM input when calling a function from an existing contract
type ContractCallInput struct {
	VMInput

	// RecipientAddr is the smart contract public key, "to".
	RecipientAddr []byte

	// Function is the name of the smart contract function that will be called.
	// The function must be public (e.g. in Iele `define public @functionName(...)`)
	Function string
}
