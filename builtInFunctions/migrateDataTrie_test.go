package builtInFunctions

import (
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

		mdtf, err := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, nil, &mock.AccountsStub{}, &mock.ShardCoordinatorStub{})
		assert.True(t, check.IfNil(mdtf))
		assert.Equal(t, ErrNilEnableEpochsHandler, err)
	})
	t.Run("nil accountsDB", func(t *testing.T) {
		t.Parallel()

		mdtf, err := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, &mock.EnableEpochsHandlerStub{}, nil, &mock.ShardCoordinatorStub{})
		assert.True(t, check.IfNil(mdtf))
		assert.Equal(t, ErrNilAccountsAdapter, err)
	})
	t.Run("nil shardCoordinator", func(t *testing.T) {
		t.Parallel()

		mdtf, err := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{}, nil)
		assert.True(t, check.IfNil(mdtf))
		assert.Equal(t, ErrNilShardCoordinator, err)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		enableEpochs := &mock.EnableEpochsHandlerStub{
			IsMigrateDataTrieEnabledField: true,
		}
		mdtf, err := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, enableEpochs, &mock.AccountsStub{}, &mock.ShardCoordinatorStub{})
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

		mdtf, _ := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{}, &mock.ShardCoordinatorStub{})
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

		mdtf, _ := NewMigrateDataTrieFunc(builtInCost, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{}, &mock.ShardCoordinatorStub{})
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

		mdtf, _ := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{}, &mock.ShardCoordinatorStub{})
		vmOutput, err := mdtf.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), mock.NewUserAccount([]byte("dest")), input)
		assert.Nil(t, vmOutput)
		assert.Equal(t, ErrBuiltInFunctionCalledWithValue, err)
	})
	t.Run("invalid number of arguments", func(t *testing.T) {
		t.Parallel()

		input := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue: big.NewInt(0),
				Arguments: [][]byte{[]byte("arg1"), []byte("arg2")},
			},
		}

		mdtf, _ := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{}, &mock.ShardCoordinatorStub{})
		vmOutput, err := mdtf.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), nil, input)
		assert.Nil(t, vmOutput)
		assert.True(t, errors.Is(err, ErrInvalidNumberOfArguments))

		input.Arguments = [][]byte{}
		vmOutput, err = mdtf.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), nil, input)
		assert.Nil(t, vmOutput)
		assert.True(t, errors.Is(err, ErrInvalidNumberOfArguments))
	})
	t.Run("tx is not to self", func(t *testing.T) {
		t.Parallel()

		input := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:  big.NewInt(0),
				Arguments:  [][]byte{[]byte("arg1")},
				CallerAddr: []byte("caller"),
			},
			RecipientAddr: []byte("recipient"),
		}

		mdtf, _ := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{}, &mock.ShardCoordinatorStub{})
		vmOutput, err := mdtf.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), nil, input)
		assert.Nil(t, vmOutput)
		assert.Equal(t, ErrOperationNotPermitted, err)
	})
	t.Run("address is empty address", func(t *testing.T) {
		t.Parallel()

		input := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:  big.NewInt(0),
				Arguments:  [][]byte{make([]byte, 6)},
				CallerAddr: []byte("caller"),
			},
			RecipientAddr: []byte("caller"),
		}

		mdtf, _ := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{}, &mock.ShardCoordinatorStub{})
		vmOutput, err := mdtf.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), nil, input)
		assert.Nil(t, vmOutput)
		assert.True(t, errors.Is(err, ErrInvalidAddress))
	})
	t.Run("address length is not equal to caller address length", func(t *testing.T) {
		t.Parallel()

		input := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:  big.NewInt(0),
				Arguments:  [][]byte{[]byte("arg")},
				CallerAddr: []byte("caller"),
			},
			RecipientAddr: []byte("caller"),
		}

		mdtf, _ := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{}, &mock.ShardCoordinatorStub{})
		vmOutput, err := mdtf.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), nil, input)
		assert.Nil(t, vmOutput)
		assert.True(t, errors.Is(err, ErrInvalidAddress))
	})
	t.Run("address is not in same shard as the caller", func(t *testing.T) {
		t.Parallel()

		input := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:  big.NewInt(0),
				Arguments:  [][]byte{[]byte("arg123")},
				CallerAddr: []byte("caller"),
			},
			RecipientAddr: []byte("caller"),
		}
		shardCoordinator := &mock.ShardCoordinatorStub{
			SameShardCalled: func(firstAddress, secondAddress []byte) bool {
				assert.Equal(t, firstAddress, input.CallerAddr)
				assert.Equal(t, secondAddress, input.Arguments[0])
				return false
			},
		}

		mdtf, _ := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{}, shardCoordinator)
		vmOutput, err := mdtf.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), nil, input)
		assert.Nil(t, vmOutput)
		assert.True(t, errors.Is(err, ErrOperationNotPermitted))
	})
	t.Run("address is system account address", func(t *testing.T) {
		t.Parallel()

		migrateCalled := false
		input := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:  big.NewInt(0),
				Arguments:  [][]byte{vmcommon.SystemAccountAddress},
				CallerAddr: []byte("12345678912345678912345678912345"),
			},
			RecipientAddr: []byte("12345678912345678912345678912345"),
		}
		adb := &mock.AccountsStub{
			LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
				return &mock.UserAccountStub{
					AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
						return &mock.DataTrieTrackerStub{
							MigrateDataTrieLeavesCalled: func(args vmcommon.ArgsMigrateDataTrieLeaves) error {
								assert.Equal(t, address, vmcommon.SystemAccountAddress)
								migrateCalled = true
								return nil
							},
						}
					},
				}, nil
			},
		}

		mdtf, _ := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, &mock.EnableEpochsHandlerStub{}, adb, &mock.ShardCoordinatorStub{})
		vmOutput, err := mdtf.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), nil, input)
		assert.Nil(t, err)
		assert.NotNil(t, vmOutput)
		assert.True(t, migrateCalled)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		dataTrieAddress := []byte("dataTrieAddress")
		migrateCalled := false
		input := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:  big.NewInt(0),
				Arguments:  [][]byte{dataTrieAddress},
				CallerAddr: []byte("123456789123456"),
			},
			RecipientAddr: []byte("123456789123456"),
		}
		adb := &mock.AccountsStub{
			LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
				return &mock.UserAccountStub{
					AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
						return &mock.DataTrieTrackerStub{
							MigrateDataTrieLeavesCalled: func(args vmcommon.ArgsMigrateDataTrieLeaves) error {
								assert.Equal(t, address, dataTrieAddress)
								migrateCalled = true
								return nil
							},
						}
					},
				}, nil
			},
		}
		shardCoordinator := &mock.ShardCoordinatorStub{
			SameShardCalled: func(firstAddress, secondAddress []byte) bool {
				return true
			},
		}

		mdtf, _ := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, &mock.EnableEpochsHandlerStub{}, adb, shardCoordinator)
		vmOutput, err := mdtf.ProcessBuiltinFunction(mock.NewUserAccount([]byte("sender")), nil, input)
		assert.Nil(t, err)
		assert.NotNil(t, vmOutput)
		assert.True(t, migrateCalled)
	})
}

func TestMigrateDataTrie_SetNewGasConfig(t *testing.T) {
	t.Parallel()

	mdtf, _ := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{TrieLoadPerNode: 50}, &mock.EnableEpochsHandlerStub{}, &mock.AccountsStub{}, &mock.ShardCoordinatorStub{})
	mdtf.SetNewGasConfig(&vmcommon.GasCost{BuiltInCost: vmcommon.BuiltInCost{TrieLoadPerNode: 100}})
	assert.Equal(t, uint64(100), mdtf.builtInCost.TrieLoadPerNode)
}

func TestMigrateDataTrie_Concurrency(t *testing.T) {
	t.Parallel()

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			GasProvided: 10000,
			Arguments:   [][]byte{[]byte("arg123")},
			CallerAddr:  []byte("caller"),
		},
		RecipientAddr: []byte("caller"),
	}
	shardCoordinator := &mock.ShardCoordinatorStub{
		SameShardCalled: func(firstAddress, secondAddress []byte) bool {
			return true
		},
	}
	adb := &mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return mock.NewUserAccount(address), nil
		},
	}

	mdtf, _ := NewMigrateDataTrieFunc(vmcommon.BuiltInCost{}, &mock.EnableEpochsHandlerStub{}, adb, shardCoordinator)
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
