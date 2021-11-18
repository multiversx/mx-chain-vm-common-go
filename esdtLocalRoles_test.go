package vmcommon

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEsdtLocalRoles_FromBytes(t *testing.T) {
	require.True(t, EsdtLocalRolesFromBytes([]byte{0, 0, 0, 0, 0, 0, 0, 1}).Mint)
	require.True(t, EsdtLocalRolesFromBytes([]byte{0, 0, 0, 0, 0, 0, 0, 2}).Burn)
	require.True(t, EsdtLocalRolesFromBytes([]byte{0, 0, 0, 0, 0, 0, 0, 4}).NFTCreate)
	require.True(t, EsdtLocalRolesFromBytes([]byte{0, 0, 0, 0, 0, 0, 0, 8}).NFTAddQuantity)
	require.True(t, EsdtLocalRolesFromBytes([]byte{0, 0, 0, 0, 0, 0, 0, 16}).NFTBurn)
	require.False(t, EsdtLocalRolesFromBytes([]byte{0, 0, 0, 0, 0, 1, 1, 0}).Mint)
	require.False(t, EsdtLocalRolesFromBytes([]byte{0, 0, 0, 0, 0, 1, 0, 0}).Burn)
	require.False(t, EsdtLocalRolesFromBytes([]byte{0, 0, 0, 0, 0, 1, 0, 0}).NFTCreate)
	require.False(t, EsdtLocalRolesFromBytes([]byte{0, 0, 0, 0, 0, 1, 0, 0}).NFTAddQuantity)
	require.False(t, EsdtLocalRolesFromBytes([]byte{0, 0, 0, 0, 0, 1, 0, 0}).NFTBurn)
}

func TestEsdtLocalRoles_ToBytes(t *testing.T) {
	require.Equal(t, byte(0), (&EsdtLocalRoles{}).ToBytes()[0])
	require.Equal(t, byte(0), (&EsdtLocalRoles{}).ToBytes()[1])
	require.Equal(t, byte(1), (&EsdtLocalRoles{Mint: true}).ToBytes()[7])
	require.Equal(t, byte(2), (&EsdtLocalRoles{Burn: true}).ToBytes()[7])
	require.Equal(t, byte(4), (&EsdtLocalRoles{NFTCreate: true}).ToBytes()[7])
}
