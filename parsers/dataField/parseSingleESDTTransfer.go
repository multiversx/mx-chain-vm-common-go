package datafield

import (
	"github.com/multiversx/mx-chain-core-go/core"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

func (odp *operationDataFieldParser) parseSingleESDTTransfer(args [][]byte, function string, sender, receiver []byte) *ResponseParseData {
	responseParse, parsedESDTTransfers, ok := odp.extractESDTData(args, function, sender, receiver)
	if !ok {
		return responseParse
	}

	if core.IsSmartContractAddress(receiver) && isASCIIString(parsedESDTTransfers.CallFunction) {
		responseParse.Function = parsedESDTTransfers.CallFunction
	}

	if len(parsedESDTTransfers.ESDTTransfers) == 0 || !isASCIIString(string(parsedESDTTransfers.ESDTTransfers[0].ESDTTokenName)) {
		return responseParse
	}

	firstTransfer := parsedESDTTransfers.ESDTTransfers[0]
	responseParse.Tokens = append(responseParse.Tokens, string(firstTransfer.ESDTTokenName))
	responseParse.ESDTValues = append(responseParse.ESDTValues, firstTransfer.ESDTValue.String())

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
