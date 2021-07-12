package parsers

import (
	"bytes"
	"errors"
	"math/big"

	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

type ParsedESDTTransfers struct {
	ESDTTransfers []*vmcommon.ESDTTransfer
	RcvAddr       []byte
	CallFunction  string
	CallArgs      [][]byte
}

// ErrNotESDTTransferInput defines the error for not ESDT transfer input
var ErrNotESDTTransferInput = errors.New("not ESDT transfer input")

// ErrNotEnoughArguments defines the error for not enough arguments
var ErrNotEnoughArguments = errors.New("not enough arguments")

// MinArgsForESDTTransfer defines the minimum arguments needed for an esdt transfer
const MinArgsForESDTTransfer = 2

// MinArgsForESDTNFTTransfer defines the minimum arguments needed for an nft transfer
const MinArgsForESDTNFTTransfer = 4

// MinArgsForMultiESDTNFTTransfer defines the minimum arguments needed for a multi transfer
const MinArgsForMultiESDTNFTTransfer = 4

// ArgsPerTransfer defines the number of arguments per transfer in multi transfer
const ArgsPerTransfer = 3

// ParseESDTTransfers returns the list of esdt transfers, the callFunction and callArgs from the given arguments
func ParseESDTTransfers(
	sndAddr []byte,
	rcvAddr []byte,
	function string,
	args [][]byte,
) (*ParsedESDTTransfers, error) {
	switch function {
	case vmcommon.BuiltInFunctionESDTTransfer:
		return parseSingleESDTTransfer(rcvAddr, args)
	case vmcommon.BuiltInFunctionESDTNFTTransfer:
		return parseSingleESDTNFTTransfer(sndAddr, rcvAddr, args)
	case vmcommon.BuiltInFunctionMultiESDTNFTTransfer:
		return parseMultiESDTNFTTransfer(sndAddr, rcvAddr, args)
	default:
		return nil, ErrNotESDTTransferInput
	}
}

func parseSingleESDTTransfer(rcvAddr []byte, args [][]byte) (*ParsedESDTTransfers, error) {
	if len(args) < MinArgsForESDTTransfer {
		return nil, ErrNotEnoughArguments
	}
	esdtTransfers := &ParsedESDTTransfers{
		ESDTTransfers: make([]*vmcommon.ESDTTransfer, 1),
		RcvAddr:       rcvAddr,
		CallArgs:      make([][]byte, 0),
		CallFunction:  "",
	}
	if len(args) > MinArgsForESDTTransfer {
		esdtTransfers.CallFunction = string(args[MinArgsForESDTTransfer])
	}
	if len(args) > MinArgsForESDTTransfer+1 {
		esdtTransfers.CallArgs = append(esdtTransfers.CallArgs, args[MinArgsForESDTTransfer+1:]...)
	}
	esdtTransfers.ESDTTransfers[0] = &vmcommon.ESDTTransfer{
		ESDTValue:      big.NewInt(0).SetBytes(args[1]),
		ESDTTokenName:  args[0],
		ESDTTokenType:  uint32(vmcommon.Fungible),
		ESDTTokenNonce: 0,
	}

	return esdtTransfers, nil
}

func parseSingleESDTNFTTransfer(sndAddr, rcvAddr []byte, args [][]byte) (*ParsedESDTTransfers, error) {
	if len(args) < MinArgsForESDTNFTTransfer {
		return nil, ErrNotEnoughArguments
	}
	esdtTransfers := &ParsedESDTTransfers{
		ESDTTransfers: make([]*vmcommon.ESDTTransfer, 1),
		RcvAddr:       rcvAddr,
		CallArgs:      make([][]byte, 0),
		CallFunction:  "",
	}

	if bytes.Equal(sndAddr, rcvAddr) {
		esdtTransfers.RcvAddr = args[3]
	}
	if len(args) > MinArgsForESDTNFTTransfer {
		esdtTransfers.CallFunction = string(args[MinArgsForESDTNFTTransfer])
	}
	if len(args) > MinArgsForESDTNFTTransfer+1 {
		esdtTransfers.CallArgs = append(esdtTransfers.CallArgs, args[MinArgsForESDTNFTTransfer+1:]...)
	}
	esdtTransfers.ESDTTransfers[0] = &vmcommon.ESDTTransfer{
		ESDTValue:      big.NewInt(0).SetBytes(args[2]),
		ESDTTokenName:  args[0],
		ESDTTokenType:  uint32(vmcommon.NonFungible),
		ESDTTokenNonce: big.NewInt(0).SetBytes(args[1]).Uint64(),
	}

	return esdtTransfers, nil
}

func parseMultiESDTNFTTransfer(sndAddr, rcvAddr []byte, args [][]byte) (*ParsedESDTTransfers, error) {
	if len(args) < MinArgsForMultiESDTNFTTransfer {
		return nil, ErrNotEnoughArguments
	}
	esdtTransfers := &ParsedESDTTransfers{
		RcvAddr:      rcvAddr,
		CallArgs:     make([][]byte, 0),
		CallFunction: "",
	}

	numOfTransfer := big.NewInt(0).SetBytes(args[0])
	startIndex := uint64(1)
	if bytes.Equal(sndAddr, rcvAddr) {
		esdtTransfers.RcvAddr = args[0]
		numOfTransfer.SetBytes(args[1])
		startIndex = 0
	}

	minLenArgs := ArgsPerTransfer * numOfTransfer.Uint64()

	if uint64(len(args)) > minLenArgs {
		esdtTransfers.CallFunction = string(args[minLenArgs])
	}
	if uint64(len(args)) > minLenArgs+1 {
		esdtTransfers.CallArgs = append(esdtTransfers.CallArgs, args[minLenArgs+1:]...)
	}

	esdtTransfers.ESDTTransfers = make([]*vmcommon.ESDTTransfer, numOfTransfer.Uint64())
	for i := startIndex; i < numOfTransfer.Uint64(); i++ {
		tokenStartIndex := startIndex + i*ArgsPerTransfer
		esdtTransfers.ESDTTransfers[i] = &vmcommon.ESDTTransfer{
			ESDTValue:      big.NewInt(0).SetBytes(args[tokenStartIndex+2]),
			ESDTTokenName:  args[tokenStartIndex],
			ESDTTokenType:  uint32(vmcommon.Fungible),
			ESDTTokenNonce: big.NewInt(0).SetBytes(args[tokenStartIndex+1]).Uint64(),
		}
		if esdtTransfers.ESDTTransfers[i].ESDTTokenNonce > 0 {
			esdtTransfers.ESDTTransfers[i].ESDTTokenType = uint32(vmcommon.NonFungible)
		}
	}

	return esdtTransfers, nil
}
