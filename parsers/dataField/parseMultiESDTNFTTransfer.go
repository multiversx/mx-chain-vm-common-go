package datafield

import (
	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/sharding"
)

func (odp *operationDataFieldParser) parseMultiESDTNFTTransfer(args [][]byte, function string, sender, receiver []byte, numOfShards uint32) *ResponseParseData {
	responseParse, parsedESDTTransfers, ok := odp.extractESDTData(args, function, sender, receiver)
	if !ok {
		return responseParse
	}
	if core.IsSmartContractAddress(parsedESDTTransfers.RcvAddr) && isASCIIString(parsedESDTTransfers.CallFunction) {
		responseParse.Function = parsedESDTTransfers.CallFunction
	}

	receiverShardID := sharding.ComputeShardID(parsedESDTTransfers.RcvAddr, numOfShards)
	for _, esdtTransferData := range parsedESDTTransfers.ESDTTransfers {
		if !isASCIIString(string(esdtTransferData.ESDTTokenName)) {
			return &ResponseParseData{
				Operation: function,
			}
		}

		token := string(esdtTransferData.ESDTTokenName)
		if esdtTransferData.ESDTTokenNonce != 0 {
			token = computeTokenIdentifier(token, esdtTransferData.ESDTTokenNonce)
		}

		responseParse.Tokens = append(responseParse.Tokens, token)
		responseParse.ESDTValues = append(responseParse.ESDTValues, esdtTransferData.ESDTValue.String())
		responseParse.Receivers = append(responseParse.Receivers, parsedESDTTransfers.RcvAddr)
		responseParse.ReceiversShardID = append(responseParse.ReceiversShardID, receiverShardID)
	}

	return responseParse
}
