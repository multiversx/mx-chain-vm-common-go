package builtInFunctions

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/data/esdt"
	"github.com/ElrondNetwork/elrond-go-core/data/vm"
	"github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/mock"
	"github.com/stretchr/testify/assert"
)

func TestESDTTransfer_ProcessBuiltInFunctionErrors(t *testing.T) {
	t.Parallel()

	shardC := &mock.ShardCoordinatorStub{}
	transferFunc, _ := NewESDTTransferFunc(
		10,
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		shardC,
		&mock.ESDTRoleHandlerStub{},
		1000,
		0,
		0,
		&mock.EpochNotifierStub{},
	)
	_ = transferFunc.SetPayableHandler(&mock.PayableHandlerStub{})
	_, err := transferFunc.ProcessBuiltinFunction(nil, nil, nil)
	assert.Equal(t, err, ErrNilVmInput)

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
		},
	}
	_, err = transferFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrInvalidArguments)

	input = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
		},
	}
	key := []byte("key")
	value := []byte("value")
	input.Arguments = [][]byte{key, value}
	_, err = transferFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Nil(t, err)

	input.GasProvided = transferFunc.funcGasCost - 1
	accSnd := mock.NewUserAccount([]byte("address"))
	_, err = transferFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrNotEnoughGas)

	input.GasProvided = transferFunc.funcGasCost
	input.RecipientAddr = core.ESDTSCAddress
	shardC.ComputeIdCalled = func(address []byte) uint32 {
		return core.MetachainShardId
	}
	_, err = transferFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrInvalidRcvAddr)
}

func TestESDTTransfer_ProcessBuiltInFunctionSingleShard(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.MarshalizerMock{}
	esdtRoleHandler := &mock.ESDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			assert.Equal(t, core.ESDTRoleTransfer, string(action))
			return nil
		},
	}
	transferFunc, _ := NewESDTTransferFunc(
		10,
		marshalizer,
		&mock.GlobalSettingsHandlerStub{},
		&mock.ShardCoordinatorStub{},
		esdtRoleHandler,
		1000,
		0,
		0,
		&mock.EpochNotifierStub{},
	)
	_ = transferFunc.SetPayableHandler(&mock.PayableHandlerStub{})

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
		},
	}
	key := []byte("key")
	value := big.NewInt(10).Bytes()
	input.Arguments = [][]byte{key, value}
	accSnd := mock.NewUserAccount([]byte("snd"))
	accDst := mock.NewUserAccount([]byte("dst"))

	_, err := transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Equal(t, err, ErrInsufficientFunds)

	esdtKey := append(transferFunc.keyPrefix, key...)
	esdtToken := &esdt.ESDigitalToken{Value: big.NewInt(100)}
	marshaledData, _ := marshalizer.Marshal(esdtToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(esdtKey, marshaledData)

	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Nil(t, err)
	marshaledData, _ = accSnd.AccountDataHandler().RetrieveValue(esdtKey)
	_ = marshalizer.Unmarshal(esdtToken, marshaledData)
	assert.True(t, esdtToken.Value.Cmp(big.NewInt(90)) == 0)

	marshaledData, _ = accDst.AccountDataHandler().RetrieveValue(esdtKey)
	_ = marshalizer.Unmarshal(esdtToken, marshaledData)
	assert.True(t, esdtToken.Value.Cmp(big.NewInt(10)) == 0)
}

func TestESDTTransfer_ProcessBuiltInFunctionSenderInShard(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.MarshalizerMock{}
	transferFunc, _ := NewESDTTransferFunc(
		10,
		marshalizer,
		&mock.GlobalSettingsHandlerStub{},
		&mock.ShardCoordinatorStub{},
		&mock.ESDTRoleHandlerStub{},
		1000,
		0,
		0,
		&mock.EpochNotifierStub{},
	)
	_ = transferFunc.SetPayableHandler(&mock.PayableHandlerStub{})

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
		},
	}
	key := []byte("key")
	value := big.NewInt(10).Bytes()
	input.Arguments = [][]byte{key, value}
	accSnd := mock.NewUserAccount([]byte("snd"))

	esdtKey := append(transferFunc.keyPrefix, key...)
	esdtToken := &esdt.ESDigitalToken{Value: big.NewInt(100)}
	marshaledData, _ := marshalizer.Marshal(esdtToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(esdtKey, marshaledData)

	_, err := transferFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Nil(t, err)
	marshaledData, _ = accSnd.AccountDataHandler().RetrieveValue(esdtKey)
	_ = marshalizer.Unmarshal(esdtToken, marshaledData)
	assert.True(t, esdtToken.Value.Cmp(big.NewInt(90)) == 0)
}

func TestESDTTransfer_ProcessBuiltInFunctionDestInShard(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.MarshalizerMock{}
	transferFunc, _ := NewESDTTransferFunc(
		10,
		marshalizer,
		&mock.GlobalSettingsHandlerStub{},
		&mock.ShardCoordinatorStub{},
		&mock.ESDTRoleHandlerStub{},
		1000,
		0,
		0,
		&mock.EpochNotifierStub{},
	)
	_ = transferFunc.SetPayableHandler(&mock.PayableHandlerStub{})

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
		},
	}
	key := []byte("key")
	value := big.NewInt(10).Bytes()
	input.Arguments = [][]byte{key, value}
	accDst := mock.NewUserAccount([]byte("dst"))

	vmOutput, err := transferFunc.ProcessBuiltinFunction(nil, accDst, input)
	assert.Nil(t, err)
	esdtKey := append(transferFunc.keyPrefix, key...)
	esdtToken := &esdt.ESDigitalToken{}
	marshaledData, _ := accDst.AccountDataHandler().RetrieveValue(esdtKey)
	_ = marshalizer.Unmarshal(esdtToken, marshaledData)
	assert.True(t, esdtToken.Value.Cmp(big.NewInt(10)) == 0)
	assert.Equal(t, uint64(0), vmOutput.GasRemaining)
}

func TestESDTTransfer_SndDstFrozen(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.MarshalizerMock{}
	accountStub := &mock.AccountsStub{}
	esdtGlobalSettingsFunc, _ := NewESDTGlobalSettingsFunc(accountStub, true, core.BuiltInFunctionESDTPause, 0, &mock.EpochNotifierStub{})
	transferFunc, _ := NewESDTTransferFunc(
		10,
		marshalizer,
		esdtGlobalSettingsFunc,
		&mock.ShardCoordinatorStub{},
		&mock.ESDTRoleHandlerStub{},
		1000,
		0,
		0,
		&mock.EpochNotifierStub{},
	)
	_ = transferFunc.SetPayableHandler(&mock.PayableHandlerStub{})

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
		},
	}
	key := []byte("key")
	value := big.NewInt(10).Bytes()
	input.Arguments = [][]byte{key, value}
	accSnd := mock.NewUserAccount([]byte("snd"))
	accDst := mock.NewUserAccount([]byte("dst"))

	esdtFrozen := ESDTUserMetadata{Frozen: true}
	esdtNotFrozen := ESDTUserMetadata{Frozen: false}

	esdtKey := append(transferFunc.keyPrefix, key...)
	esdtToken := &esdt.ESDigitalToken{Value: big.NewInt(100), Properties: esdtFrozen.ToBytes()}
	marshaledData, _ := marshalizer.Marshal(esdtToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(esdtKey, marshaledData)

	_, err := transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Equal(t, err, ErrESDTIsFrozenForAccount)

	esdtToken = &esdt.ESDigitalToken{Value: big.NewInt(100), Properties: esdtNotFrozen.ToBytes()}
	marshaledData, _ = marshalizer.Marshal(esdtToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(esdtKey, marshaledData)

	esdtToken = &esdt.ESDigitalToken{Value: big.NewInt(100), Properties: esdtFrozen.ToBytes()}
	marshaledData, _ = marshalizer.Marshal(esdtToken)
	_ = accDst.AccountDataHandler().SaveKeyValue(esdtKey, marshaledData)

	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Equal(t, err, ErrESDTIsFrozenForAccount)

	marshaledData, _ = accDst.AccountDataHandler().RetrieveValue(esdtKey)
	_ = marshalizer.Unmarshal(esdtToken, marshaledData)
	assert.True(t, esdtToken.Value.Cmp(big.NewInt(100)) == 0)

	input.ReturnCallAfterError = true
	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Nil(t, err)

	esdtToken = &esdt.ESDigitalToken{Value: big.NewInt(100), Properties: esdtNotFrozen.ToBytes()}
	marshaledData, _ = marshalizer.Marshal(esdtToken)
	_ = accDst.AccountDataHandler().SaveKeyValue(esdtKey, marshaledData)

	systemAccount := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	esdtGlobal := ESDTGlobalMetadata{Paused: true}
	pauseKey := []byte(core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier + string(key))
	_ = systemAccount.AccountDataHandler().SaveKeyValue(pauseKey, esdtGlobal.ToBytes())

	accountStub.LoadAccountCalled = func(address []byte) (vmcommon.AccountHandler, error) {
		if bytes.Equal(address, vmcommon.SystemAccountAddress) {
			return systemAccount, nil
		}
		return accDst, nil
	}

	input.ReturnCallAfterError = false
	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Equal(t, err, ErrESDTTokenIsPaused)

	input.ReturnCallAfterError = true
	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Nil(t, err)
}

func TestESDTTransfer_SndDstWithLimitedTransfer(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.MarshalizerMock{}
	accountStub := &mock.AccountsStub{}
	rolesHandler := &mock.ESDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			if bytes.Equal(action, []byte(core.ESDTRoleTransfer)) {
				return ErrActionNotAllowed
			}
			return nil
		},
	}
	esdtGlobalSettingsFunc, _ := NewESDTGlobalSettingsFunc(accountStub, true, core.BuiltInFunctionESDTSetLimitedTransfer, 0, &mock.EpochNotifierStub{})
	transferFunc, _ := NewESDTTransferFunc(
		10,
		marshalizer,
		esdtGlobalSettingsFunc,
		&mock.ShardCoordinatorStub{},
		rolesHandler,
		1000,
		0,
		0,
		&mock.EpochNotifierStub{},
	)
	_ = transferFunc.SetPayableHandler(&mock.PayableHandlerStub{})

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
		},
	}
	key := []byte("key")
	value := big.NewInt(10).Bytes()
	input.Arguments = [][]byte{key, value}
	accSnd := mock.NewUserAccount([]byte("snd"))
	accDst := mock.NewUserAccount([]byte("dst"))

	esdtKey := append(transferFunc.keyPrefix, key...)
	esdtToken := &esdt.ESDigitalToken{Value: big.NewInt(100)}
	marshaledData, _ := marshalizer.Marshal(esdtToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(esdtKey, marshaledData)

	esdtToken = &esdt.ESDigitalToken{Value: big.NewInt(100)}
	marshaledData, _ = marshalizer.Marshal(esdtToken)
	_ = accDst.AccountDataHandler().SaveKeyValue(esdtKey, marshaledData)

	systemAccount := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	esdtGlobal := ESDTGlobalMetadata{LimitedTransfer: true}
	pauseKey := []byte(core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier + string(key))
	_ = systemAccount.AccountDataHandler().SaveKeyValue(pauseKey, esdtGlobal.ToBytes())

	accountStub.LoadAccountCalled = func(address []byte) (vmcommon.AccountHandler, error) {
		if bytes.Equal(address, vmcommon.SystemAccountAddress) {
			return systemAccount, nil
		}
		return accDst, nil
	}

	_, err := transferFunc.ProcessBuiltinFunction(nil, accDst, input)
	assert.Nil(t, err)

	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Equal(t, err, ErrActionNotAllowed)

	_, err = transferFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrActionNotAllowed)

	input.ReturnCallAfterError = true
	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Nil(t, err)

	input.ReturnCallAfterError = false
	rolesHandler.CheckAllowedToExecuteCalled = func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
		if bytes.Equal(account.AddressBytes(), accSnd.Address) && bytes.Equal(tokenID, key) {
			return nil
		}
		return ErrActionNotAllowed
	}

	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Nil(t, err)

	rolesHandler.CheckAllowedToExecuteCalled = func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
		if bytes.Equal(account.AddressBytes(), accDst.Address) && bytes.Equal(tokenID, key) {
			return nil
		}
		return ErrActionNotAllowed
	}

	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Nil(t, err)
}

func TestESDTTransfer_ProcessBuiltInFunctionOnAsyncCallBack(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.MarshalizerMock{}
	transferFunc, _ := NewESDTTransferFunc(
		10,
		marshalizer,
		&mock.GlobalSettingsHandlerStub{},
		&mock.ShardCoordinatorStub{},
		&mock.ESDTRoleHandlerStub{},
		1000,
		0,
		0,
		&mock.EpochNotifierStub{},
	)
	_ = transferFunc.SetPayableHandler(&mock.PayableHandlerStub{})

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
			CallType:    vm.AsynchronousCallBack,
		},
	}
	key := []byte("key")
	value := big.NewInt(10).Bytes()
	input.Arguments = [][]byte{key, value}
	accSnd := mock.NewUserAccount([]byte("snd"))
	accDst := mock.NewUserAccount(core.ESDTSCAddress)

	esdtKey := append(transferFunc.keyPrefix, key...)
	esdtToken := &esdt.ESDigitalToken{Value: big.NewInt(100)}
	marshaledData, _ := marshalizer.Marshal(esdtToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(esdtKey, marshaledData)

	vmOutput, err := transferFunc.ProcessBuiltinFunction(nil, accDst, input)
	assert.Nil(t, err)

	marshaledData, _ = accDst.AccountDataHandler().RetrieveValue(esdtKey)
	_ = marshalizer.Unmarshal(esdtToken, marshaledData)
	assert.True(t, esdtToken.Value.Cmp(big.NewInt(10)) == 0)

	assert.Equal(t, vmOutput.GasRemaining, input.GasProvided)

	vmOutput, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Nil(t, err)
	vmOutput.GasRemaining = input.GasProvided - transferFunc.funcGasCost

	marshaledData, _ = accSnd.AccountDataHandler().RetrieveValue(esdtKey)
	_ = marshalizer.Unmarshal(esdtToken, marshaledData)
	assert.True(t, esdtToken.Value.Cmp(big.NewInt(90)) == 0)
}

func TestDetermineIsSCCallAfter(t *testing.T) {
	t.Parallel()

	scAddress, _ := hex.DecodeString("00000000000000000500e9a061848044cc9c6ac2d78dca9e4f72e72a0a5b315c")
	address, _ := hex.DecodeString("432d6fed4f1d8ac43cd3201fd047b98e27fc9c06efb20c6593ba577cd11228ab")
	minLenArguments := 4
	t.Run("less number of arguments should return false", func(t *testing.T) {
		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				Arguments: make([][]byte, 0),
			},
		}

		for i := 0; i < minLenArguments; i++ {
			assert.False(t, determineIsSCCallAfter(vmInput, scAddress, minLenArguments, false))
			assert.False(t, determineIsSCCallAfter(vmInput, scAddress, minLenArguments, true))
		}
	})
	t.Run("ReturnCallAfterError should return false", func(t *testing.T) {
		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				Arguments:            [][]byte{[]byte("arg1"), []byte("arg2"), []byte("arg3"), []byte("arg4"), []byte("arg5")},
				CallType:             vm.AsynchronousCall,
				ReturnCallAfterError: true,
			},
		}

		assert.False(t, determineIsSCCallAfter(vmInput, address, minLenArguments, false))
		assert.False(t, determineIsSCCallAfter(vmInput, address, minLenArguments, true))
	})
	t.Run("not a sc address should return false", func(t *testing.T) {
		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				Arguments: [][]byte{[]byte("arg1"), []byte("arg2"), []byte("arg3"), []byte("arg4"), []byte("arg5")},
			},
		}

		assert.False(t, determineIsSCCallAfter(vmInput, address, minLenArguments, false))
		assert.False(t, determineIsSCCallAfter(vmInput, address, minLenArguments, true))
	})
	t.Run("empty last argument", func(t *testing.T) {
		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				Arguments: [][]byte{[]byte("arg1"), []byte("arg2"), []byte("arg3"), []byte("arg4"), []byte("")},
			},
		}

		assert.False(t, determineIsSCCallAfter(vmInput, scAddress, minLenArguments, true))
		assert.True(t, determineIsSCCallAfter(vmInput, scAddress, minLenArguments, false))
	})
	t.Run("should work", func(t *testing.T) {
		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				Arguments: [][]byte{[]byte("arg1"), []byte("arg2"), []byte("arg3"), []byte("arg4"), []byte("arg5")},
			},
		}

		t.Run("ReturnCallAfterError == false", func(t *testing.T) {
			assert.True(t, determineIsSCCallAfter(vmInput, scAddress, minLenArguments, true))
			assert.True(t, determineIsSCCallAfter(vmInput, scAddress, minLenArguments, false))
		})
		t.Run("ReturnCallAfterError == true and CallType == AsynchronousCallBack", func(t *testing.T) {
			vmInput.CallType = vm.AsynchronousCallBack
			vmInput.ReturnCallAfterError = true
			assert.True(t, determineIsSCCallAfter(vmInput, scAddress, minLenArguments, true))
			assert.True(t, determineIsSCCallAfter(vmInput, scAddress, minLenArguments, false))
		})
	})
}

func TestMustVerifyPayable(t *testing.T) {
	t.Parallel()

	minLenArguments := 4
	t.Run("call type is AsynchronousCall should return false", func(t *testing.T) {
		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				Arguments: [][]byte{[]byte("arg1"), []byte("arg2"), []byte("arg3")},
				CallType:  vm.AsynchronousCall,
			},
		}

		assert.False(t, mustVerifyPayable(vmInput, minLenArguments, true))
		assert.False(t, mustVerifyPayable(vmInput, minLenArguments, false))
	})
	t.Run("call type is ESDTTransferAndExecute should return false", func(t *testing.T) {
		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				Arguments: [][]byte{[]byte("arg1"), []byte("arg2"), []byte("arg3")},
				CallType:  vm.ESDTTransferAndExecute,
			},
		}

		assert.False(t, mustVerifyPayable(vmInput, minLenArguments, true))
		assert.False(t, mustVerifyPayable(vmInput, minLenArguments, false))
	})
	t.Run("arguments represents a SC call should return false", func(t *testing.T) {
		t.Run("5 arguments", func(t *testing.T) {
			vmInput := &vmcommon.ContractCallInput{
				VMInput: vmcommon.VMInput{
					Arguments: [][]byte{[]byte("arg1"), []byte("arg2"), []byte("arg3"), []byte("arg4"), []byte("arg5")},
					CallType:  vm.DirectCall,
				},
			}
			assert.False(t, mustVerifyPayable(vmInput, minLenArguments, true))
			assert.False(t, mustVerifyPayable(vmInput, minLenArguments, false))
		})
		t.Run("6 arguments", func(t *testing.T) {
			vmInput := &vmcommon.ContractCallInput{
				VMInput: vmcommon.VMInput{
					Arguments: [][]byte{[]byte("arg1"), []byte("arg2"), []byte("arg3"), []byte("arg4"), []byte("arg5"), []byte("arg6")},
					CallType:  vm.DirectCall,
				},
			}
			assert.False(t, mustVerifyPayable(vmInput, minLenArguments, true))
			assert.False(t, mustVerifyPayable(vmInput, minLenArguments, false))
		})
	})
	t.Run("caller is ESDT address should return false", func(t *testing.T) {
		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				Arguments:  [][]byte{[]byte("arg1"), []byte("arg2"), []byte("arg3")},
				CallType:   vm.DirectCall,
				CallerAddr: core.ESDTSCAddress,
			},
		}

		assert.False(t, mustVerifyPayable(vmInput, minLenArguments, true))
		assert.False(t, mustVerifyPayable(vmInput, minLenArguments, false))
	})
	t.Run("should return true", func(t *testing.T) {
		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				Arguments: [][]byte{[]byte("arg1"), []byte("arg2"), []byte("arg3")},
			},
		}

		t.Run("call type is DirectCall", func(t *testing.T) {
			vmInput.CallType = vm.DirectCall
			assert.True(t, mustVerifyPayable(vmInput, minLenArguments, true))
			assert.True(t, mustVerifyPayable(vmInput, minLenArguments, false))
		})
		t.Run("call type is AsynchronousCallBack", func(t *testing.T) {
			vmInput.CallType = vm.AsynchronousCallBack
			assert.True(t, mustVerifyPayable(vmInput, minLenArguments, true))
			assert.True(t, mustVerifyPayable(vmInput, minLenArguments, false))
		})
		t.Run("call type is ExecOnDestByCaller", func(t *testing.T) {
			vmInput.CallType = vm.ExecOnDestByCaller
			assert.True(t, mustVerifyPayable(vmInput, minLenArguments, true))
			assert.True(t, mustVerifyPayable(vmInput, minLenArguments, false))
		})
		t.Run("equal arguments than minimum", func(t *testing.T) {
			vmInput.Arguments = [][]byte{[]byte("arg1"), []byte("arg2"), []byte("arg3"), []byte("arg4")}
			vmInput.CallType = vm.ExecOnDestByCaller
			assert.True(t, mustVerifyPayable(vmInput, minLenArguments, true))
			assert.True(t, mustVerifyPayable(vmInput, minLenArguments, false))
		})
		t.Run("5 arguments but no function", func(t *testing.T) {
			vmInput.Arguments = [][]byte{[]byte("arg1"), []byte("arg2"), []byte("arg3"), []byte("arg4"), make([]byte, 0)}
			vmInput.CallType = vm.ExecOnDestByCaller
			assert.True(t, mustVerifyPayable(vmInput, minLenArguments, true))
			t.Run("backwards compatibility", func(t *testing.T) {
				assert.False(t, mustVerifyPayable(vmInput, minLenArguments, false))
			})
		})
	})
}

func TestESDTTransfer_EpochChange(t *testing.T) {
	t.Parallel()

	var functionHandler vmcommon.EpochSubscriberHandler
	notifier := &mock.EpochNotifierStub{
		RegisterNotifyHandlerCalled: func(handler vmcommon.EpochSubscriberHandler) {
			functionHandler = handler
		},
	}
	transferFunc, _ := NewESDTTransferFunc(
		10,
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.ShardCoordinatorStub{},
		&mock.ESDTRoleHandlerStub{},
		1,
		2,
		3,
		notifier,
	)

	functionHandler.EpochConfirmed(0, 0)
	assert.False(t, transferFunc.flagTransferToMeta.IsSet())
	assert.False(t, transferFunc.flagCheckCorrectTokenID.IsSet())
	assert.False(t, transferFunc.flagCheckFunctionArgument.IsSet())

	functionHandler.EpochConfirmed(1, 0)
	assert.True(t, transferFunc.flagTransferToMeta.IsSet())
	assert.False(t, transferFunc.flagCheckCorrectTokenID.IsSet())
	assert.False(t, transferFunc.flagCheckFunctionArgument.IsSet())

	functionHandler.EpochConfirmed(2, 0)
	assert.True(t, transferFunc.flagTransferToMeta.IsSet())
	assert.True(t, transferFunc.flagCheckCorrectTokenID.IsSet())
	assert.False(t, transferFunc.flagCheckFunctionArgument.IsSet())

	functionHandler.EpochConfirmed(3, 0)
	assert.True(t, transferFunc.flagTransferToMeta.IsSet())
	assert.True(t, transferFunc.flagCheckCorrectTokenID.IsSet())
	assert.True(t, transferFunc.flagCheckFunctionArgument.IsSet())

	functionHandler.EpochConfirmed(4, 0)
	assert.True(t, transferFunc.flagTransferToMeta.IsSet())
	assert.True(t, transferFunc.flagCheckCorrectTokenID.IsSet())
	assert.True(t, transferFunc.flagCheckFunctionArgument.IsSet())
}
