package datafield

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractTokenAndNonce(t *testing.T) {
	t.Parallel()

	hexArg := "534b4537592d37336262636404"
	args, _ := hex.DecodeString(hexArg)

	token, nonce := extractTokenAndNonce(args)
	require.Equal(t, uint64(4), nonce)
	require.Equal(t, "SKE7Y-73bbcd", token)
}

func TestExtractTokenAndNonce_HyphenInNonce(t *testing.T) {
	t.Parallel()

	tokenID := []byte("SKE7Y-73bbcd")
	arg45 := append(tokenID, 0x2d)
	token, nonce := extractTokenAndNonce(arg45)
	require.Equal(t, uint64(45), nonce)
	require.Equal(t, "SKE7Y-73bbcd", token)

	arg301 := append(tokenID, 0x01, 0x2d)
	token, nonce = extractTokenAndNonce(arg301)
	require.Equal(t, uint64(301), nonce)
	require.Equal(t, "SKE7Y-73bbcd", token)

	arg11521 := append(tokenID, 0x2d, 0x01)
	token, nonce = extractTokenAndNonce(arg11521)
	require.Equal(t, uint64(11521), nonce)
	require.Equal(t, "SKE7Y-73bbcd", token)
}

func TestComputeTokenIdentifier(t *testing.T) {
	t.Parallel()

	identifier := computeTokenIdentifier("MYTOKEN-abcd", 10)
	require.Equal(t, "MYTOKEN-abcd-0a", identifier)
}

func TestIsASCIIString(t *testing.T) {
	t.Parallel()

	require.True(t, isASCIIString("hello"))
	require.True(t, isASCIIString("TOKEN-abcd"))
	require.False(t, isASCIIString(string([]byte{12, 255})))
	require.False(t, isASCIIString(string([]byte{12, 188})))
}
