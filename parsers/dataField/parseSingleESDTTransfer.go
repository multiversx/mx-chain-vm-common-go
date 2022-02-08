package datafield

import (
	"github.com/ElrondNetwork/elrond-go-core/core"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

func (odp *operationDataFieldParser) parseSingleESDTTransfer(args [][]byte, function string, sender, receiver []byte) *ResponseParseData {
	responseParse, parsedESDTTransfers, ok := odp.extractESDTData(args, function, sender, receiver)
	if !ok {
		return responseParse
	}

	if core.IsSmartContractAddress(receiver) {
		responseParse.Function = parsedESDTTransfers.CallFunction
	}

	if len(parsedESDTTransfers.ESDTTransfers) == 0 {
		return responseParse
	}
	responseParse.Tokens = append(responseParse.Tokens, string(parsedESDTTransfers.ESDTTransfers[0].ESDTTokenName))
	responseParse.ESDTValues = append(responseParse.ESDTValues, parsedESDTTransfers.ESDTTransfers[0].ESDTValue.String())

	return responseParse
}

func (odp *operationDataFieldParser) extractESDTData(args [][]byte, function string, sender, receiver []byte) (*ResponseParseData, *vmcommon.ParsedESDTTransfers, bool) {
	responseParse := &ResponseParseData{
		Operation: function,
	}

	parsedESDTTransfers, err := odp.esdtTransferParser.ParseESDTTransfers(sender, receiver, function, args)
	if err != nil {
		return responseParse, nil, false
	}

	return responseParse, parsedESDTTransfers, true
}
