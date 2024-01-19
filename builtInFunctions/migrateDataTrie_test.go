package builtInFunctions

import (
	"bytes"
	"errors"
	"math/big"
	"strings"
	"sync"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core/check"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			IsMigrateDataTrieEnabledField: true,
		}
		mdtf, err := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, enableEpochs, &mock.AccountsStub{})
		assert.False(t, check.IfNil(mdtf))
		assert.Nil(t, err)
		assert.True(t, mdtf.IsActive())

		enableEpochs.IsMigrateDataTrieEnabledField = false
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
	t.Run("not enough gas provided for at least one migration", func(t *testing.T) {
		t.Parallel()

		input := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				GasProvided: 100,
			},
		}
		builtInCost := vmcommon.BuiltInCost{
			TrieLoadPerNode:  40,
			TrieStorePerNode: 61,
		}

		mdtf, _ := NewMigrateDataTrieFunc(builtInCost, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{})
		vmOutput, err := mdtf.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), mock.NewUserAccount([]byte("dest")), input)
		assert.Nil(t, vmOutput)
		assert.True(t, strings.Contains(err.Error(), "not enough gas"))
	})
	t.Run("invalid call value", func(t *testing.T) {
		t.Parallel()

		input := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue: big.NewInt(1),
			},
		}

		mdtf, _ := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{})
		vmOutput, err := mdtf.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), mock.NewUserAccount([]byte("dest")), input)
		assert.Nil(t, vmOutput)
		assert.Equal(t, ErrBuiltInFunctionCalledWithValue, err)
	})
	t.Run("invalid number of arguments", func(t *testing.T) {
		t.Parallel()

		input := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue: big.NewInt(0),
				Arguments: [][]byte{[]byte("arg1")},
			},
		}

		mdtf, _ := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{})
		vmOutput, err := mdtf.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), nil, input)
		assert.Nil(t, vmOutput)
		assert.True(t, errors.Is(err, ErrInvalidNumberOfArguments))
	})
	t.Run("nil dest account", func(t *testing.T) {
		t.Parallel()

		input := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:  big.NewInt(0),
				CallerAddr: []byte("caller"),
			},
			RecipientAddr: []byte("caller"),
		}

		mdtf, _ := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{})
		vmOutput, err := mdtf.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), nil, input)
		assert.Nil(t, vmOutput)
		assert.True(t, errors.Is(err, ErrNilSCDestAccount))
	})
	t.Run("address is system account address from shard 0", func(t *testing.T) {
		t.Parallel()

		migrateCalled := false
		saveCalled := false
		input := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:  big.NewInt(0),
				CallerAddr: []byte("12345678912345678912345678912345"),
			},
			RecipientAddr: []byte("12345678912345678912345678912345"),
		}
		systemAcc := &mock.UserAccountStub{
			AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
				return &mock.DataTrieTrackerStub{
					MigrateDataTrieLeavesCalled: func(args vmcommon.ArgsMigrateDataTrieLeaves) error {
						migrateCalled = true
						return nil
					},
				}
			},
		}
		adb := &mock.AccountsStub{
			GetExistingAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
				return systemAcc, nil
			},
			SaveAccountCalled: func(account vmcommon.AccountHandler) error {
				assert.Equal(t, systemAcc, account)
				saveCalled = true
				return nil
			},
		}

		mdtf, _ := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, &mock.EnableEpochsHandlerStub{}, adb)

		shard0SystemAccAddr := bytes.Repeat([]byte{255}, 30)
		shard0SystemAccAddr = append(shard0SystemAccAddr, []byte{0, 0}...)
		vmOutput, err := mdtf.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), mock.NewUserAccount(shard0SystemAccAddr), input)
		assert.Nil(t, err)
		assert.NotNil(t, vmOutput)
		assert.True(t, migrateCalled)
		assert.True(t, saveCalled)
	})
	t.Run("address is system account address from shard 1", func(t *testing.T) {
		t.Parallel()

		migrateCalled := false
		input := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:  big.NewInt(0),
				CallerAddr: []byte("12345678912345678912345678912345"),
			},
			RecipientAddr: []byte("12345678912345678912345678912345"),
		}
		systemAcc := &mock.UserAccountStub{
			AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
				return &mock.DataTrieTrackerStub{
					MigrateDataTrieLeavesCalled: func(args vmcommon.ArgsMigrateDataTrieLeaves) error {
						migrateCalled = true
						return nil
					},
				}
			},
		}
		adb := &mock.AccountsStub{
			GetExistingAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
				assert.Fail(t, "should not be called")
				return nil, nil
			},
			SaveAccountCalled: func(account vmcommon.AccountHandler) error {
				assert.Fail(t, "should not be called")
				return nil
			},
		}

		mdtf, _ := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, &mock.EnableEpochsHandlerStub{}, adb)

		vmOutput, err := mdtf.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), systemAcc, input)
		assert.Nil(t, err)
		assert.NotNil(t, vmOutput)
		assert.True(t, migrateCalled)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		migrateCalled := false
		input := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:  big.NewInt(0),
				CallerAddr: []byte("sender"),
			},
		}
		destAcc := &mock.UserAccountStub{
			AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
				return &mock.DataTrieTrackerStub{
					MigrateDataTrieLeavesCalled: func(args vmcommon.ArgsMigrateDataTrieLeaves) error {
						migrateCalled = true
						return nil
					},
				}
			},
		}

		mdtf, _ := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{})
		vmOutput, err := mdtf.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), destAcc, input)
		assert.Nil(t, err)
		assert.NotNil(t, vmOutput)
		assert.True(t, migrateCalled)
	})
}

func TestMigrateDataTrie_SetNewGasConfig(t *testing.T) {
	t.Parallel()

	mdtf, _ := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{TrieLoadPerNode: 50}, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{})
	mdtf.SetNewGasConfig(&vmcommon.GasCost{BuiltInCost: vmcommon.BuiltInCost{TrieLoadPerNode: 100}})
	assert.Equal(t, uint64(100), mdtf.builtInCost.TrieLoadPerNode)
}

func TestMigrateDataTrie_Concurrency(t *testing.T) {
	t.Parallel()

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			GasProvided: 10000,
			CallerAddr:  []byte("sender"),
		},
	}
	adb := &mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return mock.NewUserAccount(address), nil
		},
	}

	mdtf, _ := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, &mock.EnableEpochsHandlerStub{}, adb)
	numOperations := 1000

	wg := sync.WaitGroup{}
	wg.Add(numOperations)
	for i := 0; i < numOperations; i++ {
		go func(index int) {
			defer func() {
				wg.Done()
			}()

			switch index % 2 {
			case 0:
				_, err := mdtf.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), mock.NewUserAccount([]byte("dest")), input)
				require.Nil(t, err)
			case 1:
				newGasCost := &vmcommon.GasCost{
					BuiltInCost: vmcommon.BuiltInCost{
						TrieLoadPerNode: uint64(index),
					},
				}
				mdtf.SetNewGasConfig(newGasCost)
			}
		}(i)
	}

	wg.Wait()
}
