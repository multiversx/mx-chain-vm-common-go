package datafield

import (
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

// ResponseParseData is the response with results after the data field was parsed
type ResponseParseData struct {
	Operation        string
	Function         string
	ESDTValues       []string
	Tokens           []string
	Receivers        [][]byte
	ReceiversShardID []uint32
	IsRelayed        bool
}

// ArgsOperationDataFieldParser holds all the components required to create a new instance of data field parser
type ArgsOperationDataFieldParser struct {
	Marshalizer      marshal.Marshalizer
	ShardCoordinator vmcommon.Coordinator
}
