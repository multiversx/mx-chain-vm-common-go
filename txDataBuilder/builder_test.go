package txDataBuilder

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTxDataBuilder_ToString(t *testing.T) {
	t.Parallel()

	t.Run("EmptyBuilder", func(t *testing.T) {
		t.Parallel()

		builder := NewBuilder()
		require.Equal(t, "", builder.ToString())
		require.Equal(t, []byte(""), builder.ToBytes())
	})

	t.Run("FunctionOnly", func(t *testing.T) {
		t.Parallel()

		builder := NewBuilder().Func("testFunc")
		require.Equal(t, "testFunc", builder.ToString())
		require.Equal(t, []byte("testFunc"), builder.ToBytes())
	})

	t.Run("FunctionWithElements", func(t *testing.T) {
		t.Parallel()

		builder := NewBuilder().
			Func("myFunc").
			Str("arg1").
			Byte(0x0a).
			Int(42).
			True()

		expected := "myFunc@61726731@0a@2a@74727565"
		require.Equal(t, expected, builder.ToString())
		require.Equal(t, []byte(expected), builder.ToBytes())
	})

	t.Run("ClearAndReuse", func(t *testing.T) {
		t.Parallel()

		builder := NewBuilder().Func("first").Str("elem")
		require.Equal(t, "first@656c656d", builder.ToString())

		builder.Clear().Func("second").Int64(100)
		require.Equal(t, "second@64", builder.ToString())
	})

	t.Run("GetAndSetLast", func(t *testing.T) {
		t.Parallel()

		builder := NewBuilder()
		require.Equal(t, "", builder.GetLast())

		builder.Str("elem1").Str("elem2")
		require.Equal(t, "656c656d32", builder.GetLast())

		builder.SetLast("replaced")
		require.Equal(t, "replaced", builder.GetLast())
		require.Equal(t, "@656c656d31@replaced", builder.ToString())
	})

	t.Run("BuiltinHelperMethods", func(t *testing.T) {
		t.Parallel()

		builder := NewBuilder().
			IssueESDT("MYTOK", "TICKER", 1000, 18).
			CanFreeze(true).
			CanWipe(false).
			BigInt(big.NewInt(500))

		res := builder.ToString()
		require.NotEmpty(t, res)
		require.False(t, builder.IsInterfaceNil())
	})
}

func BenchmarkTxDataBuilder_ToString(b *testing.B) {
	builder := NewBuilder().Func("multiTransferESDTNFT")
	for i := 0; i < 100; i++ {
		builder.Str("token-identifier-123456").Int(i).Int64(int64(i * 1000))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = builder.ToString()
	}
}
