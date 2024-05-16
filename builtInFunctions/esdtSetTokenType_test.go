package builtInFunctions

import (
	"math/big"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/require"
)

func TestNewESDTSetTokenTypeFunc(t *testing.T) {
	t.Parallel()

	t.Run("nil accounts adapter", func(t *testing.T) {
		t.Parallel()

		_, err := NewESDTSetTokenTypeFunc(nil, nil, nil, nil)
		require.Equal(t, ErrNilAccountsAdapter, err)
	})
	t.Run("nil marshaller", func(t *testing.T) {
		t.Parallel()

		_, err := NewESDTSetTokenTypeFunc(&mock.AccountsStub{}, nil, nil, nil)
		require.Equal(t, ErrNilMarshalizer, err)
	})
	t.Run("nil global settings handler", func(t *testing.T) {
		t.Parallel()

		_, err := NewESDTSetTokenTypeFunc(&mock.AccountsStub{}, nil, &mock.MarshalizerMock{}, nil)
		require.Equal(t, ErrNilGlobalSettingsHandler, err)
	})
	t.Run("nil active handler", func(t *testing.T) {
		t.Parallel()

		_, err := NewESDTSetTokenTypeFunc(&mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, &mock.MarshalizerMock{}, nil)
		require.Equal(t, ErrNilActiveHandler, err)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		_, err := NewESDTSetTokenTypeFunc(
			&mock.AccountsStub{},
			&mock.GlobalSettingsHandlerStub{},
			&mock.MarshalizerMock{},
			func() bool {
				return true
			},
		)
		require.Nil(t, err)
	})

}

func TestESDTSetTokenType_ProcessBuiltinFunction(t *testing.T) {
	t.Parallel()

	t.Run("nil vm input", func(t *testing.T) {
		t.Parallel()

		e := &esdtSetTokenType{}
		_, err := e.ProcessBuiltinFunction(nil, nil, nil)
		require.Equal(t, ErrNilVmInput, err)
	})
	t.Run("built-in function called with value", func(t *testing.T) {
		t.Parallel()

		e := &esdtSetTokenType{}
		_, err := e.ProcessBuiltinFunction(nil, nil, &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue: big.NewInt(10),
			},
		})
		require.Equal(t, ErrBuiltInFunctionCalledWithValue, err)
	})
	t.Run("invalid arguments", func(t *testing.T) {
		t.Parallel()

		e := &esdtSetTokenType{}
		_, err := e.ProcessBuiltinFunction(nil, nil, &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue: zero,
				Arguments: [][]byte{{}},
			},
		})
		require.Equal(t, ErrInvalidArguments, err)
	})
	t.Run("caller address is not ESDT system SC", func(t *testing.T) {
		t.Parallel()

		e := &esdtSetTokenType{}
		_, err := e.ProcessBuiltinFunction(nil, nil, &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:  zero,
				Arguments:  [][]byte{{}, {}},
				CallerAddr: []byte("random address"),
			},
		})
		require.Equal(t, ErrAddressIsNotESDTSystemSC, err)
	})
	t.Run("recipient addr is not system account", func(t *testing.T) {
		t.Parallel()

		e := &esdtSetTokenType{}
		_, err := e.ProcessBuiltinFunction(nil, nil, &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:  zero,
				Arguments:  [][]byte{{}, {}},
				CallerAddr: core.ESDTSCAddress,
			},
			RecipientAddr: []byte("random address"),
		})
		require.Equal(t, ErrOnlySystemAccountAccepted, err)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		tokenKey := []byte("tokenKey")
		tokenType := []byte(core.NonFungibleESDTv2)
		setTokenTypeCalled := false
		e := &esdtSetTokenType{
			globalSettingsHandler: &mock.GlobalSettingsHandlerStub{
				SetTokenTypeCalled: func(esdtTokenKey []byte, tokenType uint32, _ vmcommon.UserAccountHandler) error {
					require.Equal(t, append([]byte(baseESDTKeyPrefix), tokenKey...), esdtTokenKey)
					require.Equal(t, uint32(core.NonFungibleV2), tokenType)
					setTokenTypeCalled = true
					return nil
				},
			},
			accounts: createAccountsAdapterWithMap(),
		}

		_, err := e.ProcessBuiltinFunction(nil, nil, &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:  zero,
				Arguments:  [][]byte{tokenKey, tokenType},
				CallerAddr: core.ESDTSCAddress,
			},
			RecipientAddr: core.SystemAccountAddress,
		})
		require.Nil(t, err)
		require.True(t, setTokenTypeCalled)
	})
}
