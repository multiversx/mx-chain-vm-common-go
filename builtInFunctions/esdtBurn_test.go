package builtInFunctions

import (
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/data/esdt"
	"github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/mock"
	"github.com/stretchr/testify/assert"
)

func TestESDTBurn_ProcessBuiltInFunctionErrors(t *testing.T) {
	t.Parallel()

	pauseHandler := &mock.PauseHandlerStub{}
	burnFunc, _ := NewESDTBurnFunc(10, &mock.MarshalizerMock{}, pauseHandler)
	_, err := burnFunc.ProcessBuiltinFunction(nil, nil, nil)
	assert.Equal(t, err, ErrNilVmInput)

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
		},
	}
	_, err = burnFunc.ProcessBuiltinFunction(nil, nil, input)
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
	_, err = burnFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrAddressIsNotESDTSystemSC)

	input.RecipientAddr = vmcommon.ESDTSCAddress
	input.GasProvided = burnFunc.funcGasCost - 1
	accSnd := mock.NewUserAccount([]byte("dst"))
	_, err = burnFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrNotEnoughGas)

	_, err = burnFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrNilUserAccount)

	pauseHandler.IsPausedCalled = func(token []byte) bool {
		return true
	}
	input.GasProvided = burnFunc.funcGasCost
	_, err = burnFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrESDTTokenIsPaused)
}

func TestESDTBurn_ProcessBuiltInFunctionSenderBurns(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.MarshalizerMock{}
	pauseHandler := &mock.PauseHandlerStub{}
	burnFunc, _ := NewESDTBurnFunc(10, marshalizer, pauseHandler)

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
		},
		RecipientAddr: vmcommon.ESDTSCAddress,
	}
	key := []byte("key")
	value := big.NewInt(10).Bytes()
	input.Arguments = [][]byte{key, value}
	accSnd := mock.NewUserAccount([]byte("snd"))

	esdtFrozen := ESDTUserMetadata{Frozen: true}
	esdtNotFrozen := ESDTUserMetadata{Frozen: false}

	esdtKey := append(burnFunc.keyPrefix, key...)
	esdtToken := &esdt.ESDigitalToken{Value: big.NewInt(100), Properties: esdtFrozen.ToBytes()}
	marshaledData, _ := marshalizer.Marshal(esdtToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(esdtKey, marshaledData)

	_, err := burnFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrESDTIsFrozenForAccount)

	pauseHandler.IsPausedCalled = func(token []byte) bool {
		return true
	}
	esdtToken = &esdt.ESDigitalToken{Value: big.NewInt(100), Properties: esdtNotFrozen.ToBytes()}
	marshaledData, _ = marshalizer.Marshal(esdtToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(esdtKey, marshaledData)

	_, err = burnFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrESDTTokenIsPaused)

	pauseHandler.IsPausedCalled = func(token []byte) bool {
		return false
	}
	_, err = burnFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Nil(t, err)

	marshaledData, _ = accSnd.AccountDataHandler().RetrieveValue(esdtKey)
	_ = marshalizer.Unmarshal(esdtToken, marshaledData)
	assert.True(t, esdtToken.Value.Cmp(big.NewInt(90)) == 0)

	value = big.NewInt(100).Bytes()
	input.Arguments = [][]byte{key, value}
	_, err = burnFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrInsufficientFunds)

	value = big.NewInt(90).Bytes()
	input.Arguments = [][]byte{key, value}
	_, err = burnFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Nil(t, err)

	marshaledData, _ = accSnd.AccountDataHandler().RetrieveValue(esdtKey)
	assert.Equal(t, len(marshaledData), 0)
}
