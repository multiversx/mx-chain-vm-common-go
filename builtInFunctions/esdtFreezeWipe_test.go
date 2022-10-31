package builtInFunctions

import (
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/data/esdt"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/mock"
	"github.com/stretchr/testify/assert"
)

func TestESDTFreezeWipe_ProcessBuiltInFunctionErrors(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	freeze, _ := NewESDTFreezeWipeFunc(marshaller, true, false)
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
	freeze, _ := NewESDTFreezeWipeFunc(marshaller, true, false)
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

	unFreeze, _ := NewESDTFreezeWipeFunc(marshaller, false, false)
	_, err = unFreeze.ProcessBuiltinFunction(nil, acnt, input)
	assert.Nil(t, err)

	marshaledData, _, _ = acnt.AccountDataHandler().RetrieveValue(esdtKey)
	_ = marshaller.Unmarshal(esdtToken, marshaledData)

	esdtUserData = ESDTUserMetadataFromBytes(esdtToken.Properties)
	assert.False(t, esdtUserData.Frozen)

	// cannot wipe if account is not frozen
	wipe, _ := NewESDTFreezeWipeFunc(marshaller, false, true)
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

	wipe, _ = NewESDTFreezeWipeFunc(marshaller, false, true)
	vmOutput, err := wipe.ProcessBuiltinFunction(nil, acnt, input)
	assert.NoError(t, err)

	marshaledData, _, _ = acnt.AccountDataHandler().RetrieveValue(esdtKey)
	assert.Equal(t, 0, len(marshaledData))
	assert.Len(t, vmOutput.Logs, 1)
	assert.Equal(t, [][]byte{key, {}, wipedAmount.Bytes(), []byte("dst")}, vmOutput.Logs[0].Topics)
}
