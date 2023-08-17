package builtInFunctions

import (
	"math/big"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/data/esdt"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestESDTFreezeWipe_ProcessBuiltInFunctionErrors(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	freeze, _ := NewESDTFreezeWipeFunc(createNewESDTDataStorageHandler(), &mock.EnableEpochsHandlerStub{}, marshaller, true, false)
	_, err := freeze.ProcessBuiltinFunction(nil, nil, nil)
	assert.Equal(t, err, ErrNilVmInput)

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
		},
	}
	_, err = freeze.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrInvalidArguments)

	input = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(1),
		},
	}
	_, err = freeze.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrBuiltInFunctionCalledWithValue)

	input.CallValue = big.NewInt(0)
	key := []byte("key")
	value := []byte("value")
	input.Arguments = [][]byte{key, value}
	_, err = freeze.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrInvalidArguments)

	input.Arguments = [][]byte{key}
	_, err = freeze.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrAddressIsNotESDTSystemSC)

	input.CallerAddr = core.ESDTSCAddress
	_, err = freeze.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrNilUserAccount)

	input.RecipientAddr = []byte("dst")
	acnt := mock.NewUserAccount(input.RecipientAddr)
	vmOutput, err := freeze.ProcessBuiltinFunction(nil, acnt, input)
	assert.Nil(t, err)

	frozenAmount := big.NewInt(42)
	esdtToken := &esdt.ESDigitalToken{
		Value: frozenAmount,
	}
	esdtKey := append(freeze.keyPrefix, key...)
	marshaledData, _, _ := acnt.AccountDataHandler().RetrieveValue(esdtKey)
	_ = marshaller.Unmarshal(esdtToken, marshaledData)

	esdtUserData := ESDTUserMetadataFromBytes(esdtToken.Properties)
	assert.True(t, esdtUserData.Frozen)
	assert.Len(t, vmOutput.Logs, 1)
	assert.Equal(t, [][]byte{key, {}, frozenAmount.Bytes(), []byte("dst")}, vmOutput.Logs[0].Topics)
}

func TestESDTFreezeWipe_ProcessBuiltInFunction(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	freeze, _ := NewESDTFreezeWipeFunc(createNewESDTDataStorageHandler(), &mock.EnableEpochsHandlerStub{}, marshaller, true, false)
	_, err := freeze.ProcessBuiltinFunction(nil, nil, nil)
	assert.Equal(t, err, ErrNilVmInput)

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
		},
	}
	key := []byte("key")

	input.Arguments = [][]byte{key}
	input.CallerAddr = core.ESDTSCAddress
	input.RecipientAddr = []byte("dst")
	esdtKey := append(freeze.keyPrefix, key...)
	esdtToken := &esdt.ESDigitalToken{Value: big.NewInt(10)}
	marshaledData, _ := freeze.marshaller.Marshal(esdtToken)
	acnt := mock.NewUserAccount(input.RecipientAddr)
	_ = acnt.AccountDataHandler().SaveKeyValue(esdtKey, marshaledData)

	_, err = freeze.ProcessBuiltinFunction(nil, acnt, input)
	assert.Nil(t, err)

	esdtToken = &esdt.ESDigitalToken{}
	marshaledData, _, _ = acnt.AccountDataHandler().RetrieveValue(esdtKey)
	_ = marshaller.Unmarshal(esdtToken, marshaledData)

	esdtUserData := ESDTUserMetadataFromBytes(esdtToken.Properties)
	assert.True(t, esdtUserData.Frozen)

	unFreeze, _ := NewESDTFreezeWipeFunc(createNewESDTDataStorageHandler(), &mock.EnableEpochsHandlerStub{}, marshaller, false, false)
	_, err = unFreeze.ProcessBuiltinFunction(nil, acnt, input)
	assert.Nil(t, err)

	marshaledData, _, _ = acnt.AccountDataHandler().RetrieveValue(esdtKey)
	_ = marshaller.Unmarshal(esdtToken, marshaledData)

	esdtUserData = ESDTUserMetadataFromBytes(esdtToken.Properties)
	assert.False(t, esdtUserData.Frozen)

	// cannot wipe if account is not frozen
	wipe, _ := NewESDTFreezeWipeFunc(createNewESDTDataStorageHandler(), &mock.EnableEpochsHandlerStub{}, marshaller, false, true)
	_, err = wipe.ProcessBuiltinFunction(nil, acnt, input)
	assert.Equal(t, ErrCannotWipeAccountNotFrozen, err)

	marshaledData, _, _ = acnt.AccountDataHandler().RetrieveValue(esdtKey)
	assert.NotEqual(t, 0, len(marshaledData))

	// can wipe as account is frozen
	metaData := ESDTUserMetadata{Frozen: true}
	wipedAmount := big.NewInt(42)
	esdtToken = &esdt.ESDigitalToken{
		Value:      wipedAmount,
		Properties: metaData.ToBytes(),
	}
	esdtTokenBytes, _ := marshaller.Marshal(esdtToken)
	err = acnt.AccountDataHandler().SaveKeyValue(esdtKey, esdtTokenBytes)
	assert.NoError(t, err)

	wipe, _ = NewESDTFreezeWipeFunc(createNewESDTDataStorageHandler(), &mock.EnableEpochsHandlerStub{}, marshaller, false, true)
	vmOutput, err := wipe.ProcessBuiltinFunction(nil, acnt, input)
	assert.NoError(t, err)

	marshaledData, _, _ = acnt.AccountDataHandler().RetrieveValue(esdtKey)
	assert.Equal(t, 0, len(marshaledData))
	assert.Len(t, vmOutput.Logs, 1)
	assert.Equal(t, [][]byte{key, {}, wipedAmount.Bytes(), []byte("dst")}, vmOutput.Logs[0].Topics)
}

func TestEsdtFreezeWipe_WipeShouldDecreaseLiquidityIfFlagIsEnabled(t *testing.T) {
	t.Parallel()

	balance := big.NewInt(37)
	addToLiquiditySystemAccCalled := false
	esdtStorage := &mock.ESDTNFTStorageHandlerStub{
		AddToLiquiditySystemAccCalled: func(_ []byte, _ uint64, transferValue *big.Int) error {
			require.Equal(t, big.NewInt(0).Neg(balance), transferValue)
			addToLiquiditySystemAccCalled = true
			return nil
		},
	}

	marshaller := &mock.MarshalizerMock{}
	wipe, _ := NewESDTFreezeWipeFunc(esdtStorage, &mock.EnableEpochsHandlerStub{}, marshaller, false, true)

	acnt := mock.NewUserAccount([]byte("dst"))
	metaData := ESDTUserMetadata{Frozen: true}
	esdtToken := &esdt.ESDigitalToken{
		Value:      balance,
		Properties: metaData.ToBytes(),
	}
	esdtTokenBytes, _ := marshaller.Marshal(esdtToken)

	nonce := uint64(37)
	key := append([]byte("MYSFT-0a0a0a"), big.NewInt(int64(nonce)).Bytes()...)
	esdtKey := append(wipe.keyPrefix, key...)

	err := acnt.AccountDataHandler().SaveKeyValue(esdtKey, esdtTokenBytes)
	assert.NoError(t, err)

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
		},
	}
	input.Arguments = [][]byte{key}
	input.CallerAddr = core.ESDTSCAddress
	input.RecipientAddr = []byte("dst")

	acntCopy := acnt.Clone()
	_, err = wipe.ProcessBuiltinFunction(nil, acntCopy, input)
	assert.NoError(t, err)

	marshaledData, _, _ := acntCopy.AccountDataHandler().RetrieveValue(esdtKey)
	assert.Equal(t, 0, len(marshaledData))
	assert.False(t, addToLiquiditySystemAccCalled)

	wipe.enableEpochsHandler = &mock.EnableEpochsHandlerStub{
		IsFlagEnabledInCurrentEpochCalled: func(flag core.EnableEpochFlag) bool {
			return flag == core.WipeSingleNFTLiquidityDecreaseFlag
		},
	}

	_, err = wipe.ProcessBuiltinFunction(nil, acnt, input)
	assert.NoError(t, err)

	marshaledData, _, _ = acnt.AccountDataHandler().RetrieveValue(esdtKey)
	assert.Equal(t, 0, len(marshaledData))
	assert.True(t, addToLiquiditySystemAccCalled)
}
