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

func TestNewESDTNFTBurnFunc(t *testing.T) {
	t.Parallel()

	// nil marshaller
	ebf, err := NewESDTNFTBurnFunc(10, nil, nil, nil)
	require.True(t, check.IfNil(ebf))
	require.Equal(t, ErrNilESDTNFTStorageHandler, err)

	// nil pause handler
	ebf, err = NewESDTNFTBurnFunc(10, createNewESDTDataStorageHandler(), nil, nil)
	require.True(t, check.IfNil(ebf))
	require.Equal(t, ErrNilGlobalSettingsHandler, err)

	// nil roles handler
	ebf, err = NewESDTNFTBurnFunc(10, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, nil)
	require.True(t, check.IfNil(ebf))
	require.Equal(t, ErrNilRolesHandler, err)

	// should work
	ebf, err = NewESDTNFTBurnFunc(10, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{})
	require.False(t, check.IfNil(ebf))
	require.NoError(t, err)
}

func TestESDTNFTBurn_SetNewGasConfig_NilGasCost(t *testing.T) {
	t.Parallel()

	defaultGasCost := uint64(10)
	ebf, _ := NewESDTNFTBurnFunc(defaultGasCost, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{})

	ebf.SetNewGasConfig(nil)
	require.Equal(t, defaultGasCost, ebf.funcGasCost)
}

func TestEsdtNFTBurnFunc_SetNewGasConfig_ShouldWork(t *testing.T) {
	t.Parallel()

	defaultGasCost := uint64(10)
	newGasCost := uint64(37)
	ebf, _ := NewESDTNFTBurnFunc(defaultGasCost, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{})

	ebf.SetNewGasConfig(
		&vmcommon.GasCost{
			BuiltInCost: vmcommon.BuiltInCost{
				ESDTNFTBurn: newGasCost,
			},
		},
	)

	require.Equal(t, newGasCost, ebf.funcGasCost)
}

func TestEsdtNFTBurnFunc_ProcessBuiltinFunctionErrorOnCheckESDTNFTCreateBurnAddInput(t *testing.T) {
	t.Parallel()

	ebf, _ := NewESDTNFTBurnFunc(10, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{})

	// nil vm input
	output, err := ebf.ProcessBuiltinFunction(mock.NewAccountWrapMock([]byte("addr")), nil, nil)
	require.Nil(t, output)
	require.Equal(t, ErrNilVmInput, err)

	// vm input - value not zero
	output, err = ebf.ProcessBuiltinFunction(
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
	output, err = ebf.ProcessBuiltinFunction(
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
	output, err = ebf.ProcessBuiltinFunction(
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
	output, err = ebf.ProcessBuiltinFunction(
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
	output, err = ebf.ProcessBuiltinFunction(
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
	output, err = ebf.ProcessBuiltinFunction(
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

func TestEsdtNFTBurnFunc_ProcessBuiltinFunctionInvalidNumberOfArguments(t *testing.T) {
	t.Parallel()

	ebf, _ := NewESDTNFTBurnFunc(10, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{})
	output, err := ebf.ProcessBuiltinFunction(
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

func TestEsdtNFTBurnFunc_ProcessBuiltinFunctionCheckAllowedToExecuteError(t *testing.T) {
	t.Parallel()

	localErr := errors.New("err")
	rolesHandler := &mock.ESDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(_ vmcommon.UserAccountHandler, _ []byte, _ []byte) error {
			return localErr
		},
	}
	ebf, _ := NewESDTNFTBurnFunc(10, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, rolesHandler)
	output, err := ebf.ProcessBuiltinFunction(
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

func TestEsdtNFTBurnFunc_ProcessBuiltinFunctionNewSenderShouldErr(t *testing.T) {
	t.Parallel()

	ebf, _ := NewESDTNFTBurnFunc(10, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{})
	output, err := ebf.ProcessBuiltinFunction(
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

func TestEsdtNFTBurnFunc_ProcessBuiltinFunctionMetaDataMissing(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	ebf, _ := NewESDTNFTBurnFunc(10, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{})

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	esdtData := &esdt.ESDigitalToken{}
	esdtDataBytes, _ := marshaller.Marshal(esdtData)
	_ = userAcc.AccountDataHandler().SaveKeyValue([]byte(core.ProtectedKeyPrefix+core.ESDTKeyIdentifier+"arg0"), esdtDataBytes)
	output, err := ebf.ProcessBuiltinFunction(
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

func TestEsdtNFTBurnFunc_ProcessBuiltinFunctionInvalidBurnQuantity(t *testing.T) {
	t.Parallel()

	initialQuantity := big.NewInt(55)
	quantityToBurn := big.NewInt(75)

	marshaller := &mock.MarshalizerMock{}

	ebf, _ := NewESDTNFTBurnFunc(10, createNewESDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.ESDTRoleHandlerStub{})

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	esdtData := &esdt.ESDigitalToken{
		TokenMetaData: &esdt.MetaData{
			Name: []byte("test"),
		},
		Value: initialQuantity,
	}
	esdtDataBytes, _ := marshaller.Marshal(esdtData)
	_ = userAcc.AccountDataHandler().SaveKeyValue([]byte(core.ProtectedKeyPrefix+core.ESDTKeyIdentifier+"arg0"+"arg1"), esdtDataBytes)
	output, err := ebf.ProcessBuiltinFunction(
		userAcc,
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte("arg0"), []byte("arg1"), quantityToBurn.Bytes()},
				CallerAddr:  []byte("address 1"),
				GasProvided: 12,
			},
			RecipientAddr: []byte("address 1"),
		},
	)

	require.Nil(t, output)
	require.Equal(t, ErrInvalidNFTQuantity, err)
}

func TestEsdtNFTBurnFunc_ProcessBuiltinFunctionShouldErrOnSaveBecauseTokenIsPaused(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	globalSettingsHandler := &mock.GlobalSettingsHandlerStub{
		IsPausedCalled: func(_ []byte) bool {
			return true
		},
	}

	ebf, _ := NewESDTNFTBurnFunc(10, createNewESDTDataStorageHandlerWithArgs(globalSettingsHandler, &mock.AccountsStub{}, &mock.EnableEpochsHandlerStub{}, &mock.CrossChainTokenCheckerMock{}), globalSettingsHandler, &mock.ESDTRoleHandlerStub{})

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	esdtData := &esdt.ESDigitalToken{
		TokenMetaData: &esdt.MetaData{
			Name: []byte("test"),
		},
		Value: big.NewInt(10),
	}
	esdtDataBytes, _ := marshaller.Marshal(esdtData)
	_ = userAcc.AccountDataHandler().SaveKeyValue([]byte(core.ProtectedKeyPrefix+core.ESDTKeyIdentifier+"arg0"+"arg1"), esdtDataBytes)
	output, err := ebf.ProcessBuiltinFunction(
		userAcc,
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte("arg0"), []byte("arg1"), big.NewInt(5).Bytes()},
				CallerAddr:  []byte("address 1"),
				GasProvided: 12,
			},
			RecipientAddr: []byte("address 1"),
		},
	)

	require.Nil(t, output)
	require.Equal(t, ErrESDTTokenIsPaused, err)
}

func TestEsdtNFTBurnFunc_ProcessBuiltinFunctionShouldWork(t *testing.T) {
	t.Parallel()

	tokenIdentifier := "testTkn"
	key := baseESDTKeyPrefix + tokenIdentifier

	nonce := big.NewInt(33)
	initialQuantity := big.NewInt(100)
	quantityToBurn := big.NewInt(37)
	expectedQuantity := big.NewInt(0).Sub(initialQuantity, quantityToBurn)

	marshaller := &mock.MarshalizerMock{}
	esdtRoleHandler := &mock.ESDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			assert.Equal(t, core.ESDTRoleNFTBurn, string(action))
			return nil
		},
	}
	storageHandler := createNewESDTDataStorageHandler()
	ebf, _ := NewESDTNFTBurnFunc(10, storageHandler, &mock.GlobalSettingsHandlerStub{}, esdtRoleHandler)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	esdtData := &esdt.ESDigitalToken{
		TokenMetaData: &esdt.MetaData{
			Name: []byte("test"),
		},
		Value: initialQuantity,
	}
	esdtDataBytes, _ := marshaller.Marshal(esdtData)
	nftTokenKey := append([]byte(key), nonce.Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(nftTokenKey, esdtDataBytes)

	_ = storageHandler.saveESDTMetaDataToSystemAccount(userAcc, 0, nftTokenKey, nonce.Uint64(), esdtData, true)
	_ = storageHandler.AddToLiquiditySystemAcc([]byte(key), 0, nonce.Uint64(), initialQuantity, false)
	output, err := ebf.ProcessBuiltinFunction(
		userAcc,
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte(tokenIdentifier), nonce.Bytes(), quantityToBurn.Bytes()},
				CallerAddr:  []byte("address 1"),
				GasProvided: 12,
			},
			RecipientAddr: []byte("address 1"),
		},
	)

	require.NotNil(t, output)
	require.NoError(t, err)
	require.Equal(t, vmcommon.Ok, output.ReturnCode)

	res, _, err := userAcc.AccountDataHandler().RetrieveValue(nftTokenKey)
	require.NoError(t, err)
	require.NotNil(t, res)

	finalTokenData := esdt.ESDigitalToken{}
	_ = marshaller.Unmarshal(&finalTokenData, res)
	require.Equal(t, expectedQuantity.Bytes(), finalTokenData.Value.Bytes())
}

func TestEsdtNFTBurnFunc_ProcessBuiltinFunctionWithGlobalBurn(t *testing.T) {
	t.Parallel()

	tokenIdentifier := "testTkn"
	key := baseESDTKeyPrefix + tokenIdentifier

	nonce := big.NewInt(33)
	initialQuantity := big.NewInt(100)
	quantityToBurn := big.NewInt(37)
	expectedQuantity := big.NewInt(0).Sub(initialQuantity, quantityToBurn)

	marshaller := &mock.MarshalizerMock{}
	storageHandler := createNewESDTDataStorageHandler()
	ebf, _ := NewESDTNFTBurnFunc(10, storageHandler, &mock.GlobalSettingsHandlerStub{
		IsBurnForAllCalled: func(token []byte) bool {
			return true
		},
	}, &mock.ESDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			return errors.New("no burn allowed")
		},
	})

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	esdtData := &esdt.ESDigitalToken{
		TokenMetaData: &esdt.MetaData{
			Name: []byte("test"),
		},
		Value: initialQuantity,
	}
	esdtDataBytes, _ := marshaller.Marshal(esdtData)
	tokenKey := append([]byte(key), nonce.Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtDataBytes)
	_ = storageHandler.saveESDTMetaDataToSystemAccount(userAcc, 0, tokenKey, nonce.Uint64(), esdtData, true)
	_ = storageHandler.AddToLiquiditySystemAcc([]byte(key), 0, nonce.Uint64(), initialQuantity, false)

	output, err := ebf.ProcessBuiltinFunction(
		userAcc,
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte(tokenIdentifier), nonce.Bytes(), quantityToBurn.Bytes()},
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

	finalTokenData := esdt.ESDigitalToken{}
	_ = marshaller.Unmarshal(&finalTokenData, res)
	require.Equal(t, expectedQuantity.Bytes(), finalTokenData.Value.Bytes())
}
