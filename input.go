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

// ContractCreateInput VM input when creating a new contract
type ContractCreateInput struct {
	CallerAddr   []byte
	ContractCode []byte
	Arguments    []*big.Int
	CallValue    *big.Int
	GasPrice     *big.Int
	GasProvided  *big.Int
	Header       *SCCallHeader
}

// ContractCallInput VM input when calling a function from an existing contract
type ContractCallInput struct {
	CallerAddr    []byte
	RecipientAddr []byte
	Function      string
	Arguments     []*big.Int
	CallValue     *big.Int
	GasPrice      *big.Int
	GasProvided   *big.Int
	Header        *SCCallHeader
}
