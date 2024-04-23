package builtInFunctions

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
