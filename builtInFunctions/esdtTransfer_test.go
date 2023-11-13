package builtInFunctions

import (
	"bytes"
	"math/big"
	"strings"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data/esdt"
	"github.com/multiversx/mx-chain-core-go/data/vm"
	"github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/assert"
)

func TestNewESDTTransferFunc(t *testing.T) {
	t.Parallel()

	t.Run("nil marshaller should error", func(t *testing.T) {
		t.Parallel()

		transferFunc, err := NewESDTTransferFunc(10, nil, nil, nil, nil, nil)
		assert.Equal(t, ErrNilMarshalizer, err)
		assert.True(t, check.IfNil(transferFunc))
	})
	t.Run("nil global settings handler should error", func(t *testing.T) {
		t.Parallel()

		transferFunc, err := NewESDTTransferFunc(10, &mock.MarshalizerMock{}, nil, nil, nil, nil)
		assert.Equal(t, ErrNilGlobalSettingsHandler, err)
		assert.True(t, check.IfNil(transferFunc))
	})
	t.Run("nil shard coordinator should error", func(t *testing.T) {
		t.Parallel()

		transferFunc, err := NewESDTTransferFunc(10, &mock.MarshalizerMock{}, &mock.GlobalSettingsHandlerStub{}, nil, nil, nil)
		assert.Equal(t, ErrNilShardCoordinator, err)
		assert.True(t, check.IfNil(transferFunc))
	})
	t.Run("nil roles handler should error", func(t *testing.T) {
		t.Parallel()

		transferFunc, err := NewESDTTransferFunc(10, &mock.MarshalizerMock{}, &mock.GlobalSettingsHandlerStub{}, &mock.ShardCoordinatorStub{}, nil, nil)
		assert.Equal(t, ErrNilRolesHandler, err)
		assert.True(t, check.IfNil(transferFunc))
	})
	t.Run("nil enable epochs handler should error", func(t *testing.T) {
		t.Parallel()

		transferFunc, err := NewESDTTransferFunc(10, &mock.MarshalizerMock{}, &mock.GlobalSettingsHandlerStub{}, &mock.ShardCoordinatorStub{}, &mock.ESDTRoleHandlerStub{}, nil)
		assert.Equal(t, ErrNilEnableEpochsHandler, err)
		assert.True(t, check.IfNil(transferFunc))
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		transferFunc, err := NewESDTTransferFunc(10, &mock.MarshalizerMock{}, &mock.GlobalSettingsHandlerStub{}, &mock.ShardCoordinatorStub{}, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{})
		assert.Nil(t, err)
		assert.False(t, check.IfNil(transferFunc))
	})
}
func TestESDTTransfer_ProcessBuiltInFunctionErrors(t *testing.T) {
	t.Parallel()

	shardC := &mock.ShardCoordinatorStub{}
	transferFunc, _ := NewESDTTransferFunc(10, &mock.MarshalizerMock{}, &mock.GlobalSettingsHandlerStub{}, shardC, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsTransferToMetaFlagEnabledField:                     false,
		IsCheckCorrectTokenIDForTransferRoleFlagEnabledField: true,
	})
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})
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

	marshaller := &mock.MarshalizerMock{}
	esdtRoleHandler := &mock.ESDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			assert.Equal(t, core.ESDTRoleTransfer, string(action))
			return nil
		},
	}
	enableEpochsHandler := &mock.EnableEpochsHandlerStub{
		IsTransferToMetaFlagEnabledField:                     false,
		IsCheckCorrectTokenIDForTransferRoleFlagEnabledField: true,
	}
	transferFunc, _ := NewESDTTransferFunc(10, marshaller, &mock.GlobalSettingsHandlerStub{}, &mock.ShardCoordinatorStub{}, esdtRoleHandler, enableEpochsHandler)
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})

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
	marshaledData, _ := marshaller.Marshal(esdtToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(esdtKey, marshaledData)

	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Nil(t, err)
	marshaledData, _, _ = accSnd.AccountDataHandler().RetrieveValue(esdtKey)
	_ = marshaller.Unmarshal(esdtToken, marshaledData)
	assert.True(t, esdtToken.Value.Cmp(big.NewInt(90)) == 0)

	marshaledData, _, _ = accDst.AccountDataHandler().RetrieveValue(esdtKey)
	_ = marshaller.Unmarshal(esdtToken, marshaledData)
	assert.True(t, esdtToken.Value.Cmp(big.NewInt(10)) == 0)
}

func TestESDTTransfer_ProcessBuiltInFunctionSenderInShard(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	transferFunc, _ := NewESDTTransferFunc(10, marshaller, &mock.GlobalSettingsHandlerStub{}, &mock.ShardCoordinatorStub{}, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsTransferToMetaFlagEnabledField:                     false,
		IsCheckCorrectTokenIDForTransferRoleFlagEnabledField: true,
	})
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})

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
	marshaledData, _ := marshaller.Marshal(esdtToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(esdtKey, marshaledData)

	_, err := transferFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Nil(t, err)
	marshaledData, _, _ = accSnd.AccountDataHandler().RetrieveValue(esdtKey)
	_ = marshaller.Unmarshal(esdtToken, marshaledData)
	assert.True(t, esdtToken.Value.Cmp(big.NewInt(90)) == 0)
}

func TestESDTTransfer_ProcessBuiltInFunctionDestInShard(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	transferFunc, _ := NewESDTTransferFunc(10, marshaller, &mock.GlobalSettingsHandlerStub{}, &mock.ShardCoordinatorStub{}, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsTransferToMetaFlagEnabledField:                     false,
		IsCheckCorrectTokenIDForTransferRoleFlagEnabledField: true,
	})
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})

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
	marshaledData, _, _ := accDst.AccountDataHandler().RetrieveValue(esdtKey)
	_ = marshaller.Unmarshal(esdtToken, marshaledData)
	assert.True(t, esdtToken.Value.Cmp(big.NewInt(10)) == 0)
	assert.Equal(t, uint64(0), vmOutput.GasRemaining)
}

func TestESDTTransfer_ProcessBuiltInFunctionTooLongValue(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	transferFunc, _ := NewESDTTransferFunc(10, marshaller, &mock.GlobalSettingsHandlerStub{}, &mock.ShardCoordinatorStub{}, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsTransferToMetaFlagEnabledField:                     false,
		IsCheckCorrectTokenIDForTransferRoleFlagEnabledField: true,
	})
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})

	bigValueStr := "1" + strings.Repeat("0", 1000)
	bigValue, _ := big.NewInt(0).SetString(bigValueStr, 10)
	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("tkn"), bigValue.Bytes()},
		},
	}
	accDst := mock.NewUserAccount([]byte("dst"))

	// before the activation of the flag, large values should not return error
	vmOutput, err := transferFunc.ProcessBuiltinFunction(nil, accDst, input)
	assert.Nil(t, err)
	assert.NotEmpty(t, vmOutput)

	// after the activation, it should return an error
	transferFunc.enableEpochsHandler = &mock.EnableEpochsHandlerStub{
		IsConsistentTokensValuesLengthCheckEnabledField: true,
	}
	vmOutput, err = transferFunc.ProcessBuiltinFunction(nil, accDst, input)
	assert.Equal(t, "invalid arguments to process built-in function: max length for esdt transfer value is 100", err.Error())
	assert.Empty(t, vmOutput)
}

func TestESDTTransfer_SndDstFrozen(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	accountStub := &mock.AccountsStub{}
	esdtGlobalSettingsFunc, _ := NewESDTGlobalSettingsFunc(accountStub, marshaller, true, core.BuiltInFunctionESDTPause, trueHandler)
	transferFunc, _ := NewESDTTransferFunc(10, marshaller, esdtGlobalSettingsFunc, &mock.ShardCoordinatorStub{}, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsTransferToMetaFlagEnabledField:                     false,
		IsCheckCorrectTokenIDForTransferRoleFlagEnabledField: true,
	})
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})

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

	esdtFrozen := vmcommon.ESDTUserMetadata{Frozen: true}
	esdtNotFrozen := vmcommon.ESDTUserMetadata{Frozen: false}

	esdtKey := append(transferFunc.keyPrefix, key...)
	esdtToken := &esdt.ESDigitalToken{Value: big.NewInt(100), Properties: esdtFrozen.ToBytes()}
	marshaledData, _ := marshaller.Marshal(esdtToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(esdtKey, marshaledData)

	_, err := transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Equal(t, err, ErrESDTIsFrozenForAccount)

	esdtToken = &esdt.ESDigitalToken{Value: big.NewInt(100), Properties: esdtNotFrozen.ToBytes()}
	marshaledData, _ = marshaller.Marshal(esdtToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(esdtKey, marshaledData)

	esdtToken = &esdt.ESDigitalToken{Value: big.NewInt(100), Properties: esdtFrozen.ToBytes()}
	marshaledData, _ = marshaller.Marshal(esdtToken)
	_ = accDst.AccountDataHandler().SaveKeyValue(esdtKey, marshaledData)

	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Equal(t, err, ErrESDTIsFrozenForAccount)

	marshaledData, _, _ = accDst.AccountDataHandler().RetrieveValue(esdtKey)
	_ = marshaller.Unmarshal(esdtToken, marshaledData)
	assert.True(t, esdtToken.Value.Cmp(big.NewInt(100)) == 0)

	input.ReturnCallAfterError = true
	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Nil(t, err)

	esdtToken = &esdt.ESDigitalToken{Value: big.NewInt(100), Properties: esdtNotFrozen.ToBytes()}
	marshaledData, _ = marshaller.Marshal(esdtToken)
	_ = accDst.AccountDataHandler().SaveKeyValue(esdtKey, marshaledData)

	systemAccount := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	esdtGlobal := vmcommon.ESDTGlobalMetadata{Paused: true}
	pauseKey := []byte(baseESDTKeyPrefix + string(key))
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

	marshaller := &mock.MarshalizerMock{}
	accountStub := &mock.AccountsStub{}
	rolesHandler := &mock.ESDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			if bytes.Equal(action, []byte(core.ESDTRoleTransfer)) {
				return ErrActionNotAllowed
			}
			return nil
		},
	}
	esdtGlobalSettingsFunc, _ := NewESDTGlobalSettingsFunc(accountStub, marshaller, true, core.BuiltInFunctionESDTSetLimitedTransfer, trueHandler)
	transferFunc, _ := NewESDTTransferFunc(10, marshaller, esdtGlobalSettingsFunc, &mock.ShardCoordinatorStub{}, rolesHandler, &mock.EnableEpochsHandlerStub{
		IsTransferToMetaFlagEnabledField:                     false,
		IsCheckCorrectTokenIDForTransferRoleFlagEnabledField: true,
	})
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})

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
	marshaledData, _ := marshaller.Marshal(esdtToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(esdtKey, marshaledData)

	esdtToken = &esdt.ESDigitalToken{Value: big.NewInt(100)}
	marshaledData, _ = marshaller.Marshal(esdtToken)
	_ = accDst.AccountDataHandler().SaveKeyValue(esdtKey, marshaledData)

	systemAccount := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	esdtGlobal := vmcommon.ESDTGlobalMetadata{LimitedTransfer: true}
	pauseKey := []byte(baseESDTKeyPrefix + string(key))
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

	marshaller := &mock.MarshalizerMock{}
	transferFunc, _ := NewESDTTransferFunc(10, marshaller, &mock.GlobalSettingsHandlerStub{}, &mock.ShardCoordinatorStub{}, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsTransferToMetaFlagEnabledField:                     false,
		IsCheckCorrectTokenIDForTransferRoleFlagEnabledField: true,
	})
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})

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
	marshaledData, _ := marshaller.Marshal(esdtToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(esdtKey, marshaledData)

	vmOutput, err := transferFunc.ProcessBuiltinFunction(nil, accDst, input)
	assert.Nil(t, err)

	marshaledData, _, _ = accDst.AccountDataHandler().RetrieveValue(esdtKey)
	_ = marshaller.Unmarshal(esdtToken, marshaledData)
	assert.True(t, esdtToken.Value.Cmp(big.NewInt(10)) == 0)

	assert.Equal(t, vmOutput.GasRemaining, input.GasProvided)

	vmOutput, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Nil(t, err)
	vmOutput.GasRemaining = input.GasProvided - transferFunc.funcGasCost

	marshaledData, _, _ = accSnd.AccountDataHandler().RetrieveValue(esdtKey)
	_ = marshaller.Unmarshal(esdtToken, marshaledData)
	assert.True(t, esdtToken.Value.Cmp(big.NewInt(90)) == 0)
}
