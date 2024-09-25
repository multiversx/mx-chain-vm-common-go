package builtInFunctions

import (
	"errors"
	"math/big"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data/esdt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
)

func TestNewESDTNFTUpdateAttributesFunc(t *testing.T) {
	t.Parallel()

	t.Run("nil marshaller should error", func(t *testing.T) {
		t.Parallel()

		e, err := NewESDTNFTUpdateAttributesFunc(10, vmcommon.BaseOperationCost{}, nil, nil, nil, nil, nil)
		require.True(t, check.IfNil(e))
		require.Equal(t, ErrNilESDTNFTStorageHandler, err)
	})
	t.Run("nil global settings handler should error", func(t *testing.T) {
		t.Parallel()

		e, err := NewESDTNFTUpdateAttributesFunc(10, vmcommon.BaseOperationCost{}, createNewESDTDataStorageHandler(), nil, nil, nil, nil)
		require.True(t, check.IfNil(e))
		require.Equal(t, ErrNilGlobalSettingsHandler, err)
	})
	t.Run("nil roles handler should error", func(t *testing.T) {
		t.Parallel()

		e, err := NewESDTNFTUpdateAttributesFunc(10, vmcommon.BaseOperationCost{}, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, nil, nil, nil)
		require.True(t, check.IfNil(e))
		require.Equal(t, ErrNilRolesHandler, err)
	})
	t.Run("nil enable epochs handler should error", func(t *testing.T) {
		t.Parallel()

		e, err := NewESDTNFTUpdateAttributesFunc(10, vmcommon.BaseOperationCost{}, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{}, nil, nil)
		require.True(t, check.IfNil(e))
		require.Equal(t, ErrNilEnableEpochsHandler, err)
	})
	t.Run("nil marshaller should error", func(t *testing.T) {
		t.Parallel()

		e, err := NewESDTNFTUpdateAttributesFunc(10, vmcommon.BaseOperationCost{}, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{}, nil)
		require.True(t, check.IfNil(e))
		require.Equal(t, ErrNilMarshalizer, err)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		e, err := NewESDTNFTUpdateAttributesFunc(10, vmcommon.BaseOperationCost{}, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{}, &mock.MarshalizerMock{})
		require.False(t, check.IfNil(e))
		require.NoError(t, err)
		require.False(t, e.IsActive())
	})
}

func TestESDTNFTUpdateAttributes_SetNewGasConfig_NilGasCost(t *testing.T) {
	t.Parallel()

	defaultGasCost := uint64(10)
	e, _ := NewESDTNFTUpdateAttributesFunc(defaultGasCost, vmcommon.BaseOperationCost{}, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ESDTNFTImprovementV1Flag
		},
	}, &mock.MarshalizerMock{})

	e.SetNewGasConfig(nil)
	require.Equal(t, defaultGasCost, e.funcGasCost)
}

func TestESDTNFTUpdateAttributes_SetNewGasConfig_ShouldWork(t *testing.T) {
	t.Parallel()

	defaultGasCost := uint64(10)
	newGasCost := uint64(37)
	e, _ := NewESDTNFTUpdateAttributesFunc(defaultGasCost, vmcommon.BaseOperationCost{}, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ESDTNFTImprovementV1Flag
		},
	}, &mock.MarshalizerMock{})

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

	e, _ := NewESDTNFTUpdateAttributesFunc(10, vmcommon.BaseOperationCost{}, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ESDTNFTImprovementV1Flag
		},
	}, &mock.MarshalizerMock{})

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

	e, _ := NewESDTNFTUpdateAttributesFunc(10, vmcommon.BaseOperationCost{}, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ESDTNFTImprovementV1Flag
		},
	}, &mock.MarshalizerMock{})
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
	e, _ := NewESDTNFTUpdateAttributesFunc(10, vmcommon.BaseOperationCost{}, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, rolesHandler, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ESDTNFTImprovementV1Flag
		},
	}, &mock.MarshalizerMock{})
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

	e, _ := NewESDTNFTUpdateAttributesFunc(10, vmcommon.BaseOperationCost{}, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ESDTNFTImprovementV1Flag
		},
	}, &mock.MarshalizerMock{})
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

	marshaller := &mock.MarshalizerMock{}
	e, _ := NewESDTNFTUpdateAttributesFunc(10, vmcommon.BaseOperationCost{}, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ESDTNFTImprovementV1Flag
		},
	}, &mock.MarshalizerMock{})

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	esdtData := &esdt.ESDigitalToken{}
	esdtDataBytes, _ := marshaller.Marshal(esdtData)
	_ = userAcc.AccountDataHandler().SaveKeyValue([]byte(core.ProtectedKeyPrefix+core.ESDTKeyIdentifier+"arg0"+"arg1"), esdtDataBytes)
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

	marshaller := &mock.MarshalizerMock{}
	globalSettingsHandler := &mock.GlobalSettingsHandlerStub{
		IsPausedCalled: func(_ []byte) bool {
			return true
		},
	}
	var enableEpochsHandler = &mock.EnableEpochsHandlerStub{}
	e, _ := NewESDTNFTUpdateAttributesFunc(10, vmcommon.BaseOperationCost{}, createNewESDTDataStorageHandlerWithArgs(globalSettingsHandler, &mock.AccountsStub{}, enableEpochsHandler, &mock.CrossChainTokenCheckerMock{}), globalSettingsHandler, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ESDTNFTImprovementV1Flag
		},
	}, &mock.MarshalizerMock{})

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	esdtData := &esdt.ESDigitalToken{
		TokenMetaData: &esdt.MetaData{
			Name: []byte("test"),
		},
		Value: big.NewInt(10),
	}
	esdtDataBytes, _ := marshaller.Marshal(esdtData)
	_ = userAcc.AccountDataHandler().SaveKeyValue([]byte(core.ProtectedKeyPrefix+core.ESDTKeyIdentifier+"arg0"+"arg1"), esdtDataBytes)

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
	key := baseESDTKeyPrefix + tokenIdentifier

	nonce := big.NewInt(33)
	initialValue := big.NewInt(5)
	newAttributes := []byte("NewURI")

	esdtDataStorage := createNewESDTDataStorageHandler()
	marshaller := &mock.MarshalizerMock{}
	esdtRoleHandler := &mock.ESDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			assert.Equal(t, core.ESDTRoleNFTUpdateAttributes, string(action))
			return nil
		},
	}
	e, _ := NewESDTNFTUpdateAttributesFunc(10, vmcommon.BaseOperationCost{}, esdtDataStorage, &mock.GlobalSettingsHandlerStub{}, esdtRoleHandler, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ESDTNFTImprovementV1Flag
		},
	}, &mock.MarshalizerMock{})

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	esdtData := &esdt.ESDigitalToken{
		Type: uint32(core.NonFungible),
		TokenMetaData: &esdt.MetaData{
			Name: []byte("test"),
		},
		Value: initialValue,
	}
	esdtDataBytes, _ := marshaller.Marshal(esdtData)
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

	res, _, err := userAcc.AccountDataHandler().RetrieveValue(tokenKey)
	require.NoError(t, err)
	require.NotNil(t, res)

	metaData, _ := esdtDataStorage.getESDTMetaDataFromSystemAccount(tokenKey, defaultQueryOptions())
	require.Equal(t, metaData.Attributes, newAttributes)
}
