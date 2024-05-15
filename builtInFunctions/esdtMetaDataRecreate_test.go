package builtInFunctions

import (
	"errors"
	"math/big"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/data/esdt"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/assert"
)

func TestNewESDTMetaDataRecreateFunc(t *testing.T) {
	t.Parallel()

	t.Run("nil accounts adapter", func(t *testing.T) {
		t.Parallel()

		e, err := NewESDTMetaDataRecreateFunc(0, vmcommon.BaseOperationCost{}, nil, nil, nil, nil, nil)
		assert.Nil(t, e)
		assert.Equal(t, ErrNilAccountsAdapter, err)
	})
	t.Run("nil global settings handler", func(t *testing.T) {
		t.Parallel()

		e, err := NewESDTMetaDataRecreateFunc(0, vmcommon.BaseOperationCost{}, &mock.AccountsStub{}, nil, nil, nil, nil)
		assert.Nil(t, e)
		assert.Equal(t, ErrNilGlobalSettingsHandler, err)
	})
	t.Run("nil enable epochs handler", func(t *testing.T) {
		t.Parallel()

		e, err := NewESDTMetaDataRecreateFunc(0, vmcommon.BaseOperationCost{}, &mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, nil, nil, nil)
		assert.Nil(t, e)
		assert.Equal(t, ErrNilEnableEpochsHandler, err)
	})
	t.Run("nil storage handler", func(t *testing.T) {
		t.Parallel()

		e, err := NewESDTMetaDataRecreateFunc(0, vmcommon.BaseOperationCost{}, &mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, nil, nil, &mock.EnableEpochsHandlerStub{})
		assert.Nil(t, e)
		assert.Equal(t, ErrNilESDTNFTStorageHandler, err)
	})
	t.Run("nil roles handler", func(t *testing.T) {
		t.Parallel()

		e, err := NewESDTMetaDataRecreateFunc(0, vmcommon.BaseOperationCost{}, &mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, &mock.ESDTNFTStorageHandlerStub{}, nil, &mock.EnableEpochsHandlerStub{})
		assert.Nil(t, e)
		assert.Equal(t, ErrNilRolesHandler, err)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		funcGasCost := uint64(10)
		e, err := NewESDTMetaDataRecreateFunc(funcGasCost, vmcommon.BaseOperationCost{}, &mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, &mock.ESDTNFTStorageHandlerStub{}, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{})
		assert.NotNil(t, e)
		assert.Nil(t, err)
		assert.Equal(t, funcGasCost, e.funcGasCost)
	})
}

func TestESDTMetaDataRecreate_ProcessBuiltinFunction(t *testing.T) {
	t.Parallel()

	t.Run("nil vmInput", func(t *testing.T) {
		t.Parallel()

		e, _ := NewESDTMetaDataRecreateFunc(0, vmcommon.BaseOperationCost{}, &mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, &mock.ESDTNFTStorageHandlerStub{}, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{})
		vmOutput, err := e.ProcessBuiltinFunction(nil, nil, nil)
		assert.Nil(t, vmOutput)
		assert.Equal(t, ErrNilVmInput, err)
	})
	t.Run("nil CallValue", func(t *testing.T) {
		t.Parallel()

		e, _ := NewESDTMetaDataRecreateFunc(0, vmcommon.BaseOperationCost{}, &mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, &mock.ESDTNFTStorageHandlerStub{}, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{})
		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue: nil,
			},
		}
		vmOutput, err := e.ProcessBuiltinFunction(nil, nil, vmInput)
		assert.Nil(t, vmOutput)
		assert.Equal(t, ErrNilValue, err)
	})
	t.Run("call value not zero", func(t *testing.T) {
		t.Parallel()

		e, _ := NewESDTMetaDataRecreateFunc(0, vmcommon.BaseOperationCost{}, &mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, &mock.ESDTNFTStorageHandlerStub{}, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{})
		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue: big.NewInt(10),
			},
		}
		vmOutput, err := e.ProcessBuiltinFunction(nil, nil, vmInput)
		assert.Nil(t, vmOutput)
		assert.Equal(t, ErrBuiltInFunctionCalledWithValue, err)
	})
	t.Run("recipient address is not caller address", func(t *testing.T) {
		t.Parallel()

		e, _ := NewESDTMetaDataRecreateFunc(0, vmcommon.BaseOperationCost{}, &mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, &mock.ESDTNFTStorageHandlerStub{}, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{})
		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:  big.NewInt(0),
				CallerAddr: []byte("caller"),
			},
			RecipientAddr: []byte("recipient"),
		}
		vmOutput, err := e.ProcessBuiltinFunction(nil, nil, vmInput)
		assert.Nil(t, vmOutput)
		assert.Equal(t, ErrInvalidRcvAddr, err)
	})
	t.Run("nil sender account", func(t *testing.T) {
		t.Parallel()

		e, _ := NewESDTMetaDataRecreateFunc(0, vmcommon.BaseOperationCost{}, &mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, &mock.ESDTNFTStorageHandlerStub{}, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{})
		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:  big.NewInt(0),
				CallerAddr: []byte("caller"),
			},
			RecipientAddr: []byte("caller"),
		}
		vmOutput, err := e.ProcessBuiltinFunction(nil, nil, vmInput)
		assert.Nil(t, vmOutput)
		assert.Equal(t, ErrNilUserAccount, err)
	})
	t.Run("built-in function is not active", func(t *testing.T) {
		t.Parallel()

		enableEpochsHandler := &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return false
			},
		}
		e, _ := NewESDTMetaDataRecreateFunc(0, vmcommon.BaseOperationCost{}, &mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, &mock.ESDTNFTStorageHandlerStub{}, &mock.ESDTRoleHandlerStub{}, enableEpochsHandler)
		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:  big.NewInt(0),
				CallerAddr: []byte("caller"),
			},
			RecipientAddr: []byte("caller"),
		}
		vmOutput, err := e.ProcessBuiltinFunction(mock.NewUserAccount([]byte("addr")), nil, vmInput)
		assert.Nil(t, vmOutput)
		assert.Equal(t, ErrBuiltInFunctionIsNotActive, err)
	})
	t.Run("invalid number of arguments", func(t *testing.T) {
		t.Parallel()

		enableEpochsHandler := &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return true
			},
		}
		e, _ := NewESDTMetaDataRecreateFunc(0, vmcommon.BaseOperationCost{}, &mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, &mock.ESDTNFTStorageHandlerStub{}, &mock.ESDTRoleHandlerStub{}, enableEpochsHandler)
		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:  big.NewInt(0),
				CallerAddr: []byte("caller"),
				Arguments:  [][]byte{},
			},
			RecipientAddr: []byte("caller"),
		}
		vmOutput, err := e.ProcessBuiltinFunction(mock.NewUserAccount([]byte("addr")), nil, vmInput)
		assert.Nil(t, vmOutput)
		assert.Equal(t, ErrInvalidNumberOfArguments, err)
	})
	t.Run("check allowed to execute failed", func(t *testing.T) {
		t.Parallel()

		allowedToExecuteCalled := false
		expectedErr := errors.New("expected error")
		enableEpochsHandler := &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return true
			},
		}
		rolesHandler := &mock.ESDTRoleHandlerStub{
			CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, role []byte) error {
				allowedToExecuteCalled = true
				return expectedErr
			},
		}
		e, _ := NewESDTMetaDataRecreateFunc(0, vmcommon.BaseOperationCost{}, &mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, &mock.ESDTNFTStorageHandlerStub{}, rolesHandler, enableEpochsHandler)
		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:  big.NewInt(0),
				CallerAddr: []byte("caller"),
				Arguments:  [][]byte{[]byte("tokenID"), {}, {}, {}, {}, {}, {}},
			},
			RecipientAddr: []byte("caller"),
		}
		vmOutput, err := e.ProcessBuiltinFunction(mock.NewUserAccount([]byte("addr")), nil, vmInput)
		assert.Nil(t, vmOutput)
		assert.Equal(t, expectedErr, err)
		assert.True(t, allowedToExecuteCalled)
	})
	t.Run("recreate dynamic esdt data", func(t *testing.T) {
		t.Parallel()

		getMetaDataFromSystemAccountCalled := false
		saveMetaDataToSystemAccountCalled := false
		tokenId := []byte("tokenID")
		esdtTokenKey := append([]byte(baseESDTKeyPrefix), tokenId...)
		nonce := uint64(15)

		enableEpochsHandler := &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return true
			},
		}
		globalSettingsHandler := &mock.GlobalSettingsHandlerStub{
			GetTokenTypeCalled: func(key []byte) (uint32, error) {
				assert.Equal(t, esdtTokenKey, key)
				return uint32(core.DynamicNFT), nil
			},
		}
		accounts := &mock.AccountsStub{}
		newMetadata := &esdt.MetaData{
			Nonce:      nonce,
			Name:       []byte("name"),
			Creator:    []byte("caller"),
			Royalties:  50,
			Hash:       []byte("hash"),
			URIs:       [][]byte{[]byte("uri1"), []byte("uri2")},
			Attributes: []byte("attributes"),
		}
		storageHandler := &mock.ESDTNFTStorageHandlerStub{
			GetMetaDataFromSystemAccountCalled: func(bytes []byte, u uint64) (*esdt.MetaData, error) {
				getMetaDataFromSystemAccountCalled = true
				return &esdt.MetaData{Nonce: nonce}, nil
			},
			SaveMetaDataToSystemAccountCalled: func(tokenKey []byte, n uint64, esdtData *esdt.ESDigitalToken) error {
				assert.Equal(t, esdtTokenKey, tokenKey)
				assert.Equal(t, nonce, n)
				assert.Equal(t, newMetadata, esdtData.TokenMetaData)
				saveMetaDataToSystemAccountCalled = true
				return nil
			},
		}
		e, _ := NewESDTMetaDataRecreateFunc(101, vmcommon.BaseOperationCost{StorePerByte: 1}, accounts, globalSettingsHandler, storageHandler, &mock.ESDTRoleHandlerStub{}, enableEpochsHandler)

		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				CallerAddr:  []byte("caller"),
				GasProvided: 1000,
				Arguments:   [][]byte{tokenId, {15}, newMetadata.Name, {50}, newMetadata.Hash, newMetadata.Attributes, newMetadata.URIs[0], newMetadata.URIs[1]},
			},
			RecipientAddr: []byte("caller"),
		}

		vmOutput, err := e.ProcessBuiltinFunction(mock.NewUserAccount([]byte("addr")), nil, vmInput)
		assert.Nil(t, err)
		assert.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)
		assert.Equal(t, uint64(866), vmOutput.GasRemaining)
		assert.True(t, saveMetaDataToSystemAccountCalled)
		assert.True(t, getMetaDataFromSystemAccountCalled)
	})

	t.Run("recreate non dynamic esdt data", func(t *testing.T) {
		t.Parallel()

		getESDTNFTTokenOnSenderCalled := false
		saveESDTNFTTokenCalled := false
		tokenId := []byte("tokenID")
		esdtTokenKey := append([]byte(baseESDTKeyPrefix), tokenId...)
		nonce := uint64(15)

		enableEpochsHandler := &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return true
			},
		}
		globalSettingsHandler := &mock.GlobalSettingsHandlerStub{
			GetTokenTypeCalled: func(key []byte) (uint32, error) {
				assert.Equal(t, esdtTokenKey, key)
				return uint32(core.NonFungible), nil
			},
		}
		accounts := &mock.AccountsStub{}
		newMetadata := &esdt.MetaData{
			Nonce:      nonce,
			Name:       []byte("name"),
			Creator:    []byte("caller"),
			Royalties:  50,
			Hash:       []byte("hash"),
			URIs:       [][]byte{[]byte("uri1"), []byte("uri2")},
			Attributes: []byte("attributes"),
		}
		storageHandler := &mock.ESDTNFTStorageHandlerStub{
			GetESDTNFTTokenOnSenderCalled: func(acnt vmcommon.UserAccountHandler, esdtTokenKey []byte, nonce uint64) (*esdt.ESDigitalToken, error) {
				getESDTNFTTokenOnSenderCalled = true
				return &esdt.ESDigitalToken{
					Value:         big.NewInt(1),
					TokenMetaData: &esdt.MetaData{Nonce: nonce},
				}, nil
			},
			SaveESDTNFTTokenCalled: func(senderAddress []byte, acnt vmcommon.UserAccountHandler, esdtTokenKey []byte, nonce uint64, esdtData *esdt.ESDigitalToken, mustUpdateAllFields bool, isReturnWithError bool) ([]byte, error) {
				assert.Equal(t, newMetadata, esdtData.TokenMetaData)
				saveESDTNFTTokenCalled = true
				return nil, nil
			},
		}
		e, _ := NewESDTMetaDataRecreateFunc(101, vmcommon.BaseOperationCost{StorePerByte: 1}, accounts, globalSettingsHandler, storageHandler, &mock.ESDTRoleHandlerStub{}, enableEpochsHandler)

		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				CallerAddr:  []byte("caller"),
				GasProvided: 1000,
				Arguments:   [][]byte{tokenId, {15}, newMetadata.Name, {50}, newMetadata.Hash, newMetadata.Attributes, newMetadata.URIs[0], newMetadata.URIs[1]},
			},
			RecipientAddr: []byte("caller"),
		}

		vmOutput, err := e.ProcessBuiltinFunction(mock.NewUserAccount([]byte("addr")), nil, vmInput)
		assert.Nil(t, err)
		assert.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)
		assert.Equal(t, uint64(866), vmOutput.GasRemaining)
		assert.True(t, saveESDTNFTTokenCalled)
		assert.True(t, getESDTNFTTokenOnSenderCalled)
	})
}

func TestEsdtMetaDataRecreate_SetNewGasConfig(t *testing.T) {
	t.Parallel()

	e, _ := NewESDTMetaDataRecreateFunc(0, vmcommon.BaseOperationCost{}, &mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, &mock.ESDTNFTStorageHandlerStub{}, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{})

	newGasCost := &vmcommon.GasCost{
		BaseOperationCost: vmcommon.BaseOperationCost{
			StorePerByte: 15,
		},
		BuiltInCost: vmcommon.BuiltInCost{
			ESDTNFTRecreate: 10,
		},
	}
	e.SetNewGasConfig(newGasCost)

	assert.Equal(t, newGasCost.BuiltInCost.ESDTNFTRecreate, e.funcGasCost)
	assert.Equal(t, newGasCost.BaseOperationCost.StorePerByte, e.gasConfig.StorePerByte)
}

func TestEsdtMetaDataRecreate_changeEsdtVersion(t *testing.T) {
	t.Parallel()

	t.Run("flag disabled does nothing", func(t *testing.T) {
		t.Parallel()

		enableEpochsHandler := &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return false
			},
		}
		esdtData := &esdt.ESDigitalToken{}
		err := changeEsdtVersion(esdtData, 1, enableEpochsHandler)
		assert.Nil(t, err)
		assert.Nil(t, esdtData.Reserved)
	})
	t.Run("current version is higher than new version should err", func(t *testing.T) {
		t.Parallel()

		enableEpochsHandler := &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return true
			},
		}
		esdtData := &esdt.ESDigitalToken{
			Reserved: []byte{2},
			TokenMetaData: &esdt.MetaData{
				Name: []byte("name"),
			},
		}
		err := changeEsdtVersion(esdtData, 1, enableEpochsHandler)
		assert.True(t, errors.Is(err, ErrInvalidVersion))
	})
	t.Run("current version is lower than new version should change version", func(t *testing.T) {
		t.Parallel()

		enableEpochsHandler := &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return true
			},
		}
		esdtData := &esdt.ESDigitalToken{
			Reserved: []byte{1},
			TokenMetaData: &esdt.MetaData{
				Name: []byte("name"),
			},
		}

		err := changeEsdtVersion(esdtData, 2, enableEpochsHandler)
		assert.Nil(t, err)
		assert.Equal(t, []byte{2}, esdtData.Reserved)
	})
}