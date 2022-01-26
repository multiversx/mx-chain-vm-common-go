package datafield

import (
	"bytes"

	"github.com/ElrondNetwork/elrond-go-core/core"
)

func (odp *operationDataFieldParser) parseESDTNFTTransfer(args [][]byte, sender, receiver []byte) *ResponseParseData {
	responseParse := &ResponseParseData{
		Operation: core.BuiltInFunctionESDTNFTTransfer,
	}

	parsedESDTTransfers, err := odp.esdtTransferParser.ParseESDTTransfers(sender, receiver, core.BuiltInFunctionESDTNFTTransfer, args)
	if err != nil {
		return responseParse
	}

	if core.IsSmartContractAddress(parsedESDTTransfers.RcvAddr) {
		responseParse.Function = parsedESDTTransfers.CallFunction
	}

	if len(parsedESDTTransfers.ESDTTransfers) == 0 {
		return responseParse
	}

	rcvAddr := receiver
	if bytes.Equal(sender, receiver) {
		rcvAddr = parsedESDTTransfers.RcvAddr
	}

	esdtNFTTransfer := parsedESDTTransfers.ESDTTransfers[0]
	receiverShardID := odp.shardCoordinator.ComputeId(rcvAddr)
	token := computeTokenIdentifier(string(esdtNFTTransfer.ESDTTokenName), esdtNFTTransfer.ESDTTokenNonce)

	responseParse.Tokens = append(responseParse.Tokens, token)
	responseParse.ESDTValues = append(responseParse.ESDTValues, esdtNFTTransfer.ESDTValue.String())
	responseParse.Receivers = append(responseParse.Receivers, rcvAddr)
	responseParse.ReceiversShardID = append(responseParse.ReceiversShardID, receiverShardID)

	return responseParse
}
