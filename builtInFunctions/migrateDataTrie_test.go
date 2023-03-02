package builtInFunctions

import (
	"math/big"
	"strings"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core/check"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/assert"
)

func TestNewMigrateDataTrieFunc(t *testing.T) {
	t.Parallel()

	t.Run("nil enableEpochsHandler", func(t *testing.T) {
		t.Parallel()

		mdtf, err := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, nil, &mock.AccountsStub{})
		assert.True(t, check.IfNil(mdtf))
		assert.Equal(t, ErrNilEnableEpochsHandler, err)
	})

	t.Run("nil accountsDB", func(t *testing.T) {
		t.Parallel()

		mdtf, err := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, &mock.EnableEpochsHandlerStub{}, nil)
		assert.True(t, check.IfNil(mdtf))
		assert.Equal(t, ErrNilAccountsAdapter, err)
	})

	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		enableEpochs := &mock.EnableEpochsHandlerStub{
			IsAutoBalanceDataTriesEnabledField: true,
		}
		mdtf, err := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, enableEpochs, &mock.AccountsStub{})
		assert.False(t, check.IfNil(mdtf))
		assert.Nil(t, err)
		assert.True(t, mdtf.IsActive())

		enableEpochs.IsAutoBalanceDataTriesEnabledField = false
		assert.False(t, mdtf.IsActive())
	})
}

func TestMigrateDataTrie_ProcessBuiltinFunction(t *testing.T) {
	t.Parallel()

	t.Run("nil vm input", func(t *testing.T) {
		t.Parallel()

		mdtf, _ := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{})
		vmOutput, err := mdtf.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), mock.NewUserAccount([]byte("dest")), nil)
		assert.Nil(t, vmOutput)
		assert.Equal(t, ErrNilVmInput, err)
	})

	t.Run("invalid arguments", func(t *testing.T) {
		t.Parallel()

		input1 := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				Arguments: [][]byte{[]byte("arg1")},
			},
		}
		input2 := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				Arguments: [][]byte{[]byte("arg1"), []byte("arg2"), []byte("arg3")},
			},
		}

		mdtf, _ := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{})
		vmOutput, err := mdtf.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), mock.NewUserAccount([]byte("dest")), input1)
		assert.Nil(t, vmOutput)
		assert.Equal(t, ErrInvalidArguments, err)

		vmOutput, err = mdtf.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), mock.NewUserAccount([]byte("dest")), input2)
		assert.Nil(t, vmOutput)
		assert.Equal(t, ErrInvalidArguments, err)
	})

	t.Run("invalid old version", func(t *testing.T) {
		t.Parallel()

		input := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				Arguments: [][]byte{[]byte("arg1"), {2}},
			},
		}

		mdtf, _ := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{})
		vmOutput, err := mdtf.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), mock.NewUserAccount([]byte("dest")), input)
		assert.Nil(t, vmOutput)
		assert.True(t, strings.Contains(err.Error(), ErrInvalidArguments.Error()))
		assert.True(t, strings.Contains(err.Error(), "old version"))
	})

	t.Run("invalid new version", func(t *testing.T) {
		t.Parallel()

		input := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				Arguments: [][]byte{{1}, []byte("arg2")},
			},
		}

		mdtf, _ := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{})
		vmOutput, err := mdtf.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), mock.NewUserAccount([]byte("dest")), input)
		assert.Nil(t, vmOutput)
		assert.True(t, strings.Contains(err.Error(), ErrInvalidArguments.Error()))
		assert.True(t, strings.Contains(err.Error(), "new version"))
	})

	t.Run("invalid call value", func(t *testing.T) {
		t.Parallel()

		input := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				Arguments: [][]byte{{1}, {2}},
				CallValue: big.NewInt(1),
			},
		}

		mdtf, _ := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{})
		vmOutput, err := mdtf.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), mock.NewUserAccount([]byte("dest")), input)
		assert.Nil(t, vmOutput)
		assert.Equal(t, ErrBuiltInFunctionCalledWithValue, err)
	})

	t.Run("nil dest account", func(t *testing.T) {
		t.Parallel()

		input := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				Arguments: [][]byte{{1}, {2}},
				CallValue: big.NewInt(0),
			},
		}

		mdtf, _ := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{})
		vmOutput, err := mdtf.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), nil, input)
		assert.Nil(t, vmOutput)
		assert.Equal(t, ErrNilSCDestAccount, err)
	})

	t.Run("dataTrieMigrator err", func(t *testing.T) {
		t.Parallel()

		input := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				Arguments:   [][]byte{{1}, {2}},
				CallValue:   big.NewInt(0),
				GasProvided: 100,
			},
		}

		mdtf, _ := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{TrieLoadPerNode: 110}, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{})
		vmOutput, err := mdtf.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), mock.NewUserAccount([]byte("dest")), input)
		assert.Nil(t, vmOutput)
		assert.True(t, strings.Contains(err.Error(), "not enough gas"))
	})

	t.Run("saves dest account", func(t *testing.T) {
		t.Parallel()

		gasProvided := uint64(100)
		input := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				Arguments:   [][]byte{{1}, {2}},
				CallValue:   big.NewInt(0),
				GasProvided: gasProvided,
			},
		}
		saveAccountsCalled := false
		destAddr := []byte("dest")
		accStub := &mock.AccountsStub{
			SaveAccountCalled: func(account vmcommon.AccountHandler) error {
				assert.Equal(t, destAddr, account.AddressBytes())
				saveAccountsCalled = true
				return nil
			},
		}

		mdtf, _ := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{TrieLoadPerNode: 50}, &mock.EnableEpochsHandlerStub{}, accStub)
		vmOutput, err := mdtf.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), mock.NewUserAccount(destAddr), input)
		assert.Nil(t, err)
		assert.True(t, saveAccountsCalled)
		assert.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)
		assert.Equal(t, gasProvided, vmOutput.GasRemaining)
	})
}

func TestMigrateDataTrie_SetNewGasConfig(t *testing.T) {
	t.Parallel()

	mdtf, _ := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{TrieLoadPerNode: 50}, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{})
	mdtf.SetNewGasConfig(&vmcommon.GasCost{BuiltInCost: vmcommon.BuiltInCost{TrieLoadPerNode: 100}})
	assert.Equal(t, uint64(100), mdtf.builtInCost.TrieLoadPerNode)
}
