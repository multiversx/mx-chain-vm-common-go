package vmcommon

import "math/big"

// VMExecutionHandler interface for any Elrond VM endpoint
type VMExecutionHandler interface {
	// G0Create yields the initial gas cost of creating a new smart contract
	G0Create(input *ContractCreateInput) (*big.Int, error)

	// G0Call yields the initial gas cost of calling an existing smart contract
	G0Call(input *ContractCallInput) (*big.Int, error)

	// Computes how a smart contract creation should be performed
	RunSmartContractCreate(input *ContractCreateInput) (*VMOutput, error)

	// Computes the result of a smart contract call and how the system must change after the execution
	RunSmartContractCall(input *ContractCallInput) (*VMOutput, error)
}
