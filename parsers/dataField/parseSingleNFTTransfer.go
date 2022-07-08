package datafield

import (
	"bytes"

	"github.com/ElrondNetwork/elrond-go-core/core"
)

func (odp *operationDataFieldParser) parseSingleESDTNFTTransfer(args [][]byte, function string, sender, receiver []byte) *ResponseParseData {
	responseParse, parsedESDTTransfers, ok := odp.extractESDTData(args, function, sender, receiver)
	if !ok {
		return responseParse
	}

	if core.IsSmartContractAddress(parsedESDTTransfers.RcvAddr) && isASCIIString(parsedESDTTransfers.CallFunction) {
		responseParse.Function = parsedESDTTransfers.CallFunction
	}

	if len(parsedESDTTransfers.ESDTTransfers) == 0 || !isASCIIString(string(parsedESDTTransfers.ESDTTransfers[0].ESDTTokenName)) {
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
