package builtInFunctions

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data/esdt"
	"github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewESDTNFTAddQuantityFunc(t *testing.T) {
	t.Parallel()

	// nil marshaller
	eqf, err := NewESDTNFTAddQuantityFunc(10, nil, nil, nil, 0, nil)
	require.True(t, check.IfNil(eqf))
	require.Equal(t, ErrNilESDTNFTStorageHandler, err)

	// nil pause handler
	eqf, err = NewESDTNFTAddQuantityFunc(10, createNewESDTDataStorageHandler(), nil, nil, 0, nil)
	require.True(t, check.IfNil(eqf))
	require.Equal(t, ErrNilGlobalSettingsHandler, err)

	// nil roles handler
	eqf, err = NewESDTNFTAddQuantityFunc(10, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, nil, 0, nil)
	require.True(t, check.IfNil(eqf))
	require.Equal(t, ErrNilRolesHandler, err)

	// nil epoch handler
	eqf, err = NewESDTNFTAddQuantityFunc(10, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{}, 0, nil)
	require.True(t, check.IfNil(eqf))
	require.Equal(t, ErrNilEpochHandler, err)

	// should work
	eqf, err = NewESDTNFTAddQuantityFunc(10, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})
	require.False(t, check.IfNil(eqf))
	require.NoError(t, err)
}

func TestEsdtNFTAddQuantity_SetNewGasConfig_NilGasCost(t *testing.T) {
	t.Parallel()

	defaultGasCost := uint64(10)
	eqf, _ := NewESDTNFTAddQuantityFunc(defaultGasCost, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})

	eqf.SetNewGasConfig(nil)
	require.Equal(t, defaultGasCost, eqf.funcGasCost)
}

func TestEsdtNFTAddQuantity_SetNewGasConfig_ShouldWork(t *testing.T) {
	t.Parallel()

	defaultGasCost := uint64(10)
	newGasCost := uint64(37)
	eqf, _ := NewESDTNFTAddQuantityFunc(defaultGasCost, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})

	eqf.SetNewGasConfig(
		&vmcommon.GasCost{
			BuiltInCost: vmcommon.BuiltInCost{
				ESDTNFTAddQuantity: newGasCost,
			},
		},
	)

	require.Equal(t, newGasCost, eqf.funcGasCost)
}

func TestEsdtNFTAddQuantity_ProcessBuiltinFunctionErrorOnCheckESDTNFTCreateBurnAddInput(t *testing.T) {
	t.Parallel()

	eqf, _ := NewESDTNFTAddQuantityFunc(10, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})

	// nil vm input
	output, err := eqf.ProcessBuiltinFunction(mock.NewAccountWrapMock([]byte("addr")), nil, nil)
	require.Nil(t, output)
	require.Equal(t, ErrNilVmInput, err)

	// vm input - value not zero
	output, err = eqf.ProcessBuiltinFunction(
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
	output, err = eqf.ProcessBuiltinFunction(
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
	output, err = eqf.ProcessBuiltinFunction(
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
	output, err = eqf.ProcessBuiltinFunction(
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
	output, err = eqf.ProcessBuiltinFunction(
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
	output, err = eqf.ProcessBuiltinFunction(
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

func TestEsdtNFTAddQuantity_ProcessBuiltinFunctionInvalidNumberOfArguments(t *testing.T) {
	t.Parallel()

	eqf, _ := NewESDTNFTAddQuantityFunc(10, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})
	output, err := eqf.ProcessBuiltinFunction(
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

func TestEsdtNFTAddQuantity_ProcessBuiltinFunctionCheckAllowedToExecuteError(t *testing.T) {
	t.Parallel()

	localErr := errors.New("err")
	rolesHandler := &mock.ESDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(_ vmcommon.UserAccountHandler, _ []byte, _ []byte) error {
			return localErr
		},
	}
	eqf, _ := NewESDTNFTAddQuantityFunc(10, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, rolesHandler, 0, &mock.EpochNotifierStub{})
	output, err := eqf.ProcessBuiltinFunction(
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

func TestEsdtNFTAddQuantity_ProcessBuiltinFunctionNewSenderShouldErr(t *testing.T) {
	t.Parallel()

	eqf, _ := NewESDTNFTAddQuantityFunc(10, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})
	output, err := eqf.ProcessBuiltinFunction(
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

func TestEsdtNFTAddQuantity_ProcessBuiltinFunctionMetaDataMissing(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	eqf, _ := NewESDTNFTAddQuantityFunc(10, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	esdtData := &esdt.ESDigitalToken{}
	esdtDataBytes, _ := marshaller.Marshal(esdtData)
	_ = userAcc.AccountDataHandler().SaveKeyValue([]byte(core.ElrondProtectedKeyPrefix+core.ESDTKeyIdentifier+"arg0"), esdtDataBytes)
	output, err := eqf.ProcessBuiltinFunction(
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

func TestEsdtNFTAddQuantity_ProcessBuiltinFunctionShouldErrOnSaveBecauseTokenIsPaused(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	globalSettingsHandler := &mock.GlobalSettingsHandlerStub{
		IsPausedCalled: func(_ []byte) bool {
			return true
		},
	}

	eqf, _ := NewESDTNFTAddQuantityFunc(10, createNewESDTDataStorageHandlerWithArgs(globalSettingsHandler, &mock.AccountsStub{}), globalSettingsHandler, &mock.ESDTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	esdtData := &esdt.ESDigitalToken{
		TokenMetaData: &esdt.MetaData{
			Name: []byte("test"),
		},
		Value: big.NewInt(10),
	}
	esdtDataBytes, _ := marshaller.Marshal(esdtData)
	_ = userAcc.AccountDataHandler().SaveKeyValue([]byte(core.ElrondProtectedKeyPrefix+core.ESDTKeyIdentifier+"arg0"+"arg1"), esdtDataBytes)

	output, err := eqf.ProcessBuiltinFunction(
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

func TestEsdtNFTAddQuantity_ProcessBuiltinFunctionShouldWork(t *testing.T) {
	t.Parallel()

	tokenIdentifier := "testTkn"
	key := baseESDTKeyPrefix + tokenIdentifier

	nonce := big.NewInt(33)
	initialValue := big.NewInt(5)
	valueToAdd := big.NewInt(37)
	expectedValue := big.NewInt(0).Add(initialValue, valueToAdd)

	marshaller := &mock.MarshalizerMock{}
	esdtRoleHandler := &mock.ESDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			assert.Equal(t, core.ESDTRoleNFTAddQuantity, string(action))
			return nil
		},
	}
	eqf, _ := NewESDTNFTAddQuantityFunc(10, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, esdtRoleHandler, 0, &mock.EpochNotifierStub{})

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	esdtData := &esdt.ESDigitalToken{
		TokenMetaData: &esdt.MetaData{
			Name: []byte("test"),
		},
		Value: initialValue,
	}
	esdtDataBytes, _ := marshaller.Marshal(esdtData)
	tokenKey := append([]byte(key), nonce.Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtDataBytes)

	output, err := eqf.ProcessBuiltinFunction(
		userAcc,
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte(tokenIdentifier), nonce.Bytes(), valueToAdd.Bytes()},
				CallerAddr:  []byte("address 1"),
				GasProvided: 12,
			},
			RecipientAddr: []byte("address 1"),
		},
	)

	require.NotNil(t, output)
	require.NoError(t, err)
	require.Equal(t, vmcommon.Ok, output.ReturnCode)

	res, err := userAcc.AccountDataHandler().RetrieveValue(tokenKey)
	require.NoError(t, err)
	require.NotNil(t, res)

	finalTokenData := esdt.ESDigitalToken{}
	_ = marshaller.Unmarshal(&finalTokenData, res)
	require.Equal(t, expectedValue.Bytes(), finalTokenData.Value.Bytes())
}
