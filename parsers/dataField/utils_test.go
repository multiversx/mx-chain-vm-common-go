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
