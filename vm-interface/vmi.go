package vminterface

import "math/big"

// VM ... interface for any Elrond VM endpoint
type VM interface {

	// Yields the initial gas cost of creating a new smart contract
	G0Create(input *ContractCreateInput) (*big.Int, error)

	// Yields the initial gas cost of calling an existing new smart contract
	G0Call(input *ContractCallInput) (*big.Int, error)

	// Executes a smart contract creation operation
	CreateSmartContract(input *ContractCreateInput) (*VMOutput, error)

	// Executes a smart contract call
	CreateSmartCall(input *ContractCallInput) (*VMOutput, error)
}
