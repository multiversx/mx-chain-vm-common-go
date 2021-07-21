package builtInFunctions

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/data/esdt"
	"github.com/ElrondNetwork/elrond-go-core/data/vm"
	"github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/mock"
	"github.com/stretchr/testify/assert"
)

func TestESDTTransfer_ProcessBuiltInFunctionErrors(t *testing.T) {
	t.Parallel()

	shardC := &mock.ShardCoordinatorStub{}
	transferFunc, _ := NewESDTTransferFunc(10, &mock.MarshalizerMock{}, &mock.PauseHandlerStub{}, shardC)
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
	input.RecipientAddr = vmcommon.ESDTSCAddress
	shardC.ComputeIdCalled = func(address []byte) uint32 {
		return vmcommon.MetachainShardId
	}
	_, err = transferFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrInvalidRcvAddr)
}

func TestESDTTransfer_ProcessBuiltInFunctionSingleShard(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.MarshalizerMock{}
	transferFunc, _ := NewESDTTransferFunc(10, marshalizer, &mock.PauseHandlerStub{}, &mock.ShardCoordinatorStub{})
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
	transferFunc, _ := NewESDTTransferFunc(10, marshalizer, &mock.PauseHandlerStub{}, &mock.ShardCoordinatorStub{})
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
	transferFunc, _ := NewESDTTransferFunc(10, marshalizer, &mock.PauseHandlerStub{}, &mock.ShardCoordinatorStub{})
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
	esdtPauseFunc, _ := NewESDTPauseFunc(accountStub, true)
	transferFunc, _ := NewESDTTransferFunc(10, marshalizer, esdtPauseFunc, &mock.ShardCoordinatorStub{})
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
	pauseKey := []byte(vmcommon.ElrondProtectedKeyPrefix + vmcommon.ESDTKeyIdentifier + string(key))
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

func TestESDTTransfer_ProcessBuiltInFunctionOnAsyncCallBack(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.MarshalizerMock{}
	transferFunc, _ := NewESDTTransferFunc(10, marshalizer, &mock.PauseHandlerStub{}, &mock.ShardCoordinatorStub{})
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
	accDst := mock.NewUserAccount(vmcommon.ESDTSCAddress)

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
