package vmcommon

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func EsdtLocalRoles_FromBytes(t *testing.T) {
	require.True(t, EsdtLocalRolesFromBytes([]byte{0, 0, 0, 0, 0, 0, 0, 1}).Mint)
	require.True(t, EsdtLocalRolesFromBytes([]byte{0, 0, 0, 0, 0, 0, 0, 2}).Burn)
	require.True(t, EsdtLocalRolesFromBytes([]byte{0, 0, 0, 0, 0, 0, 0, 4}).NFTCreate)
	require.True(t, EsdtLocalRolesFromBytes([]byte{0, 0, 0, 0, 0, 0, 1, 0}).NFTAddQuantity)
	require.True(t, EsdtLocalRolesFromBytes([]byte{0, 0, 0, 0, 0, 0, 2, 0}).NFTBurn)
	require.False(t, EsdtLocalRolesFromBytes([]byte{0, 0, 0, 0, 0, 1, 1, 0}).Mint)
	require.False(t, EsdtLocalRolesFromBytes([]byte{0, 0, 0, 0, 0, 1, 0, 0}).Burn)
	require.False(t, EsdtLocalRolesFromBytes([]byte{0, 0, 0, 0, 0, 1, 0, 0}).NFTCreate)
	require.False(t, EsdtLocalRolesFromBytes([]byte{0, 0, 0, 0, 0, 1, 0, 0}).NFTAddQuantity)
	require.False(t, EsdtLocalRolesFromBytes([]byte{0, 0, 0, 0, 0, 1, 0, 0}).NFTBurn)
}

func EsdtLocalRoles_ToBytes(t *testing.T) {
	require.Equal(t, byte(0), (&CodeMetadata{}).ToBytes()[0])
	require.Equal(t, byte(0), (&CodeMetadata{}).ToBytes()[1])
	require.Equal(t, byte(1), (&CodeMetadata{Upgradeable: true}).ToBytes()[0])
	require.Equal(t, byte(2), (&CodeMetadata{Payable: true}).ToBytes()[1])
	require.Equal(t, byte(4), (&CodeMetadata{Readable: true}).ToBytes()[0])
}
