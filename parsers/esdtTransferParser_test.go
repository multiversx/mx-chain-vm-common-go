package parsers

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/data/esdt"
	"github.com/ElrondNetwork/elrond-vm-common/mock"
	"github.com/stretchr/testify/assert"
)

var sndAddr = bytes.Repeat([]byte{1}, 32)
var dstAddr = bytes.Repeat([]byte{1}, 32)

func TestNewESDTTransferParser(t *testing.T) {
	t.Parallel()

	esdtParser, err := NewESDTTransferParser(nil)
	assert.Nil(t, esdtParser)
	assert.Equal(t, err, ErrNilMarshalizer)

	esdtParser, err = NewESDTTransferParser(&mock.MarshalizerMock{})
	assert.Nil(t, err)
	assert.False(t, esdtParser.IsInterfaceNil())
}

func TestEsdtTransferParser_ParseESDTTransfersWrongFunction(t *testing.T) {
	t.Parallel()

	esdtParser, _ := NewESDTTransferParser(&mock.MarshalizerMock{})
	parsedData, err := esdtParser.ParseESDTTransfers(nil, nil, "some", nil)
	assert.Equal(t, err, ErrNotESDTTransferInput)
	assert.Nil(t, parsedData)
}

func TestEsdtTransferParser_ParseSingleESDTFunction(t *testing.T) {
	t.Parallel()

	esdtParser, _ := NewESDTTransferParser(&mock.MarshalizerMock{})
	parsedData, err := esdtParser.ParseESDTTransfers(
		nil,
		dstAddr,
		core.BuiltInFunctionESDTTransfer,
		[][]byte{[]byte("one")},
	)
	assert.Equal(t, err, ErrNotEnoughArguments)
	assert.Nil(t, parsedData)

	parsedData, err = esdtParser.ParseESDTTransfers(
		nil,
		dstAddr,
		core.BuiltInFunctionESDTTransfer,
		[][]byte{[]byte("one"), big.NewInt(10).Bytes()},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.ESDTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 0)
	assert.Equal(t, parsedData.RcvAddr, dstAddr)
	assert.Equal(t, parsedData.ESDTTransfers[0].ESDTValue.Uint64(), big.NewInt(10).Uint64())

	parsedData, err = esdtParser.ParseESDTTransfers(
		nil,
		dstAddr,
		core.BuiltInFunctionESDTTransfer,
		[][]byte{[]byte("one"), big.NewInt(10).Bytes(), []byte("function"), []byte("arg")},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.ESDTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 1)
	assert.Equal(t, parsedData.CallFunction, "function")
}

func TestEsdtTransferParser_ParseSingleNFTTransfer(t *testing.T) {
	t.Parallel()

	esdtParser, _ := NewESDTTransferParser(&mock.MarshalizerMock{})
	parsedData, err := esdtParser.ParseESDTTransfers(
		nil,
		dstAddr,
		core.BuiltInFunctionESDTNFTTransfer,
		[][]byte{[]byte("one"), []byte("two")},
	)
	assert.Equal(t, err, ErrNotEnoughArguments)
	assert.Nil(t, parsedData)

	parsedData, err = esdtParser.ParseESDTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionESDTNFTTransfer,
		[][]byte{[]byte("one"), big.NewInt(10).Bytes(), big.NewInt(10).Bytes(), dstAddr},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.ESDTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 0)
	assert.Equal(t, parsedData.RcvAddr, dstAddr)
	assert.Equal(t, parsedData.ESDTTransfers[0].ESDTValue.Uint64(), big.NewInt(10).Uint64())
	assert.Equal(t, parsedData.ESDTTransfers[0].ESDTTokenNonce, big.NewInt(10).Uint64())

	parsedData, err = esdtParser.ParseESDTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionESDTNFTTransfer,
		[][]byte{[]byte("one"), big.NewInt(10).Bytes(), big.NewInt(10).Bytes(), dstAddr, []byte("function"), []byte("arg")})
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.ESDTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 1)
	assert.Equal(t, parsedData.CallFunction, "function")

	parsedData, err = esdtParser.ParseESDTTransfers(
		sndAddr,
		dstAddr,
		core.BuiltInFunctionESDTNFTTransfer,
		[][]byte{[]byte("one"), big.NewInt(10).Bytes(), big.NewInt(10).Bytes(), dstAddr, []byte("function"), []byte("arg")})
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.ESDTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 1)
	assert.Equal(t, parsedData.RcvAddr, dstAddr)
	assert.Equal(t, parsedData.ESDTTransfers[0].ESDTValue.Uint64(), big.NewInt(10).Uint64())
	assert.Equal(t, parsedData.ESDTTransfers[0].ESDTTokenNonce, big.NewInt(10).Uint64())
}

func TestEsdtTransferParser_ParseMultiNFTTransferTransferOne(t *testing.T) {
	t.Parallel()

	esdtParser, _ := NewESDTTransferParser(&mock.MarshalizerMock{})
	parsedData, err := esdtParser.ParseESDTTransfers(
		nil,
		sndAddr,
		core.BuiltInFunctionMultiESDTNFTTransfer,
		[][]byte{[]byte("one"), []byte("two")},
	)
	assert.Equal(t, err, ErrNotEnoughArguments)
	assert.Nil(t, parsedData)

	parsedData, err = esdtParser.ParseESDTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionMultiESDTNFTTransfer,
		[][]byte{dstAddr, big.NewInt(1).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes()},
	)
	assert.Equal(t, err, ErrNotEnoughArguments)
	assert.Nil(t, parsedData)

	parsedData, err = esdtParser.ParseESDTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionMultiESDTNFTTransfer,
		[][]byte{dstAddr, big.NewInt(1).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), big.NewInt(20).Bytes()},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.ESDTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 0)
	assert.Equal(t, parsedData.RcvAddr, dstAddr)
	assert.Equal(t, parsedData.ESDTTransfers[0].ESDTValue.Uint64(), big.NewInt(20).Uint64())
	assert.Equal(t, parsedData.ESDTTransfers[0].ESDTTokenNonce, big.NewInt(10).Uint64())

	parsedData, err = esdtParser.ParseESDTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionMultiESDTNFTTransfer,
		[][]byte{dstAddr, big.NewInt(1).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), big.NewInt(20).Bytes(), []byte("function"), []byte("arg")})
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.ESDTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 1)
	assert.Equal(t, parsedData.CallFunction, "function")

	esdtData := &esdt.ESDigitalToken{Value: big.NewInt(20)}
	marshaled, _ := esdtParser.marshalizer.Marshal(esdtData)

	parsedData, err = esdtParser.ParseESDTTransfers(
		sndAddr,
		dstAddr,
		core.BuiltInFunctionMultiESDTNFTTransfer,
		[][]byte{big.NewInt(1).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), marshaled, []byte("function"), []byte("arg")})
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.ESDTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 1)
	assert.Equal(t, parsedData.RcvAddr, dstAddr)
	assert.Equal(t, parsedData.ESDTTransfers[0].ESDTValue.Uint64(), big.NewInt(20).Uint64())
	assert.Equal(t, parsedData.ESDTTransfers[0].ESDTTokenNonce, big.NewInt(10).Uint64())
}

func TestEsdtTransferParser_ParseMultiNFTTransferTransferMore(t *testing.T) {
	t.Parallel()

	esdtParser, _ := NewESDTTransferParser(&mock.MarshalizerMock{})
	parsedData, err := esdtParser.ParseESDTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionMultiESDTNFTTransfer,
		[][]byte{dstAddr, big.NewInt(2).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), big.NewInt(20).Bytes()},
	)
	assert.Equal(t, err, ErrNotEnoughArguments)
	assert.Nil(t, parsedData)

	parsedData, err = esdtParser.ParseESDTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionMultiESDTNFTTransfer,
		[][]byte{dstAddr, big.NewInt(2).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), big.NewInt(20).Bytes(), []byte("tokenID"), big.NewInt(0).Bytes(), big.NewInt(20).Bytes()},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.ESDTTransfers), 2)
	assert.Equal(t, len(parsedData.CallArgs), 0)
	assert.Equal(t, parsedData.RcvAddr, dstAddr)
	assert.Equal(t, parsedData.ESDTTransfers[0].ESDTValue.Uint64(), big.NewInt(20).Uint64())
	assert.Equal(t, parsedData.ESDTTransfers[0].ESDTTokenNonce, big.NewInt(10).Uint64())
	assert.Equal(t, parsedData.ESDTTransfers[1].ESDTValue.Uint64(), big.NewInt(20).Uint64())
	assert.Equal(t, parsedData.ESDTTransfers[1].ESDTTokenNonce, uint64(0))
	assert.Equal(t, parsedData.ESDTTransfers[1].ESDTTokenType, uint32(core.Fungible))

	parsedData, err = esdtParser.ParseESDTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionMultiESDTNFTTransfer,
		[][]byte{dstAddr, big.NewInt(2).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), big.NewInt(20).Bytes(), []byte("tokenID"), big.NewInt(0).Bytes(), big.NewInt(20).Bytes(), []byte("function"), []byte("arg")},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.ESDTTransfers), 2)
	assert.Equal(t, len(parsedData.CallArgs), 1)
	assert.Equal(t, parsedData.CallFunction, "function")

	esdtData := &esdt.ESDigitalToken{Value: big.NewInt(20)}
	marshaled, _ := esdtParser.marshalizer.Marshal(esdtData)
	parsedData, err = esdtParser.ParseESDTTransfers(
		sndAddr,
		dstAddr,
		core.BuiltInFunctionMultiESDTNFTTransfer,
		[][]byte{big.NewInt(2).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), marshaled, []byte("tokenID"), big.NewInt(0).Bytes(), big.NewInt(20).Bytes()},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.ESDTTransfers), 2)
	assert.Equal(t, len(parsedData.CallArgs), 0)
	assert.Equal(t, parsedData.RcvAddr, dstAddr)
	assert.Equal(t, parsedData.ESDTTransfers[0].ESDTValue.Uint64(), big.NewInt(20).Uint64())
	assert.Equal(t, parsedData.ESDTTransfers[0].ESDTTokenNonce, big.NewInt(10).Uint64())
	assert.Equal(t, parsedData.ESDTTransfers[1].ESDTValue.Uint64(), big.NewInt(20).Uint64())
	assert.Equal(t, parsedData.ESDTTransfers[1].ESDTTokenNonce, uint64(0))
	assert.Equal(t, parsedData.ESDTTransfers[1].ESDTTokenType, uint32(core.Fungible))

	parsedData, err = esdtParser.ParseESDTTransfers(
		sndAddr,
		dstAddr,
		core.BuiltInFunctionMultiESDTNFTTransfer,
		[][]byte{big.NewInt(2).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), marshaled, []byte("tokenID"), big.NewInt(0).Bytes(), big.NewInt(20).Bytes(), []byte("function"), []byte("arg")},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.ESDTTransfers), 2)
	assert.Equal(t, len(parsedData.CallArgs), 1)
	assert.Equal(t, parsedData.CallFunction, "function")
}
