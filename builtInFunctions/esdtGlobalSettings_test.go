package builtInFunctions

import (
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/mock"
	"github.com/stretchr/testify/assert"
)

func TestESDTPause_ProcessBuiltInFunction(t *testing.T) {
	t.Parallel()

	acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	pauseFunc, _ := NewESDTGlobalSettingsFunc(&mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		},
	}, true, core.BuiltInFunctionESDTPause, 0, &mock.EpochNotifierStub{})
	_, err := pauseFunc.ProcessBuiltinFunction(nil, nil, nil)
	assert.Equal(t, err, ErrNilVmInput)

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
		},
	}
	_, err = pauseFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrInvalidArguments)

	input = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(1),
		},
	}
	_, err = pauseFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrBuiltInFunctionCalledWithValue)

	input.CallValue = big.NewInt(0)
	key := []byte("key")
	value := []byte("value")
	input.Arguments = [][]byte{key, value}
	_, err = pauseFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrInvalidArguments)

	input.Arguments = [][]byte{key}
	_, err = pauseFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrAddressIsNotESDTSystemSC)

	input.CallerAddr = core.ESDTSCAddress
	_, err = pauseFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrOnlySystemAccountAccepted)

	input.RecipientAddr = vmcommon.SystemAccountAddress
	_, err = pauseFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Nil(t, err)

	pauseKey := []byte(core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier + string(key))
	assert.True(t, pauseFunc.IsPaused(pauseKey))

	esdtGlobalSettingsFalse, _ := NewESDTGlobalSettingsFunc(&mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		},
	}, false, core.BuiltInFunctionESDTUnPause, 0, &mock.EpochNotifierStub{})

	_, err = esdtGlobalSettingsFalse.ProcessBuiltinFunction(nil, nil, input)
	assert.Nil(t, err)

	assert.False(t, pauseFunc.IsPaused(pauseKey))
}
