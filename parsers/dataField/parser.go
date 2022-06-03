package datafield

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data/transaction"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/parsers"
)

const (
	operationTransfer = `transfer`
	operationDeploy   = `scDeploy`

	minArgumentsQuantityOperationESDT = 2
	minArgumentsQuantityOperationNFT  = 3
	numArgsRelayedV2                  = 4
	receiverAddressIndexRelayedV2     = 0
	dataFieldIndexRelayedV2           = 2

	argsTokenPosition                   = 0
	argsNoncePosition                   = 1
	argsValuePositionNonAndSemiFungible = 2
	argsValuePositionFungible           = 1
)

var errInvalidAddressLength = errors.New("invalid address length")

type operationDataFieldParser struct {
	builtInFunctionsList []string

	addressLength      int
	argsParser         vmcommon.CallArgsParser
	shardCoordinator   vmcommon.Coordinator
	esdtTransferParser vmcommon.ESDTTransferParser
}

// NewOperationDataFieldParser will return a new instance of operationDataFieldParser
func NewOperationDataFieldParser(args *ArgsOperationDataFieldParser) (*operationDataFieldParser, error) {
	if check.IfNil(args.ShardCoordinator) {
		return nil, core.ErrNilShardCoordinator
	}
	if check.IfNil(args.Marshalizer) {
		return nil, core.ErrNilMarshalizer
	}
	if args.AddressLength == 0 {
		return nil, errInvalidAddressLength
	}

	argsParser := parsers.NewCallArgsParser()
	esdtTransferParser, err := parsers.NewESDTTransferParser(args.Marshalizer)
	if err != nil {
		return nil, err
	}

	return &operationDataFieldParser{
		argsParser:           argsParser,
		shardCoordinator:     args.ShardCoordinator,
		esdtTransferParser:   esdtTransferParser,
		addressLength:        args.AddressLength,
		builtInFunctionsList: getAllBuiltInFunctions(),
	}, nil
}

// Parse will parse the provided data field
func (odp *operationDataFieldParser) Parse(dataField []byte, sender, receiver []byte) *ResponseParseData {
	return odp.parse(dataField, sender, receiver, false)
}

func (odp *operationDataFieldParser) parse(dataField []byte, sender, receiver []byte, ignoreRelayed bool) *ResponseParseData {
	responseParse := &ResponseParseData{
		Operation: operationTransfer,
	}

	isSCDeploy := len(dataField) > 0 && isEmptyAddr(odp.addressLength, receiver)
	if isSCDeploy {
		responseParse.Operation = operationDeploy
		return responseParse
	}

	function, args, err := odp.argsParser.ParseData(string(dataField))
	if err != nil {
		return responseParse
	}

	switch function {
	case core.BuiltInFunctionESDTTransfer:
		return odp.parseSingleESDTTransfer(args, function, sender, receiver)
	case core.BuiltInFunctionESDTNFTTransfer:
		return odp.parseSingleESDTNFTTransfer(args, function, sender, receiver)
	case core.BuiltInFunctionMultiESDTNFTTransfer:
		return odp.parseMultiESDTNFTTransfer(args, function, sender, receiver)
	case core.BuiltInFunctionESDTLocalBurn, core.BuiltInFunctionESDTLocalMint:
		return parseQuantityOperationESDT(args, function)
	case core.BuiltInFunctionESDTWipe, core.BuiltInFunctionESDTFreeze, core.BuiltInFunctionESDTUnFreeze:
		return parseBlockingOperationESDT(args, function)
	case core.BuiltInFunctionESDTNFTCreate, core.BuiltInFunctionESDTNFTBurn, core.BuiltInFunctionESDTNFTAddQuantity:
		return parseQuantityOperationNFT(args, function)
	case core.RelayedTransaction, core.RelayedTransactionV2:
		if ignoreRelayed {
			return NewResponseParseDataAsRelayed()
		}
		return odp.parseRelayed(function, args, receiver)
	}

	isBuiltInFunc := isBuiltInFunction(odp.builtInFunctionsList, function)
	if isBuiltInFunc {
		responseParse.Operation = function
	}

	if function != "" && core.IsSmartContractAddress(receiver) && isASCIIString(function) {
		responseParse.Function = function
	}

	return responseParse
}

func (odp *operationDataFieldParser) parseRelayed(function string, args [][]byte, receiver []byte) *ResponseParseData {
	if len(args) == 0 {
		return &ResponseParseData{
			IsRelayed: true,
		}
	}

	tx, ok := extractInnerTx(function, args, receiver)
	if !ok {
		return &ResponseParseData{
			IsRelayed: true,
		}
	}

	res := odp.parse(tx.Data, tx.SndAddr, tx.RcvAddr, true)
	if res.IsRelayed {
		return &ResponseParseData{
			IsRelayed: true,
		}
	}

	receivers := [][]byte{tx.RcvAddr}
	receiversShardID := []uint32{odp.shardCoordinator.ComputeId(tx.RcvAddr)}
	if res.Operation == core.BuiltInFunctionMultiESDTNFTTransfer || res.Operation == core.BuiltInFunctionESDTNFTTransfer {
		receivers = res.Receivers
		receiversShardID = res.ReceiversShardID
	}

	return &ResponseParseData{
		Operation:        res.Operation,
		Function:         res.Function,
		ESDTValues:       res.ESDTValues,
		Tokens:           res.Tokens,
		Receivers:        receivers,
		ReceiversShardID: receiversShardID,
		IsRelayed:        true,
	}
}

func extractInnerTx(function string, args [][]byte, receiver []byte) (*transaction.Transaction, bool) {
	tx := &transaction.Transaction{}

	if function == core.RelayedTransaction {
		err := json.Unmarshal(args[0], &tx)

		return tx, err == nil
	}

	if len(args) != numArgsRelayedV2 {
		return nil, false
	}

	// sender of the inner tx is the receiver of the relayed tx
	tx.SndAddr = receiver
	tx.RcvAddr = args[receiverAddressIndexRelayedV2]
	tx.Data = args[dataFieldIndexRelayedV2]

	return tx, true
}

func parseBlockingOperationESDT(args [][]byte, funcName string) *ResponseParseData {
	responseData := &ResponseParseData{
		Operation: funcName,
	}

	if len(args) == 0 {
		return responseData
	}

	token, nonce := extractTokenAndNonce(args[argsTokenPosition])
	if !isASCIIString(token) {
		return responseData
	}

	if nonce != 0 {
		token = computeTokenIdentifier(token, nonce)
	}

	responseData.Tokens = append(responseData.Tokens, token)
	return responseData
}

func parseQuantityOperationESDT(args [][]byte, funcName string) *ResponseParseData {
	responseData := &ResponseParseData{
		Operation: funcName,
	}

	if len(args) < minArgumentsQuantityOperationESDT {
		return responseData
	}

	token := string(args[argsTokenPosition])
	if !isASCIIString(token) {
		return responseData
	}

	responseData.Tokens = append(responseData.Tokens, token)
	responseData.ESDTValues = append(responseData.ESDTValues, big.NewInt(0).SetBytes(args[argsValuePositionFungible]).String())

	return responseData
}

func parseQuantityOperationNFT(args [][]byte, funcName string) *ResponseParseData {
	responseData := &ResponseParseData{
		Operation: funcName,
	}

	if len(args) < minArgumentsQuantityOperationNFT {
		return responseData
	}

	token := string(args[argsTokenPosition])
	if !isASCIIString(token) {
		return responseData
	}

	nonce := big.NewInt(0).SetBytes(args[argsNoncePosition]).Uint64()
	tokenIdentifier := computeTokenIdentifier(token, nonce)
	responseData.Tokens = append(responseData.Tokens, tokenIdentifier)

	value := big.NewInt(0).SetBytes(args[argsValuePositionNonAndSemiFungible]).String()
	responseData.ESDTValues = append(responseData.ESDTValues, value)

	return responseData
}
