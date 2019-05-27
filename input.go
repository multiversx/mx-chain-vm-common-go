package vminterface

import (
	"math/big"
)

// SCCallHeader contains data about the block in which the transaction resides
type SCCallHeader struct {
	Beneficiary *big.Int // "coinbase"
	Number      *big.Int
	GasLimit    *big.Int
	Timestamp   *big.Int
}

// VMInput contains the common fields between the 2 types of SC call
type VMInput struct {
	CallerAddr  []byte
	Arguments   []*big.Int
	CallValue   *big.Int
	GasPrice    *big.Int
	GasProvided *big.Int
	Header      *SCCallHeader
}

// ContractCreateInput VM input when creating a new contract
type ContractCreateInput struct {
	VMInput
	ContractCode []byte
}

// ContractCallInput VM input when calling a function from an existing contract
type ContractCallInput struct {
	VMInput
	RecipientAddr []byte
	Function      string
}
