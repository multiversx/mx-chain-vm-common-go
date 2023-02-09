package builtInFunctions

import (
	"bytes"
	"errors"
	"math/big"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data/esdt"
	"github.com/multiversx/mx-chain-core-go/data/vm"
	"github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createNftCreateWithStubArguments() *esdtNFTCreate {
	nftCreate, _ := NewESDTNFTCreateFunc(
		1,
		vmcommon.BaseOperationCost{},
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.ESDTRoleHandlerStub{},
		createNewESDTDataStorageHandler(),
		&mock.AccountsStub{},
		&mock.EnableEpochsHandlerStub{
			IsValueLengthCheckFlagEnabledField: true,
		},
	)

	return nftCreate
}

func TestNewESDTNFTCreateFunc_NilArgumentsShouldErr(t *testing.T) {
	t.Parallel()

	t.Run("nil marshaller should error", func(t *testing.T) {
		t.Parallel()

		nftCreate, err := NewESDTNFTCreateFunc(
			0,
			vmcommon.BaseOperationCost{},
			nil,
			&mock.GlobalSettingsHandlerStub{},
			&mock.ESDTRoleHandlerStub{},
			createNewESDTDataStorageHandler(),
			&mock.AccountsStub{},
			&mock.EnableEpochsHandlerStub{},
		)
		assert.True(t, check.IfNil(nftCreate))
		assert.Equal(t, ErrNilMarshalizer, err)
	})
	t.Run("nil global settings handler should error", func(t *testing.T) {
		t.Parallel()

		nftCreate, err := NewESDTNFTCreateFunc(
			0,
			vmcommon.BaseOperationCost{},
			&mock.MarshalizerMock{},
			nil,
			&mock.ESDTRoleHandlerStub{},
			createNewESDTDataStorageHandler(),
			&mock.AccountsStub{},
			&mock.EnableEpochsHandlerStub{},
		)
		assert.True(t, check.IfNil(nftCreate))
		assert.Equal(t, ErrNilGlobalSettingsHandler, err)
	})
	t.Run("nil roles handler should error", func(t *testing.T) {
		t.Parallel()

		nftCreate, err := NewESDTNFTCreateFunc(
			0,
			vmcommon.BaseOperationCost{},
			&mock.MarshalizerMock{},
			&mock.GlobalSettingsHandlerStub{},
			nil,
			createNewESDTDataStorageHandler(),
			&mock.AccountsStub{},
			&mock.EnableEpochsHandlerStub{},
		)
		assert.True(t, check.IfNil(nftCreate))
		assert.Equal(t, ErrNilRolesHandler, err)
	})
	t.Run("nil esdt storage handler should error", func(t *testing.T) {
		t.Parallel()

		nftCreate, err := NewESDTNFTCreateFunc(
			0,
			vmcommon.BaseOperationCost{},
			&mock.MarshalizerMock{},
			&mock.GlobalSettingsHandlerStub{},
			&mock.ESDTRoleHandlerStub{},
			nil,
			&mock.AccountsStub{},
			&mock.EnableEpochsHandlerStub{},
		)
		assert.True(t, check.IfNil(nftCreate))
		assert.Equal(t, ErrNilESDTNFTStorageHandler, err)
	})
	t.Run("nil enable epochs handler should error", func(t *testing.T) {
		t.Parallel()

		nftCreate, err := NewESDTNFTCreateFunc(
			0,
			vmcommon.BaseOperationCost{},
			&mock.MarshalizerMock{},
			&mock.GlobalSettingsHandlerStub{},
			&mock.ESDTRoleHandlerStub{},
			createNewESDTDataStorageHandler(),
			&mock.AccountsStub{},
			nil,
		)
		assert.True(t, check.IfNil(nftCreate))
		assert.Equal(t, ErrNilEnableEpochsHandler, err)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		nftCreate, err := NewESDTNFTCreateFunc(
			0,
			vmcommon.BaseOperationCost{},
			&mock.MarshalizerMock{},
			&mock.GlobalSettingsHandlerStub{},
			&mock.ESDTRoleHandlerStub{},
			createNewESDTDataStorageHandler(),
			&mock.AccountsStub{},
			&mock.EnableEpochsHandlerStub{},
		)
		assert.Nil(t, err)
		assert.False(t, check.IfNil(nftCreate))
	})
}

func TestNewESDTNFTCreateFunc(t *testing.T) {
	t.Parallel()

	nftCreate, err := NewESDTNFTCreateFunc(
		0,
		vmcommon.BaseOperationCost{},
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.ESDTRoleHandlerStub{},
		createNewESDTDataStorageHandler(),
		&mock.AccountsStub{},
		&mock.EnableEpochsHandlerStub{
			IsValueLengthCheckFlagEnabledField: true,
		},
	)
	assert.False(t, check.IfNil(nftCreate))
	assert.Nil(t, err)
}

func TestEsdtNFTCreate_SetNewGasConfig(t *testing.T) {
	t.Parallel()

	nftCreate := createNftCreateWithStubArguments()
	nftCreate.SetNewGasConfig(nil)
	assert.Equal(t, uint64(1), nftCreate.funcGasCost)
	assert.Equal(t, vmcommon.BaseOperationCost{}, nftCreate.gasConfig)

	gasCost := createMockGasCost()
	nftCreate.SetNewGasConfig(&gasCost)
	assert.Equal(t, gasCost.BuiltInCost.ESDTNFTCreate, nftCreate.funcGasCost)
	assert.Equal(t, gasCost.BaseOperationCost, nftCreate.gasConfig)
}

func TestEsdtNFTCreate_ProcessBuiltinFunctionInvalidArguments(t *testing.T) {
	t.Parallel()

	nftCreate := createNftCreateWithStubArguments()
	sender := mock.NewAccountWrapMock([]byte("address"))
	vmOutput, err := nftCreate.ProcessBuiltinFunction(sender, nil, nil)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrNilVmInput, err)

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: []byte("caller"),
			CallValue:  big.NewInt(0),
			Arguments:  [][]byte{[]byte("arg1"), []byte("arg2")},
		},
		RecipientAddr: []byte("recipient"),
	}
	vmOutput, err = nftCreate.ProcessBuiltinFunction(sender, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrInvalidRcvAddr, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: sender.AddressBytes(),
			CallValue:  big.NewInt(0),
			Arguments:  [][]byte{[]byte("arg1"), []byte("arg2")},
		},
		RecipientAddr: sender.AddressBytes(),
	}
	vmOutput, err = nftCreate.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrNilUserAccount, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: sender.AddressBytes(),
			CallValue:  big.NewInt(0),
			Arguments:  [][]byte{[]byte("arg1"), []byte("arg2")},
		},
		RecipientAddr: sender.AddressBytes(),
	}
	vmOutput, err = nftCreate.ProcessBuiltinFunction(sender, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrNotEnoughGas, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  sender.AddressBytes(),
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("arg1"), []byte("arg2")},
			GasProvided: 1,
		},
		RecipientAddr: sender.AddressBytes(),
	}
	vmOutput, err = nftCreate.ProcessBuiltinFunction(sender, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.True(t, errors.Is(err, ErrInvalidArguments))
}

func TestEsdtNFTCreate_ProcessBuiltinFunctionNotAllowedToExecute(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("expected error")
	esdtDataStorage := createNewESDTDataStorageHandler()
	nftCreate, _ := NewESDTNFTCreateFunc(
		0,
		vmcommon.BaseOperationCost{},
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.ESDTRoleHandlerStub{
			CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
				return expectedErr
			},
		},
		esdtDataStorage,
		esdtDataStorage.accounts,
		&mock.EnableEpochsHandlerStub{
			IsValueLengthCheckFlagEnabledField: true,
		},
	)
	sender := mock.NewAccountWrapMock([]byte("address"))
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: sender.AddressBytes(),
			CallValue:  big.NewInt(0),
			Arguments:  make([][]byte, 7),
		},
		RecipientAddr: sender.AddressBytes(),
	}
	vmOutput, err := nftCreate.ProcessBuiltinFunction(sender, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, expectedErr, err)
}

func TestEsdtNFTCreate_ProcessBuiltinFunctionShouldWork(t *testing.T) {
	t.Parallel()

	esdtDataStorage := createNewESDTDataStorageHandler()
	firstCheck := true
	esdtRoleHandler := &mock.ESDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			if firstCheck {
				assert.Equal(t, core.ESDTRoleNFTCreate, string(action))
				firstCheck = false
			} else {
				assert.Equal(t, core.ESDTRoleNFTAddQuantity, string(action))
			}
			return nil
		},
	}
	nftCreate, _ := NewESDTNFTCreateFunc(
		0,
		vmcommon.BaseOperationCost{},
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		esdtRoleHandler,
		esdtDataStorage,
		esdtDataStorage.accounts,
		&mock.EnableEpochsHandlerStub{
			IsValueLengthCheckFlagEnabledField: true,
		},
	)
	address := bytes.Repeat([]byte{1}, 32)
	sender := mock.NewUserAccount(address)
	//add some data in the trie, otherwise the creation will fail (it won't happen in real case usage as the create NFT
	//will be called after the creation permission was set in the account's data)
	_ = sender.AccountDataHandler().SaveKeyValue([]byte("key"), []byte("value"))

	token := "token"
	quantity := big.NewInt(2)
	name := "name"
	royalties := 100 //1%
	hash := []byte("12345678901234567890123456789012")
	attributes := []byte("attributes")
	uris := [][]byte{[]byte("uri1"), []byte("uri2")}
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: sender.AddressBytes(),
			CallValue:  big.NewInt(0),
			Arguments: [][]byte{
				[]byte(token),
				quantity.Bytes(),
				[]byte(name),
				big.NewInt(int64(royalties)).Bytes(),
				hash,
				attributes,
				uris[0],
				uris[1],
			},
		},
		RecipientAddr: sender.AddressBytes(),
	}
	vmOutput, err := nftCreate.ProcessBuiltinFunction(sender, nil, vmInput)
	assert.Nil(t, err)
	require.NotNil(t, vmOutput)

	createdEsdt, latestNonce := readNFTData(t, sender, nftCreate.marshaller, []byte(token), 1, address)
	assert.Equal(t, uint64(1), latestNonce)
	expectedEsdt := &esdt.ESDigitalToken{
		Type:  uint32(core.NonFungible),
		Value: quantity,
	}
	assert.Equal(t, expectedEsdt, createdEsdt)

	tokenMetaData := &esdt.MetaData{
		Nonce:      1,
		Name:       []byte(name),
		Creator:    address,
		Royalties:  uint32(royalties),
		Hash:       hash,
		URIs:       uris,
		Attributes: attributes,
	}

	tokenKey := []byte(baseESDTKeyPrefix + token)
	tokenKey = append(tokenKey, big.NewInt(1).Bytes()...)

	esdtData, _, _ := esdtDataStorage.getESDTDigitalTokenDataFromSystemAccount(tokenKey, defaultQueryOptions())
	assert.Equal(t, tokenMetaData, esdtData.TokenMetaData)
	assert.Equal(t, esdtData.Value, quantity)

	esdtDataBytes := vmOutput.Logs[0].Topics[3]
	var esdtDataFromLog esdt.ESDigitalToken
	_ = nftCreate.marshaller.Unmarshal(&esdtDataFromLog, esdtDataBytes)
	require.Equal(t, esdtData.TokenMetaData, esdtDataFromLog.TokenMetaData)
}

func TestEsdtNFTCreate_ProcessBuiltinFunctionWithExecByCaller(t *testing.T) {
	t.Parallel()

	accounts := createAccountsAdapterWithMap()
	enableEpochsHandler := &mock.EnableEpochsHandlerStub{
		IsValueLengthCheckFlagEnabledField:      true,
		IsSaveToSystemAccountFlagEnabledField:   true,
		IsCheckFrozenCollectionFlagEnabledField: true,
	}
	esdtDataStorage := createNewESDTDataStorageHandlerWithArgs(&mock.GlobalSettingsHandlerStub{}, accounts, enableEpochsHandler)
	nftCreate, _ := NewESDTNFTCreateFunc(
		0,
		vmcommon.BaseOperationCost{},
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.ESDTRoleHandlerStub{},
		esdtDataStorage,
		esdtDataStorage.accounts,
		enableEpochsHandler,
	)
	address := bytes.Repeat([]byte{1}, 32)
	userAddress := bytes.Repeat([]byte{2}, 32)
	token := "token"
	quantity := big.NewInt(2)
	name := "name"
	royalties := 100 //1%
	hash := []byte("12345678901234567890123456789012")
	attributes := []byte("attributes")
	uris := [][]byte{[]byte("uri1"), []byte("uri2")}
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: userAddress,
			CallValue:  big.NewInt(0),
			Arguments: [][]byte{
				[]byte(token),
				quantity.Bytes(),
				[]byte(name),
				big.NewInt(int64(royalties)).Bytes(),
				hash,
				attributes,
				uris[0],
				uris[1],
				address,
			},
			CallType: vm.ExecOnDestByCaller,
		},
		RecipientAddr: userAddress,
	}
	vmOutput, err := nftCreate.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, err)
	require.NotNil(t, vmOutput)

	roleAcc, _ := nftCreate.getAccount(address)

	createdEsdt, latestNonce := readNFTData(t, roleAcc, nftCreate.marshaller, []byte(token), 1, address)
	assert.Equal(t, uint64(1), latestNonce)
	expectedEsdt := &esdt.ESDigitalToken{
		Type:  uint32(core.NonFungible),
		Value: quantity,
	}
	assert.Equal(t, expectedEsdt, createdEsdt)

	tokenMetaData := &esdt.MetaData{
		Nonce:      1,
		Name:       []byte(name),
		Creator:    userAddress,
		Royalties:  uint32(royalties),
		Hash:       hash,
		URIs:       uris,
		Attributes: attributes,
	}

	tokenKey := []byte(baseESDTKeyPrefix + token)
	tokenKey = append(tokenKey, big.NewInt(1).Bytes()...)

	metaData, _ := esdtDataStorage.getESDTMetaDataFromSystemAccount(tokenKey, defaultQueryOptions())
	assert.Equal(t, tokenMetaData, metaData)
}

func readNFTData(t *testing.T, account vmcommon.UserAccountHandler, marshaller vmcommon.Marshalizer, tokenID []byte, nonce uint64, _ []byte) (*esdt.ESDigitalToken, uint64) {
	nonceKey := getNonceKey(tokenID)
	latestNonceBytes, _, err := account.(vmcommon.UserAccountHandler).AccountDataHandler().RetrieveValue(nonceKey)
	require.Nil(t, err)
	latestNonce := big.NewInt(0).SetBytes(latestNonceBytes).Uint64()

	createdTokenID := []byte(baseESDTKeyPrefix)
	createdTokenID = append(createdTokenID, tokenID...)
	tokenKey := computeESDTNFTTokenKey(createdTokenID, nonce)
	data, _, err := account.(vmcommon.UserAccountHandler).AccountDataHandler().RetrieveValue(tokenKey)
	require.Nil(t, err)

	esdtData := &esdt.ESDigitalToken{}
	err = marshaller.Unmarshal(esdtData, data)
	require.Nil(t, err)

	return esdtData, latestNonce
}
