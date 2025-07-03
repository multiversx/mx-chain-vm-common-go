package datafield

import (
	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/marshal"
)

// ArgsOperationDataFieldParser holds all the components required to create a new instance of data field parser
type ArgsOperationDataFieldParser struct {
	AddressLength       int
	Marshalizer         marshal.Marshalizer
	EnableEpochsHandler core.EnableEpochsHandler
}
