package vmcommon

import (
	"math"
	"math/big"
	"testing"

	"github.com/multiversx/mx-chain-core-go/data/vm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFirstReturnData_VMOutputWithNoReturnDataShouldErr(t *testing.T) {
	vmOutput := VMOutput{
		ReturnData: [][]byte{},
	}

	_, err := vmOutput.GetFirstReturnData(vm.AsBigInt)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "no return data")
}

func TestGetFirstReturnData_WithBadReturnDataKindShouldErr(t *testing.T) {
	vmOutput := VMOutput{
		ReturnData: [][]byte{[]byte("100")},
	}

	_, err := vmOutput.GetFirstReturnData(42)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "can't interpret")
}

func TestGetFirstReturnData(t *testing.T) {
	value := big.NewInt(100)

	vmOutput := VMOutput{
		ReturnData: [][]byte{value.Bytes()},
	}

	dataAsBigInt, _ := vmOutput.GetFirstReturnData(vm.AsBigInt)
	dataAsBigIntString, _ := vmOutput.GetFirstReturnData(vm.AsBigIntString)
	dataAsString, _ := vmOutput.GetFirstReturnData(vm.AsString)
	dataAsHex, _ := vmOutput.GetFirstReturnData(vm.AsHex)

	assert.Equal(t, value, dataAsBigInt)
	assert.Equal(t, "100", dataAsBigIntString)
	assert.Equal(t, string(value.Bytes()), dataAsString)
	assert.Equal(t, "64", dataAsHex)
}

func TestOutputContext_MergeCompleteAccounts(t *testing.T) {
	t.Parallel()

	transfer1 := OutputTransfer{
		Value:    big.NewInt(0),
		GasLimit: 9999,
		Data:     []byte("data1"),
	}
	left := &OutputAccount{
		Address:         []byte("addr1"),
		Nonce:           1,
		Balance:         big.NewInt(1000),
		BalanceDelta:    big.NewInt(10000),
		StorageUpdates:  nil,
		Code:            []byte("code1"),
		OutputTransfers: []OutputTransfer{transfer1},
	}
	right := &OutputAccount{
		Address:         []byte("addr2"),
		Nonce:           2,
		Balance:         big.NewInt(2000),
		BalanceDelta:    big.NewInt(20000),
		StorageUpdates:  map[string]*StorageUpdate{"key": {Data: []byte("data"), Offset: []byte("offset")}},
		Code:            []byte("code2"),
		OutputTransfers: []OutputTransfer{transfer1, transfer1},
	}

	expected := &OutputAccount{
		Address:         []byte("addr2"),
		Nonce:           2,
		Balance:         big.NewInt(2000),
		BalanceDelta:    big.NewInt(30000),
		StorageUpdates:  map[string]*StorageUpdate{"key": {Data: []byte("data"), Offset: []byte("offset")}},
		Code:            []byte("code2"),
		OutputTransfers: []OutputTransfer{transfer1, transfer1},
	}

	left.MergeOutputAccounts(right)
	require.Equal(t, expected, left)
}

func TestOutputAccount_MergeStorageUpdates_DeepCopy(t *testing.T) {
	t.Parallel()

	left := &OutputAccount{
		Address: []byte("addr1"),
	}
	originalOffset := []byte("offset")
	originalData := []byte("data")
	right := &OutputAccount{
		Address: []byte("addr1"),
		StorageUpdates: map[string]*StorageUpdate{
			"k1": {
				Offset:  originalOffset,
				Data:    originalData,
				Written: true,
			},
		},
	}

	left.MergeStorageUpdates(right)
	require.NotNil(t, left.StorageUpdates)
	require.NotNil(t, left.StorageUpdates["k1"])
	require.Equal(t, []byte("offset"), left.StorageUpdates["k1"].Offset)
	require.Equal(t, []byte("data"), left.StorageUpdates["k1"].Data)
	require.True(t, left.StorageUpdates["k1"].Written)

	// Mutate right's storage update to confirm left is independent
	right.StorageUpdates["k1"].Data[0] = 'X'
	right.StorageUpdates["k1"].Offset[0] = 'Y'
	right.StorageUpdates["k1"].Written = false

	require.Equal(t, []byte("offset"), left.StorageUpdates["k1"].Offset)
	require.Equal(t, []byte("data"), left.StorageUpdates["k1"].Data)
	require.True(t, left.StorageUpdates["k1"].Written)
}

type stubTransferIndexer struct {
	crtIndex uint32
}

func (s *stubTransferIndexer) NextOutputTransferIndex() uint32 {
	s.crtIndex++
	return s.crtIndex
}

func (s *stubTransferIndexer) GetCrtTransferIndex() uint32 {
	return s.crtIndex
}

func (s *stubTransferIndexer) SetCrtTransferIndex(idx uint32) {
	s.crtIndex = idx
}

func (s *stubTransferIndexer) IsInterfaceNil() bool {
	return s == nil
}

func TestVMOutput_ReindexTransfers(t *testing.T) {
	t.Parallel()

	t.Run("nil nextIndexProvider returns ErrNilTransferIndexer", func(t *testing.T) {
		t.Parallel()
		output := &VMOutput{}
		err := output.ReindexTransfers(nil)
		require.Equal(t, ErrNilTransferIndexer, err)
	})

	t.Run("zero crtIndex returns ErrTransferIndexOverflow", func(t *testing.T) {
		t.Parallel()
		output := &VMOutput{}
		indexer := &stubTransferIndexer{crtIndex: 0}
		err := output.ReindexTransfers(indexer)
		require.Equal(t, ErrTransferIndexOverflow, err)
	})

	t.Run("nil output account returns ErrNilOutputAccount", func(t *testing.T) {
		t.Parallel()
		output := &VMOutput{
			OutputAccounts: map[string]*OutputAccount{
				"acc1": nil,
			},
		}
		indexer := &stubTransferIndexer{crtIndex: 1}
		err := output.ReindexTransfers(indexer)
		require.Equal(t, ErrNilOutputAccount, err)
	})

	t.Run("unindexed transfer returns ErrTransfersNotIndexed", func(t *testing.T) {
		t.Parallel()
		output := &VMOutput{
			OutputAccounts: map[string]*OutputAccount{
				"acc1": {
					OutputTransfers: []OutputTransfer{
						{Index: 0},
					},
				},
			},
		}
		indexer := &stubTransferIndexer{crtIndex: 2}
		err := output.ReindexTransfers(indexer)
		require.Equal(t, ErrTransfersNotIndexed, err)
	})

	t.Run("transfer index overflow returns ErrTransferIndexOverflow", func(t *testing.T) {
		t.Parallel()
		output := &VMOutput{
			OutputAccounts: map[string]*OutputAccount{
				"acc1": {
					OutputTransfers: []OutputTransfer{
						{Index: 10},
					},
				},
			},
		}
		// crtIndex is MaxUint32, so offset is MaxUint32 - 1, and 10 + offset overflows
		indexer := &stubTransferIndexer{crtIndex: math.MaxUint32}
		err := output.ReindexTransfers(indexer)
		require.Equal(t, ErrTransferIndexOverflow, err)
	})

	t.Run("successful reindexing without mutating prior accounts on failure", func(t *testing.T) {
		t.Parallel()
		output := &VMOutput{
			OutputAccounts: map[string]*OutputAccount{
				"acc1": {
					OutputTransfers: []OutputTransfer{
						{Index: 1, Value: big.NewInt(100)},
						{Index: 2, Value: big.NewInt(200)},
					},
				},
			},
		}
		indexer := &stubTransferIndexer{crtIndex: 5} // offset = 4
		err := output.ReindexTransfers(indexer)
		require.NoError(t, err)

		acc := output.OutputAccounts["acc1"]
		require.Equal(t, uint32(5), acc.OutputTransfers[0].Index)
		require.Equal(t, uint32(6), acc.OutputTransfers[1].Index)
		require.Equal(t, uint32(7), indexer.GetCrtTransferIndex())
	})
}
