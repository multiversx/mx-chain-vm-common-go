package builtInFunctions

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data/esdt"
	"github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/mock"
	"github.com/stretchr/testify/require"
)

func TestNewESDTNFTUpdateAttributesFunc(t *testing.T) {
	t.Parallel()

	// nil marshalizer
	e, err := NewESDTNFTUpdateAttributesFunc(10, vmcommon.BaseOperationCost{}, nil, nil, nil, 0, &mock.EpochNotifierStub{})
	require.True(t, check.IfNil(e))
	require.Equal(t, ErrNilMarshalizer, err)

	// nil pause handler
	e, err = NewESDTNFTUpdateAttributesFunc(10, vmcommon.BaseOperationCost{}, &mock.MarshalizerMock{}, nil, nil, 0, &mock.EpochNotifierStub{})
	require.True(t, check.IfNil(e))
	require.Equal(t, ErrNilPauseHandler, err)

	// nil roles handler
	e, err = NewESDTNFTUpdateAttributesFunc(10, vmcommon.BaseOperationCost{}, &mock.MarshalizerMock{}, &mock.PauseHandlerStub{}, nil, 0, &mock.EpochNotifierStub{})
	require.True(t, check.IfNil(e))
	require.Equal(t, ErrNilRolesHandler, err)

	// nil epoch notifier
	e, err = NewESDTNFTUpdateAttributesFunc(10, vmcommon.BaseOperationCost{}, &mock.MarshalizerMock{}, &mock.PauseHandlerStub{}, &mock.ESDTRoleHandlerStub{}, 0, nil)
	require.True(t, check.IfNil(e))
	require.Equal(t, ErrNilEpochHandler, err)

	// should work
	e, err = NewESDTNFTUpdateAttributesFunc(10, vmcommon.BaseOperationCost{}, &mock.MarshalizerMock{}, &mock.PauseHandlerStub{}, &mock.ESDTRoleHandlerStub{}, 1, &mock.EpochNotifierStub{})
	require.False(t, check.IfNil(e))
	require.NoError(t, err)
	require.False(t, e.IsActive())
}

func TestESDTNFTUpdateAttributes_SetNewGasConfig_NilGasCost(t *testing.T) {
	t.Parallel()

	defaultGasCost := uint64(10)
	e, _ := NewESDTNFTUpdateAttributesFunc(defaultGasCost, vmcommon.BaseOperationCost{}, &mock.MarshalizerMock{}, &mock.PauseHandlerStub{}, &mock.ESDTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})

	e.SetNewGasConfig(nil)
	require.Equal(t, defaultGasCost, e.funcGasCost)
}

func TestESDTNFTUpdateAttributes_SetNewGasConfig_ShouldWork(t *testing.T) {
	t.Parallel()

	defaultGasCost := uint64(10)
	newGasCost := uint64(37)
	e, _ := NewESDTNFTUpdateAttributesFunc(defaultGasCost, vmcommon.BaseOperationCost{}, &mock.MarshalizerMock{}, &mock.PauseHandlerStub{}, &mock.ESDTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})

	e.SetNewGasConfig(
		&vmcommon.GasCost{
			BuiltInCost: vmcommon.BuiltInCost{
				ESDTNFTUpdateAttributes: newGasCost,
			},
		},
	)

	require.Equal(t, newGasCost, e.funcGasCost)
}

func TestESDTNFTUpdateAttributes_ProcessBuiltinFunctionErrorOnCheckInput(t *testing.T) {
	t.Parallel()

	e, _ := NewESDTNFTUpdateAttributesFunc(10, vmcommon.BaseOperationCost{}, &mock.MarshalizerMock{}, &mock.PauseHandlerStub{}, &mock.ESDTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})

	// nil vm input
	output, err := e.ProcessBuiltinFunction(mock.NewAccountWrapMock([]byte("addr")), nil, nil)
	require.Nil(t, output)
	require.Equal(t, ErrNilVmInput, err)

	// vm input - value not zero
	output, err = e.ProcessBuiltinFunction(
		mock.NewAccountWrapMock([]byte("addr")),
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue: big.NewInt(37),
			},
		},
	)
	require.Nil(t, output)
	require.Equal(t, ErrBuiltInFunctionCalledWithValue, err)

	// vm input - invalid number of arguments
	output, err = e.ProcessBuiltinFunction(
		mock.NewAccountWrapMock([]byte("addr")),
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue: big.NewInt(0),
				Arguments: [][]byte{[]byte("single arg")},
			},
		},
	)
	require.Nil(t, output)
	require.Equal(t, ErrInvalidArguments, err)

	// vm input - invalid number of arguments
	output, err = e.ProcessBuiltinFunction(
		mock.NewAccountWrapMock([]byte("addr")),
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue: big.NewInt(0),
				Arguments: [][]byte{[]byte("arg0")},
			},
		},
	)
	require.Nil(t, output)
	require.Equal(t, ErrInvalidArguments, err)

	// vm input - invalid receiver
	output, err = e.ProcessBuiltinFunction(
		mock.NewAccountWrapMock([]byte("addr")),
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:  big.NewInt(0),
				Arguments:  [][]byte{[]byte("arg0"), []byte("arg1")},
				CallerAddr: []byte("address 1"),
			},
			RecipientAddr: []byte("address 2"),
		},
	)
	require.Nil(t, output)
	require.Equal(t, ErrInvalidRcvAddr, err)

	// nil user account
	output, err = e.ProcessBuiltinFunction(
		nil,
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:  big.NewInt(0),
				Arguments:  [][]byte{[]byte("arg0"), []byte("arg1")},
				CallerAddr: []byte("address 1"),
			},
			RecipientAddr: []byte("address 1"),
		},
	)
	require.Nil(t, output)
	require.Equal(t, ErrNilUserAccount, err)

	// not enough gas
	output, err = e.ProcessBuiltinFunction(
		mock.NewAccountWrapMock([]byte("addr")),
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte("arg0"), []byte("arg1")},
				CallerAddr:  []byte("address 1"),
				GasProvided: 1,
			},
			RecipientAddr: []byte("address 1"),
		},
	)
	require.Nil(t, output)
	require.Equal(t, ErrNotEnoughGas, err)
}

func TestESDTNFTUpdateAttributes_ProcessBuiltinFunctionInvalidNumberOfArguments(t *testing.T) {
	t.Parallel()

	e, _ := NewESDTNFTUpdateAttributesFunc(10, vmcommon.BaseOperationCost{}, &mock.MarshalizerMock{}, &mock.PauseHandlerStub{}, &mock.ESDTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})
	output, err := e.ProcessBuiltinFunction(
		mock.NewAccountWrapMock([]byte("addr")),
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte("arg0"), []byte("arg1")},
				CallerAddr:  []byte("address 1"),
				GasProvided: 12,
			},
			RecipientAddr: []byte("address 1"),
		},
	)
	require.Nil(t, output)
	require.Equal(t, ErrInvalidArguments, err)
}

func TestESDTNFTUpdateAttributes_ProcessBuiltinFunctionCheckAllowedToExecuteError(t *testing.T) {
	t.Parallel()

	localErr := errors.New("err")
	rolesHandler := &mock.ESDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(_ vmcommon.UserAccountHandler, _ []byte, _ []byte) error {
			return localErr
		},
	}
	e, _ := NewESDTNFTUpdateAttributesFunc(10, vmcommon.BaseOperationCost{}, &mock.MarshalizerMock{}, &mock.PauseHandlerStub{}, rolesHandler, 0, &mock.EpochNotifierStub{})
	output, err := e.ProcessBuiltinFunction(
		mock.NewAccountWrapMock([]byte("addr")),
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte("arg0"), []byte("arg1"), []byte("arg2")},
				CallerAddr:  []byte("address 1"),
				GasProvided: 12,
			},
			RecipientAddr: []byte("address 1"),
		},
	)

	require.Nil(t, output)
	require.Equal(t, localErr, err)
}

func TestESDTNFTUpdateAttributes_ProcessBuiltinFunctionNewSenderShouldErr(t *testing.T) {
	t.Parallel()

	e, _ := NewESDTNFTUpdateAttributesFunc(10, vmcommon.BaseOperationCost{}, &mock.MarshalizerMock{}, &mock.PauseHandlerStub{}, &mock.ESDTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})
	output, err := e.ProcessBuiltinFunction(
		mock.NewAccountWrapMock([]byte("addr")),
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte("arg0"), []byte("arg1"), []byte("arg2")},
				CallerAddr:  []byte("address 1"),
				GasProvided: 12,
			},
			RecipientAddr: []byte("address 1"),
		},
	)

	require.Nil(t, output)
	require.Error(t, err)
	require.Equal(t, ErrNewNFTDataOnSenderAddress, err)
}

func TestESDTNFTUpdateAttributes_ProcessBuiltinFunctionMetaDataMissing(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.MarshalizerMock{}
	e, _ := NewESDTNFTUpdateAttributesFunc(10, vmcommon.BaseOperationCost{}, &mock.MarshalizerMock{}, &mock.PauseHandlerStub{}, &mock.ESDTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	esdtData := &esdt.ESDigitalToken{}
	esdtDataBytes, _ := marshalizer.Marshal(esdtData)
	_ = userAcc.AccountDataHandler().SaveKeyValue([]byte(vmcommon.ElrondProtectedKeyPrefix+vmcommon.ESDTKeyIdentifier+"arg0"+"arg1"), esdtDataBytes)
	output, err := e.ProcessBuiltinFunction(
		userAcc,
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte("arg0"), {0}, []byte("arg2")},
				CallerAddr:  []byte("address 1"),
				GasProvided: 12,
			},
			RecipientAddr: []byte("address 1"),
		},
	)

	require.Nil(t, output)
	require.Equal(t, ErrNFTDoesNotHaveMetadata, err)
}

func TestESDTNFTUpdateAttributes_ProcessBuiltinFunctionShouldErrOnSaveBecauseTokenIsPaused(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.MarshalizerMock{}
	pauseHandler := &mock.PauseHandlerStub{
		IsPausedCalled: func(_ []byte) bool {
			return true
		},
	}

	e, _ := NewESDTNFTUpdateAttributesFunc(10, vmcommon.BaseOperationCost{}, &mock.MarshalizerMock{}, pauseHandler, &mock.ESDTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	esdtData := &esdt.ESDigitalToken{
		TokenMetaData: &esdt.MetaData{
			Name: []byte("test"),
		},
		Value: big.NewInt(10),
	}
	esdtDataBytes, _ := marshalizer.Marshal(esdtData)
	_ = userAcc.AccountDataHandler().SaveKeyValue([]byte(vmcommon.ElrondProtectedKeyPrefix+vmcommon.ESDTKeyIdentifier+"arg0"+"arg1"), esdtDataBytes)

	output, err := e.ProcessBuiltinFunction(
		userAcc,
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte("arg0"), []byte("arg1"), []byte("arg2")},
				CallerAddr:  []byte("address 1"),
				GasProvided: 12,
			},
			RecipientAddr: []byte("address 1"),
		},
	)

	require.Nil(t, output)
	require.Equal(t, ErrESDTTokenIsPaused, err)
}

func TestESDTNFTUpdateAttributes_ProcessBuiltinFunctionShouldWork(t *testing.T) {
	t.Parallel()

	tokenIdentifier := "testTkn"
	key := vmcommon.ElrondProtectedKeyPrefix + vmcommon.ESDTKeyIdentifier + tokenIdentifier

	nonce := big.NewInt(33)
	initialValue := big.NewInt(5)
	newAttributes := []byte("NewURI")

	marshalizer := &mock.MarshalizerMock{}
	e, _ := NewESDTNFTUpdateAttributesFunc(10, vmcommon.BaseOperationCost{}, &mock.MarshalizerMock{}, &mock.PauseHandlerStub{}, &mock.ESDTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	esdtData := &esdt.ESDigitalToken{
		TokenMetaData: &esdt.MetaData{
			Name: []byte("test"),
		},
		Value: initialValue,
	}
	esdtDataBytes, _ := marshalizer.Marshal(esdtData)
	tokenKey := append([]byte(key), nonce.Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtDataBytes)

	output, err := e.ProcessBuiltinFunction(
		userAcc,
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte(tokenIdentifier), nonce.Bytes(), newAttributes},
				CallerAddr:  []byte("address 1"),
				GasProvided: 12,
			},
			RecipientAddr: []byte("address 1"),
		},
	)

	require.NotNil(t, output)
	require.NoError(t, err)
	require.Equal(t, vmcommon.Ok, output.ReturnCode)

	res, err := userAcc.AccountDataHandler().RetrieveValue([]byte(key))
	require.NoError(t, err)
	require.NotNil(t, res)

	finalTokenData := esdt.ESDigitalToken{}
	_ = marshalizer.Unmarshal(&finalTokenData, res)
	require.Equal(t, finalTokenData.TokenMetaData.Attributes, newAttributes)
}
