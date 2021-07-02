package builtInFunctions

import (
	"math/big"
	"testing"

	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/stretchr/testify/require"
)

func TestNewEntryForNFT(t *testing.T) {
	t.Parallel()

	entry := newEntryForNFT(vmcommon.BuiltInFunctionESDTNFTCreate, []byte("caller"), []byte("my-token"), 5)
	require.Equal(t, &vmcommon.LogEntry{
		Identifier: []byte(vmcommon.BuiltInFunctionESDTNFTCreate),
		Address:    []byte("caller"),
		Topics:     [][]byte{[]byte("my-token"), big.NewInt(0).SetUint64(5).Bytes()},
		Data:       nil,
	}, entry)
}
