package parsers

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/require"
)

func TestEsdtTransferParser_ParseMultiNFTTransferRejectsOverflowCount(t *testing.T) {
	t.Parallel()

	esdtParser, _ := NewESDTTransferParser(&mock.MarshalizerMock{})
	overflowCount := []byte{1, 0, 0, 0, 0, 0, 0, 0, 0}
	receiver := bytes.Repeat([]byte{2}, 32)

	testCases := []struct {
		name string
		args [][]byte
	}{
		{
			name: "count first",
			args: [][]byte{overflowCount, []byte("tokenID"), []byte{1}, []byte{1}},
		},
		{
			name: "receiver first",
			args: [][]byte{receiver, overflowCount, []byte("tokenID"), []byte{1}, []byte{1}},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			parsedData, err := esdtParser.ParseESDTTransfers(
				nil,
				receiver,
				core.BuiltInFunctionMultiESDTNFTTransfer,
				testCase.args,
			)

			require.Nil(t, parsedData)
			require.Equal(t, ErrTooManyTransfers, err)
		})
	}
}

func TestEsdtTransferParser_ParseMultiNFTTransferRejectsCountAboveLimitBeforeArgumentLengthValidation(t *testing.T) {
	t.Parallel()

	esdtParser, _ := NewESDTTransferParser(&mock.MarshalizerMock{})
	aboveLimitCount := big.NewInt(MaxNumberOfTransferInMultiTransfer + 1).Bytes()
	receiver := bytes.Repeat([]byte{2}, 32)

	testCases := []struct {
		name string
		args [][]byte
	}{
		{
			name: "count first",
			args: [][]byte{aboveLimitCount, []byte("tokenID"), []byte{1}, []byte{1}},
		},
		{
			name: "receiver first",
			args: [][]byte{receiver, aboveLimitCount, []byte("tokenID"), []byte{1}, []byte{1}},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			parsedData, err := esdtParser.ParseESDTTransfers(
				nil,
				receiver,
				core.BuiltInFunctionMultiESDTNFTTransfer,
				testCase.args,
			)

			require.Nil(t, parsedData)
			require.Equal(t, ErrTooManyTransfers, err)
		})
	}
}

func TestEsdtTransferParser_ParseMultiNFTTransferRejectsZeroCount(t *testing.T) {
	t.Parallel()

	esdtParser, _ := NewESDTTransferParser(&mock.MarshalizerMock{})
	receiver := bytes.Repeat([]byte{2}, 32)

	testCases := []struct {
		name string
		args [][]byte
	}{
		{
			name: "count first",
			args: [][]byte{nil, []byte("tokenID"), []byte{1}, []byte{1}},
		},
		{
			name: "receiver first",
			args: [][]byte{receiver, nil, []byte("tokenID"), []byte{1}, []byte{1}},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			parsedData, err := esdtParser.ParseESDTTransfers(
				nil,
				receiver,
				core.BuiltInFunctionMultiESDTNFTTransfer,
				testCase.args,
			)

			require.Nil(t, parsedData)
			require.Equal(t, ErrNotEnoughArguments, err)
		})
	}
}
