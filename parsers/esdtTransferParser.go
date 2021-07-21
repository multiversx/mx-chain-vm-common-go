package parsers

import (
	"bytes"
	"math/big"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data/esdt"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

// MinArgsForESDTTransfer defines the minimum arguments needed for an esdt transfer
const MinArgsForESDTTransfer = 2

// MinArgsForESDTNFTTransfer defines the minimum arguments needed for an nft transfer
const MinArgsForESDTNFTTransfer = 4

// MinArgsForMultiESDTNFTTransfer defines the minimum arguments needed for a multi transfer
const MinArgsForMultiESDTNFTTransfer = 4

// ArgsPerTransfer defines the number of arguments per transfer in multi transfer
const ArgsPerTransfer = 3

type esdtTransferParser struct {
	marshalizer vmcommon.Marshalizer
}

// NewESDTTransferParser creates a new esdt transfer parser
func NewESDTTransferParser(
	marshalizer vmcommon.Marshalizer,
) (*esdtTransferParser, error) {
	if check.IfNil(marshalizer) {
		return nil, ErrNilMarshalizer
	}

	return &esdtTransferParser{marshalizer: marshalizer}, nil
}

// ParseESDTTransfers returns the list of esdt transfers, the callFunction and callArgs from the given arguments
func (e *esdtTransferParser) ParseESDTTransfers(
	sndAddr []byte,
	rcvAddr []byte,
	function string,
	args [][]byte,
) (*vmcommon.ParsedESDTTransfers, error) {
	switch function {
	case vmcommon.BuiltInFunctionESDTTransfer:
		return e.parseSingleESDTTransfer(rcvAddr, args)
	case vmcommon.BuiltInFunctionESDTNFTTransfer:
		return e.parseSingleESDTNFTTransfer(sndAddr, rcvAddr, args)
	case vmcommon.BuiltInFunctionMultiESDTNFTTransfer:
		return e.parseMultiESDTNFTTransfer(sndAddr, rcvAddr, args)
	default:
		return nil, ErrNotESDTTransferInput
	}
}

func (e *esdtTransferParser) parseSingleESDTTransfer(rcvAddr []byte, args [][]byte) (*vmcommon.ParsedESDTTransfers, error) {
	if len(args) < MinArgsForESDTTransfer {
		return nil, ErrNotEnoughArguments
	}
	esdtTransfers := &vmcommon.ParsedESDTTransfers{
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

func (e *esdtTransferParser) parseSingleESDTNFTTransfer(sndAddr, rcvAddr []byte, args [][]byte) (*vmcommon.ParsedESDTTransfers, error) {
	if len(args) < MinArgsForESDTNFTTransfer {
		return nil, ErrNotEnoughArguments
	}
	esdtTransfers := &vmcommon.ParsedESDTTransfers{
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

func (e *esdtTransferParser) parseMultiESDTNFTTransfer(sndAddr, rcvAddr []byte, args [][]byte) (*vmcommon.ParsedESDTTransfers, error) {
	if len(args) < MinArgsForMultiESDTNFTTransfer {
		return nil, ErrNotEnoughArguments
	}
	esdtTransfers := &vmcommon.ParsedESDTTransfers{
		RcvAddr:      rcvAddr,
		CallArgs:     make([][]byte, 0),
		CallFunction: "",
	}

	numOfTransfer := big.NewInt(0).SetBytes(args[0])
	startIndex := uint64(1)
	isTxAtSender := false
	if bytes.Equal(sndAddr, rcvAddr) {
		esdtTransfers.RcvAddr = args[0]
		numOfTransfer.SetBytes(args[1])
		startIndex = 2
		isTxAtSender = true
	}

	minLenArgs := ArgsPerTransfer*numOfTransfer.Uint64() + startIndex
	if uint64(len(args)) < minLenArgs {
		return nil, ErrNotEnoughArguments
	}

	if uint64(len(args)) > minLenArgs {
		esdtTransfers.CallFunction = string(args[minLenArgs])
	}
	if uint64(len(args)) > minLenArgs+1 {
		esdtTransfers.CallArgs = append(esdtTransfers.CallArgs, args[minLenArgs+1:]...)
	}

	var err error
	esdtTransfers.ESDTTransfers = make([]*vmcommon.ESDTTransfer, numOfTransfer.Uint64())
	for i := uint64(0); i < numOfTransfer.Uint64(); i++ {
		tokenStartIndex := startIndex + i*ArgsPerTransfer
		esdtTransfers.ESDTTransfers[i], err = e.createNewESDTTransfer(tokenStartIndex, args, isTxAtSender)
		if err != nil {
			return nil, err
		}
	}

	return esdtTransfers, nil
}

func (e *esdtTransferParser) createNewESDTTransfer(
	tokenStartIndex uint64,
	args [][]byte,
	isTxAtSender bool,
) (*vmcommon.ESDTTransfer, error) {
	esdtTransfer := &vmcommon.ESDTTransfer{
		ESDTValue:      big.NewInt(0).SetBytes(args[tokenStartIndex+2]),
		ESDTTokenName:  args[tokenStartIndex],
		ESDTTokenType:  uint32(vmcommon.Fungible),
		ESDTTokenNonce: big.NewInt(0).SetBytes(args[tokenStartIndex+1]).Uint64(),
	}
	if esdtTransfer.ESDTTokenNonce > 0 {
		esdtTransfer.ESDTTokenType = uint32(vmcommon.NonFungible)
		if !isTxAtSender {
			transferESDTData := &esdt.ESDigitalToken{}
			err := e.marshalizer.Unmarshal(transferESDTData, args[tokenStartIndex+2])
			if err != nil {
				return nil, err
			}
			esdtTransfer.ESDTValue.Set(transferESDTData.Value)
		}
	}

	return esdtTransfer, nil
}

// IsInterfaceNil returns true if underlying object is nil
func (e *esdtTransferParser) IsInterfaceNil() bool {
	return e == nil
}
