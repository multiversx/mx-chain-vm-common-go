package datafield

import (
	"bytes"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/sharding"
)

func (odp *operationDataFieldParser) parseSingleESDTNFTTransfer(args [][]byte, function string, sender, receiver []byte, numOfShards uint32) *ResponseParseData {
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
	receiverShardID := sharding.ComputeShardID(rcvAddr, numOfShards)
	token := computeTokenIdentifier(string(esdtNFTTransfer.ESDTTokenName), esdtNFTTransfer.ESDTTokenNonce)

	responseParse.Tokens = append(responseParse.Tokens, token)
	responseParse.ESDTValues = append(responseParse.ESDTValues, esdtNFTTransfer.ESDTValue.String())

	if len(rcvAddr) != len(sender) {
		return responseParse
	}

	responseParse.Receivers = append(responseParse.Receivers, rcvAddr)
	responseParse.ReceiversShardID = append(responseParse.ReceiversShardID, receiverShardID)

	return responseParse
}
