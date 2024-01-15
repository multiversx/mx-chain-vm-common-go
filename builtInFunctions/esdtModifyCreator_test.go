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

func TestNewESDTModifyCreatorFunc(t *testing.T) {
	t.Parallel()

	t.Run("nil accounts adapter", func(t *testing.T) {
		t.Parallel()

		e, err := NewESDTModifyCreatorFunc(0, nil, nil, nil, nil, nil)
		assert.Nil(t, e)
		assert.Equal(t, ErrNilAccountsAdapter, err)
	})
	t.Run("nil global settings handler", func(t *testing.T) {
		t.Parallel()

		e, err := NewESDTModifyCreatorFunc(0, &mock.AccountsStub{}, nil, nil, nil, nil)
		assert.Nil(t, e)
		assert.Equal(t, ErrNilGlobalSettingsHandler, err)
	})
	t.Run("nil enable epochs handler", func(t *testing.T) {
		t.Parallel()

		e, err := NewESDTModifyCreatorFunc(0, &mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, nil, nil, nil)
		assert.Nil(t, e)
		assert.Equal(t, ErrNilEnableEpochsHandler, err)
	})
	t.Run("nil storage handler", func(t *testing.T) {
		t.Parallel()

		e, err := NewESDTModifyCreatorFunc(0, &mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, nil, nil, &mock.EnableEpochsHandlerStub{})
		assert.Nil(t, e)
		assert.Equal(t, ErrNilESDTNFTStorageHandler, err)
	})
	t.Run("nil roles handler", func(t *testing.T) {
		t.Parallel()

		e, err := NewESDTModifyCreatorFunc(0, &mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, &mock.ESDTNFTStorageHandlerStub{}, nil, &mock.EnableEpochsHandlerStub{})
		assert.Nil(t, e)
		assert.Equal(t, ErrNilRolesHandler, err)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		funcGasCost := uint64(10)
		e, err := NewESDTModifyCreatorFunc(funcGasCost, &mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, &mock.ESDTNFTStorageHandlerStub{}, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{})
		assert.NotNil(t, e)
		assert.Nil(t, err)
		assert.Equal(t, funcGasCost, e.funcGasCost)
	})
}

func TestESDTModifyCreator_ProcessBuiltinFunction(t *testing.T) {
	t.Parallel()

	t.Run("nil vmInput", func(t *testing.T) {
		t.Parallel()

		e, _ := NewESDTModifyCreatorFunc(0, &mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, &mock.ESDTNFTStorageHandlerStub{}, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{})
		vmOutput, err := e.ProcessBuiltinFunction(nil, nil, nil)
		assert.Nil(t, vmOutput)
		assert.Equal(t, ErrNilVmInput, err)
	})
	t.Run("nil CallValue", func(t *testing.T) {
		t.Parallel()

		e, _ := NewESDTModifyCreatorFunc(0, &mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, &mock.ESDTNFTStorageHandlerStub{}, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{})
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

		e, _ := NewESDTModifyCreatorFunc(0, &mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, &mock.ESDTNFTStorageHandlerStub{}, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{})
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

		e, _ := NewESDTModifyCreatorFunc(0, &mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, &mock.ESDTNFTStorageHandlerStub{}, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{})
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

		e, _ := NewESDTModifyCreatorFunc(0, &mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, &mock.ESDTNFTStorageHandlerStub{}, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{})
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
		e, _ := NewESDTModifyCreatorFunc(0, &mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, &mock.ESDTNFTStorageHandlerStub{}, &mock.ESDTRoleHandlerStub{}, enableEpochsHandler)
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
		e, _ := NewESDTModifyCreatorFunc(0, &mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, &mock.ESDTNFTStorageHandlerStub{}, &mock.ESDTRoleHandlerStub{}, enableEpochsHandler)
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
		e, _ := NewESDTModifyCreatorFunc(0, &mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, &mock.ESDTNFTStorageHandlerStub{}, rolesHandler, enableEpochsHandler)
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
	t.Run("sets the creator as the caller", func(t *testing.T) {
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
		oldMetaData := &esdt.MetaData{
			Nonce:      nonce,
			Name:       []byte("name"),
			Creator:    []byte("creator"),
			Royalties:  10,
			Hash:       []byte("hash"),
			URIs:       [][]byte{[]byte("uri")},
			Attributes: []byte("attributes"),
		}
		storageHandler := &mock.ESDTNFTStorageHandlerStub{
			GetMetaDataFromSystemAccountCalled: func(bytes []byte, u uint64) (*esdt.MetaData, error) {
				getMetaDataFromSystemAccountCalled = true
				return oldMetaData, nil
			},
			SaveMetaDataToSystemAccountCalled: func(tokenKey []byte, n uint64, esdtData *esdt.ESDigitalToken) error {
				assert.Equal(t, esdtTokenKey, tokenKey)
				assert.Equal(t, nonce, n)
				assert.Equal(t, oldMetaData.Name, esdtData.TokenMetaData.Name)
				assert.Equal(t, oldMetaData.URIs, esdtData.TokenMetaData.URIs)
				assert.Equal(t, oldMetaData.Royalties, esdtData.TokenMetaData.Royalties)
				assert.Equal(t, oldMetaData.Hash, esdtData.TokenMetaData.Hash)
				assert.Equal(t, oldMetaData.Attributes, esdtData.TokenMetaData.Attributes)
				assert.Equal(t, []byte("caller"), esdtData.TokenMetaData.Creator)
				saveMetaDataToSystemAccountCalled = true
				return nil
			},
		}
		e, _ := NewESDTModifyCreatorFunc(101, accounts, globalSettingsHandler, storageHandler, &mock.ESDTRoleHandlerStub{}, enableEpochsHandler)
		_ = e.SetBlockDataHandler(&mock.BlockDataHandlerStub{})

		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				CallerAddr:  []byte("caller"),
				GasProvided: 1000,
				Arguments:   [][]byte{tokenId, {15}},
			},
			RecipientAddr: []byte("caller"),
		}

		vmOutput, err := e.ProcessBuiltinFunction(mock.NewUserAccount([]byte("addr")), nil, vmInput)
		assert.Nil(t, err)
		assert.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)
		assert.Equal(t, uint64(899), vmOutput.GasRemaining)
		assert.True(t, saveMetaDataToSystemAccountCalled)
		assert.True(t, getMetaDataFromSystemAccountCalled)
	})
}

func TestESDTModifyCreator_SetNewGasConfig(t *testing.T) {
	t.Parallel()

	e, _ := NewESDTModifyCreatorFunc(0, &mock.AccountsStub{}, &mock.GlobalSettingsHandlerStub{}, &mock.ESDTNFTStorageHandlerStub{}, &mock.ESDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{})

	newGasCost := &vmcommon.GasCost{
		BuiltInCost: vmcommon.BuiltInCost{
			ESDTNFTUpdate: 10,
		},
	}
	e.SetNewGasConfig(newGasCost)

	assert.Equal(t, newGasCost.BuiltInCost.ESDTModifyCreator, e.funcGasCost)
}
