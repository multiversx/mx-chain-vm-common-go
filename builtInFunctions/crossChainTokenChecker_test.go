package builtInFunctions

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewCrossChainTokenChecker(t *testing.T) {
	t.Parallel()

	t.Run("main chain, should work", func(t *testing.T) {
		ctc, err := NewCrossChainTokenChecker([]byte{})
		require.Nil(t, err)
		require.False(t, ctc.IsInterfaceNil())
	})

	t.Run("sovereign chain with invalid prefix, should not work", func(t *testing.T) {
		ctc, err := NewCrossChainTokenChecker([]byte("PREFIX"))
		require.ErrorIs(t, err, ErrInvalidTokenPrefix)
		require.Nil(t, ctc)
	})

	t.Run("sovereign chain valid prefix, should work", func(t *testing.T) {
		ctc, err := NewCrossChainTokenChecker([]byte("pref"))
		require.Nil(t, err)
		require.False(t, ctc.IsInterfaceNil())
	})
}

func TestCrossChainTokenChecker_IsCrossChainOperation(t *testing.T) {
	t.Parallel()

	t.Run("cross chain operations in a sovereign shard", func(t *testing.T) {
		ctc, _ := NewCrossChainTokenChecker([]byte("sov1"))

		require.True(t, ctc.IsCrossChainOperation([]byte("ALICE-abcdef")))
		require.True(t, ctc.IsCrossChainOperation([]byte("sov2-ALICE-abcdef")))
		require.False(t, ctc.IsCrossChainOperation([]byte("sov1-ALICE-abcdef")))
	})

	t.Run("cross chain operations in a main chain", func(t *testing.T) {
		ctc, _ := NewCrossChainTokenChecker(nil)

		require.True(t, ctc.IsCrossChainOperation([]byte("sov2-ALICE-abcdef")))
		require.True(t, ctc.IsCrossChainOperation([]byte("sov1-ALICE-abcdef")))
		require.False(t, ctc.IsCrossChainOperation([]byte("ALICE-abcdef")))
	})
}

func TestCrossChainTokenChecker_IsSelfMainChain(t *testing.T) {
	t.Parallel()

	ctcSov, _ := NewCrossChainTokenChecker([]byte("sov1"))
	require.False(t, ctcSov.IsSelfMainChain())

	ctcMain, _ := NewCrossChainTokenChecker(nil)
	require.True(t, ctcMain.IsSelfMainChain())
}
