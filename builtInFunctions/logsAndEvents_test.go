package builtInFunctions

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	"github.com/stretchr/testify/require"
)

func TestNewEntryForNFT(t *testing.T) {
	t.Parallel()

	vmOutput := &vmcommon.VMOutput{}
	addESDTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionESDTNFTCreate), []byte("my-token"), 5, big.NewInt(1), []byte("caller"), []byte("receiver"))
	require.Equal(t, &vmcommon.LogEntry{
		Identifier: []byte(core.BuiltInFunctionESDTNFTCreate),
		Address:    []byte("caller"),
		Topics:     [][]byte{[]byte("my-token"), big.NewInt(0).SetUint64(5).Bytes(), big.NewInt(1).Bytes(), []byte("receiver")},
		Data:       nil,
	}, vmOutput.Logs[0])
}

func TestExtractTokenIdentifierAndNonceESDTWipe(t *testing.T) {
	t.Parallel()

	hexArg := "534b4537592d37336262636404"
	args, _ := hex.DecodeString(hexArg)

	identifier, nonce := extractTokenIdentifierAndNonceESDTWipe(args)
	require.Equal(t, uint64(4), nonce)
	require.Equal(t, []byte("SKE7Y-73bbcd"), identifier)

	hexArg = "5745474c442d376662623930"
	args, _ = hex.DecodeString(hexArg)

	identifier, nonce = extractTokenIdentifierAndNonceESDTWipe(args)
	require.Equal(t, uint64(0), nonce)
	require.Equal(t, []byte("WEGLD-7fbb90"), identifier)

	// Nonce 45 (0x2d = '-')
	tokenID := []byte("TEST-123456")
	argWithNonce45 := append(tokenID, byte(0x2d))
	identifier, nonce = extractTokenIdentifierAndNonceESDTWipe(argWithNonce45)
	require.Equal(t, uint64(45), nonce)
	require.Equal(t, tokenID, identifier)

	// Nonce 301 (0x012d)
	argWithNonce301 := append(tokenID, 0x01, 0x2d)
	identifier, nonce = extractTokenIdentifierAndNonceESDTWipe(argWithNonce301)
	require.Equal(t, uint64(301), nonce)
	require.Equal(t, tokenID, identifier)

	// Nonce 11521 (0x2d01)
	argWithNonce11521 := append(tokenID, 0x2d, 0x01)
	identifier, nonce = extractTokenIdentifierAndNonceESDTWipe(argWithNonce11521)
	require.Equal(t, uint64(11521), nonce)
	require.Equal(t, tokenID, identifier)
}
