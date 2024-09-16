package builtInFunctions

import (
	"bytes"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data/esdt"
	"github.com/multiversx/mx-chain-core-go/data/vm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
)

func createESDTNFTCreateArgs() ESDTNFTCreateFuncArgs {
	return ESDTNFTCreateFuncArgs{
		FuncGasCost:  0,
		Marshaller:   &mock.MarshalizerMock{},
		RolesHandler: &mock.ESDTRoleHandlerStub{},
		EnableEpochsHandler: &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return flag == ValueLengthCheckFlag
			},
		},
		EsdtStorageHandler:            createNewESDTDataStorageHandler(),
		Accounts:                      &mock.AccountsStub{},
		GasConfig:                     vmcommon.BaseOperationCost{},
		GlobalSettingsHandler:         &mock.GlobalSettingsHandlerStub{},
		CrossChainTokenCheckerHandler: &mock.CrossChainTokenCheckerMock{},
	}
}

func createNftCreateWithStubArguments() *esdtNFTCreate {
	args := createESDTNFTCreateArgs()
	args.FuncGasCost = 1
	nftCreate, _ := NewESDTNFTCreateFunc(args)
	return nftCreate
}

func TestNewESDTNFTCreateFunc_NilArgumentsShouldErr(t *testing.T) {
	t.Parallel()

	t.Run("nil marshaller should error", func(t *testing.T) {
		t.Parallel()

		args := createESDTNFTCreateArgs()
		args.Marshaller = nil
		nftCreate, err := NewESDTNFTCreateFunc(args)
		assert.True(t, check.IfNil(nftCreate))
		assert.Equal(t, ErrNilMarshalizer, err)
	})
	t.Run("nil global settings handler should error", func(t *testing.T) {
		t.Parallel()

		args := createESDTNFTCreateArgs()
		args.GlobalSettingsHandler = nil
		nftCreate, err := NewESDTNFTCreateFunc(args)
		assert.True(t, check.IfNil(nftCreate))
		assert.Equal(t, ErrNilGlobalSettingsHandler, err)
	})
	t.Run("nil roles handler should error", func(t *testing.T) {
		t.Parallel()

		args := createESDTNFTCreateArgs()
		args.RolesHandler = nil
		nftCreate, err := NewESDTNFTCreateFunc(args)
		assert.True(t, check.IfNil(nftCreate))
		assert.Equal(t, ErrNilRolesHandler, err)
	})
	t.Run("nil esdt storage handler should error", func(t *testing.T) {
		t.Parallel()

		args := createESDTNFTCreateArgs()
		args.EsdtStorageHandler = nil
		nftCreate, err := NewESDTNFTCreateFunc(args)
		assert.True(t, check.IfNil(nftCreate))
		assert.Equal(t, ErrNilESDTNFTStorageHandler, err)
	})
	t.Run("nil enable epochs handler should error", func(t *testing.T) {
		t.Parallel()

		args := createESDTNFTCreateArgs()
		args.EnableEpochsHandler = nil
		nftCreate, err := NewESDTNFTCreateFunc(args)
		assert.True(t, check.IfNil(nftCreate))
		assert.Equal(t, ErrNilEnableEpochsHandler, err)
	})
	t.Run("nil cross chain token checker should error", func(t *testing.T) {
		t.Parallel()

		args := createESDTNFTCreateArgs()
		args.CrossChainTokenCheckerHandler = nil
		nftCreate, err := NewESDTNFTCreateFunc(args)
		assert.True(t, check.IfNil(nftCreate))
		assert.Equal(t, ErrNilCrossChainTokenChecker, err)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		args := createESDTNFTCreateArgs()
		nftCreate, err := NewESDTNFTCreateFunc(args)
		assert.Nil(t, err)
		assert.False(t, check.IfNil(nftCreate))
	})
}

func TestNewESDTNFTCreateFunc(t *testing.T) {
	t.Parallel()

	nftCreate, err := NewESDTNFTCreateFunc(createESDTNFTCreateArgs())
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
	esdtDtaStorage := createNewESDTDataStorageHandler()

	args := createESDTNFTCreateArgs()
	args.EsdtStorageHandler = esdtDtaStorage
	args.Accounts = esdtDtaStorage.accounts
	args.RolesHandler = &mock.ESDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			return expectedErr
		},
	}
	nftCreate, _ := NewESDTNFTCreateFunc(args)
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

	esdtDtaStorage := createNewESDTDataStorageHandler()
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
	args := createESDTNFTCreateArgs()
	args.RolesHandler = esdtRoleHandler
	args.Accounts = esdtDtaStorage.accounts
	args.EsdtStorageHandler = esdtDtaStorage

	nftCreate, _ := NewESDTNFTCreateFunc(args)
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

	esdtData, _, _ := esdtDtaStorage.getESDTDigitalTokenDataFromSystemAccount(tokenKey, defaultQueryOptions())
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
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ValueLengthCheckFlag || flag == SaveToSystemAccountFlag || flag == CheckFrozenCollectionFlag
		},
	}
	esdtDtaStorage := createNewESDTDataStorageHandlerWithArgs(&mock.GlobalSettingsHandlerStub{}, accounts, enableEpochsHandler, &mock.CrossChainTokenCheckerMock{})

	args := createESDTNFTCreateArgs()
	args.EnableEpochsHandler = enableEpochsHandler
	args.Accounts = esdtDtaStorage.accounts
	args.EsdtStorageHandler = esdtDtaStorage

	nftCreate, _ := NewESDTNFTCreateFunc(args)
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

	metaData, _ := esdtDtaStorage.getESDTMetaDataFromSystemAccount(tokenKey, defaultQueryOptions())
	assert.Equal(t, tokenMetaData, metaData)
}

func TestEsdtNFTCreate_ProcessBuiltinFunctionWithExecByCallerCrossChainToken(t *testing.T) {
	t.Parallel()

	accounts := createAccountsAdapterWithMap()
	enableEpochsHandler := &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ValueLengthCheckFlag || flag == SaveToSystemAccountFlag || flag == SendAlwaysFlag || flag == DynamicEsdtFlag
		},
	}
	crossChainTokenHandler := &mock.CrossChainTokenCheckerMock{
		IsCrossChainOperationCalled: func(tokenID []byte) bool {
			return true
		},
	}
	ctc, _ := NewCrossChainTokenChecker(nil, getWhiteListedAddress())
	esdtRoleHandler, _ := NewESDTRolesFunc(marshallerMock, ctc, false)
	esdtDtaStorage := createNewESDTDataStorageHandlerWithArgs(&mock.GlobalSettingsHandlerStub{}, accounts, enableEpochsHandler, crossChainTokenHandler)

	args := createESDTNFTCreateArgs()
	args.CrossChainTokenCheckerHandler = ctc
	args.EnableEpochsHandler = enableEpochsHandler
	args.Accounts = esdtDtaStorage.accounts
	args.EsdtStorageHandler = esdtDtaStorage
	args.RolesHandler = esdtRoleHandler

	nftCreate, _ := NewESDTNFTCreateFunc(args)
	whiteListedAddr := []byte("whiteListedAddress")
	whiteListedAcc := mock.NewUserAccount(whiteListedAddr)
	userAddr := []byte("userAccountAddress")
	token := "sov1-TOKEN-abcdef"
	tokenType := core.NonFungibleV2
	nonce := big.NewInt(1234)
	quantity := big.NewInt(1)
	name := "name"
	royalties := 100 //1%
	hash := []byte("12345678901234567890123456789012")
	attributes := []byte("attributes")
	uris := [][]byte{[]byte("uri1"), []byte("uri2")}
	originalCreator := []byte("originalCreator")
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: userAddr,
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
				big.NewInt(int64(tokenType)).Bytes(),
				nonce.Bytes(),
				originalCreator,
				whiteListedAcc.AddressBytes(),
			},
			CallType: vm.ExecOnDestByCaller,
		},
		RecipientAddr: userAddr,
	}
	vmOutput, err := nftCreate.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, err)
	require.NotNil(t, vmOutput)

	// Nonce was not saved in account
	nonceKey := getNonceKey([]byte(token))
	latestNonceBytes, _, err := whiteListedAcc.AccountDataHandler().RetrieveValue(nonceKey)
	require.Nil(t, err)
	latestNonce := big.NewInt(0).SetBytes(latestNonceBytes).Uint64()
	require.Zero(t, latestNonce)

	// check metadata from vm output
	esdtDataBytes := vmOutput.Logs[0].Topics[3]
	var esdtDataFromLog esdt.ESDigitalToken
	err = nftCreate.marshaller.Unmarshal(&esdtDataFromLog, esdtDataBytes)
	require.Nil(t, err)
	expectedMetaEsdt := &esdt.ESDigitalToken{
		Type:  uint32(tokenType),
		Value: quantity,
		TokenMetaData: &esdt.MetaData{
			Nonce:      nonce.Uint64(),
			Name:       []byte(name),
			Creator:    originalCreator,
			Royalties:  uint32(royalties),
			Hash:       hash,
			URIs:       uris,
			Attributes: attributes,
		},
	}
	require.Equal(t, expectedMetaEsdt, &esdtDataFromLog)
}

func TestEsdtNFTCreate_ProcessBuiltinFunctionCrossChainToken(t *testing.T) {
	t.Parallel()

	accounts := createAccountsAdapterWithMap()
	enableEpochsHandler := &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ValueLengthCheckFlag || flag == SaveToSystemAccountFlag || flag == SendAlwaysFlag || flag == AlwaysSaveTokenMetaDataFlag || flag == DynamicEsdtFlag
		},
	}
	crossChainTokenHandler := &mock.CrossChainTokenCheckerMock{
		IsCrossChainOperationCalled: func(tokenID []byte) bool {
			return true
		},
	}
	ctc, _ := NewCrossChainTokenChecker(nil, getWhiteListedAddress())
	esdtRoleHandler, _ := NewESDTRolesFunc(marshallerMock, ctc, false)
	esdtDtaStorage := createNewESDTDataStorageHandlerWithArgs(&mock.GlobalSettingsHandlerStub{}, accounts, enableEpochsHandler, crossChainTokenHandler)

	args := createESDTNFTCreateArgs()
	args.CrossChainTokenCheckerHandler = ctc
	args.RolesHandler = esdtRoleHandler
	args.Accounts = esdtDtaStorage.accounts
	args.EsdtStorageHandler = esdtDtaStorage
	args.EnableEpochsHandler = enableEpochsHandler

	nftCreate, _ := NewESDTNFTCreateFunc(args)
	address := []byte("whiteListedAddress")
	sender := mock.NewUserAccount(address)
	sysAccount, err := esdtDtaStorage.getSystemAccount(defaultQueryOptions())
	require.Nil(t, err)
	uris := [][]byte{[]byte("uri1"), []byte("uri2")}

	t.Run("create nft v2 should work", func(t *testing.T) {
		token := "sov1-NFTV2-123456"
		tokenType := core.NonFungibleV2
		quantity := big.NewInt(1)
		nonce := big.NewInt(22)
		esdtMetaData := processCrossChainCreate(t, nftCreate, sender, token, nonce, tokenType, quantity, uris)

		data, err := getTokenDataFromAccount(sysAccount, []byte(token), nonce.Uint64())
		require.Nil(t, data) // key should not be in system account
		require.Nil(t, err)

		esdtData, latestNonce := readNFTData(t, sender, nftCreate.marshaller, []byte(token), nonce.Uint64(), nil) // from user account
		require.Zero(t, latestNonce)
		checkESDTNFTMetaData(t, tokenType, quantity, esdtMetaData, esdtData)
	})

	t.Run("create dynamic nft should work", func(t *testing.T) {
		token := "sov2-DYNFT-123456"
		tokenType := core.DynamicNFT
		quantity := big.NewInt(1)
		nonce := big.NewInt(16)
		esdtMetaData := processCrossChainCreate(t, nftCreate, sender, token, nonce, tokenType, quantity, uris)

		data, err := getTokenDataFromAccount(sysAccount, []byte(token), nonce.Uint64())
		require.Nil(t, data) // key should not be in system account
		require.Nil(t, err)

		esdtData, latestNonce := readNFTData(t, sender, nftCreate.marshaller, []byte(token), nonce.Uint64(), nil) // from user account
		require.Zero(t, latestNonce)
		checkESDTNFTMetaData(t, tokenType, quantity, esdtMetaData, esdtData)
	})

	t.Run("create sft should work", func(t *testing.T) {
		token := "sov2-SFT-1q2w3e"
		tokenType := core.SemiFungible
		quantity := big.NewInt(20)
		nonce := big.NewInt(3)
		esdtMetaData := processCrossChainCreate(t, nftCreate, sender, token, nonce, tokenType, quantity, uris)

		esdtData, latestNonce := readNFTData(t, sysAccount, nftCreate.marshaller, []byte(token), nonce.Uint64(), nil) // from system account
		require.Zero(t, latestNonce)
		checkESDTNFTMetaData(t, tokenType, quantity, esdtMetaData, esdtData)

		esdtData, latestNonce = readNFTData(t, sender, nftCreate.marshaller, []byte(token), nonce.Uint64(), nil) // from user account
		require.Zero(t, latestNonce)
		checkESDTNFTMetaData(t, tokenType, quantity, nil, esdtData)
	})

	t.Run("create dynamic sft should work", func(t *testing.T) {
		token := "sov3-DSFT-1q2w33"
		tokenType := core.DynamicSFT
		quantity := big.NewInt(15)
		nonce := big.NewInt(33)
		esdtMetaData := processCrossChainCreate(t, nftCreate, sender, token, nonce, tokenType, quantity, uris)

		esdtData, latestNonce := readNFTData(t, sysAccount, nftCreate.marshaller, []byte(token), nonce.Uint64(), nil) // from system account
		require.Zero(t, latestNonce)
		checkESDTNFTMetaData(t, tokenType, quantity, esdtMetaData, esdtData)

		esdtData, latestNonce = readNFTData(t, sender, nftCreate.marshaller, []byte(token), nonce.Uint64(), nil) // from user account
		require.Zero(t, latestNonce)
		checkESDTNFTMetaData(t, tokenType, quantity, nil, esdtData)
	})

	t.Run("create metaesdt should work", func(t *testing.T) {
		token := "sov3-MESDT-1fg23d"
		tokenType := core.MetaFungible
		quantity := big.NewInt(56)
		nonce := big.NewInt(684)
		esdtMetaData := processCrossChainCreate(t, nftCreate, sender, token, nonce, tokenType, quantity, uris)

		esdtData, latestNonce := readNFTData(t, sysAccount, nftCreate.marshaller, []byte(token), nonce.Uint64(), nil) // from system account
		require.Zero(t, latestNonce)
		checkESDTNFTMetaData(t, tokenType, quantity, esdtMetaData, esdtData)

		esdtData, latestNonce = readNFTData(t, sender, nftCreate.marshaller, []byte(token), nonce.Uint64(), nil) // from user account
		require.Zero(t, latestNonce)
		checkESDTNFTMetaData(t, tokenType, quantity, nil, esdtData)
	})

	t.Run("create dynamic metaesdt should work", func(t *testing.T) {
		token := "sov1-DMESDT-f2f2d3"
		tokenType := core.DynamicMeta
		quantity := big.NewInt(1024)
		nonce := big.NewInt(1024)
		uris1 := [][]byte{[]byte("uri1")} // simulate with different uris
		esdtMetaData := processCrossChainCreate(t, nftCreate, sender, token, nonce, tokenType, quantity, uris1)

		esdtData, latestNonce := readNFTData(t, sysAccount, nftCreate.marshaller, []byte(token), nonce.Uint64(), nil) // from system account
		require.Zero(t, latestNonce)
		checkESDTNFTMetaData(t, tokenType, quantity, esdtMetaData, esdtData)

		esdtData, latestNonce = readNFTData(t, sender, nftCreate.marshaller, []byte(token), nonce.Uint64(), nil) // from user account
		require.Zero(t, latestNonce)
		checkESDTNFTMetaData(t, tokenType, quantity, nil, esdtData)
	})
}

func processCrossChainCreate(
	t *testing.T,
	nftCreate *esdtNFTCreate,
	sender vmcommon.UserAccountHandler,
	token string,
	nonce *big.Int,
	tokenType core.ESDTType,
	quantity *big.Int,
	uris [][]byte,
) *esdt.MetaData {
	name := "name"
	royalties := 100 //1%
	hash := []byte("12345678901234567890123456789012")
	attributes := []byte("attributes")
	originalCreator := []byte("originalCreator")

	arguments := [][]byte{
		[]byte(token),
		quantity.Bytes(),
		[]byte(name),
		big.NewInt(int64(royalties)).Bytes(),
		hash,
		attributes,
	}
	arguments = append(arguments, uris...)
	arguments = append(arguments,
		big.NewInt(int64(tokenType)).Bytes(),
		nonce.Bytes(),
		originalCreator,
	)

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: sender.AddressBytes(),
			CallValue:  big.NewInt(0),
			Arguments:  arguments,
		},
		RecipientAddr: sender.AddressBytes(),
	}
	vmOutput, err := nftCreate.ProcessBuiltinFunction(sender, nil, vmInput)
	require.Nil(t, err)
	require.NotNil(t, vmOutput)

	return &esdt.MetaData{
		Nonce:      nonce.Uint64(),
		Name:       []byte(name),
		Creator:    originalCreator,
		Royalties:  uint32(royalties),
		Hash:       hash,
		URIs:       uris,
		Attributes: attributes,
	}
}

func TestEsdtNFTCreate_ProcessBuiltinFunctionCrossChainTokenErrorCases(t *testing.T) {
	t.Parallel()

	accounts := createAccountsAdapterWithMap()
	enableEpochsHandler := &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ValueLengthCheckFlag || flag == SaveToSystemAccountFlag || flag == SendAlwaysFlag || flag == AlwaysSaveTokenMetaDataFlag || flag == DynamicEsdtFlag
		},
	}
	crossChainTokenHandler := &mock.CrossChainTokenCheckerMock{
		IsCrossChainOperationCalled: func(tokenID []byte) bool {
			return true
		},
	}
	esdtDtaStorage := createNewESDTDataStorageHandlerWithArgs(&mock.GlobalSettingsHandlerStub{}, accounts, enableEpochsHandler, crossChainTokenHandler)
	ctc, _ := NewCrossChainTokenChecker(nil, getWhiteListedAddress())
	esdtRoleHandler, _ := NewESDTRolesFunc(marshallerMock, ctc, false)

	args := createESDTNFTCreateArgs()
	args.CrossChainTokenCheckerHandler = ctc
	args.RolesHandler = esdtRoleHandler
	args.Accounts = esdtDtaStorage.accounts
	args.EsdtStorageHandler = esdtDtaStorage

	nftCreate, _ := NewESDTNFTCreateFunc(args)
	address := []byte("whiteListedAddress")
	userSender := []byte("userAccountAddress")
	sender := mock.NewUserAccount(address)

	t.Run("invalid num of args without exec on dest", func(t *testing.T) {
		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallerAddr: sender.AddressBytes(),
				CallValue:  big.NewInt(0),
				Arguments: [][]byte{
					[]byte("sov1-TOKEN-abcdef"),
					big.NewInt(2).Bytes(),
					[]byte("name"),
					big.NewInt(int64(100)).Bytes(),
					[]byte("12345678901234567890123456789012"),
					[]byte("attributes"),
					[]byte("uri1"),
				},
			},
			RecipientAddr: sender.AddressBytes(),
		}

		// missing token type
		vmOutput, err := nftCreate.ProcessBuiltinFunction(sender, nil, vmInput)
		requireErrorIsInvalidArgsCrossChain(t, vmOutput, err)

		// missing nonce
		vmInput.VMInput.Arguments = append(vmInput.VMInput.Arguments, big.NewInt(int64(core.NonFungibleV2)).Bytes())
		vmOutput, err = nftCreate.ProcessBuiltinFunction(sender, nil, vmInput)
		requireErrorIsInvalidArgsCrossChain(t, vmOutput, err)

		// missing original creator
		vmInput.VMInput.Arguments = append(vmInput.VMInput.Arguments, big.NewInt(1).Bytes())
		vmOutput, err = nftCreate.ProcessBuiltinFunction(sender, nil, vmInput)
		requireErrorIsInvalidArgsCrossChain(t, vmOutput, err)
	})

	t.Run("invalid num of args in exec on dest", func(t *testing.T) {
		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallerAddr: userSender,
				CallValue:  big.NewInt(0),
				Arguments: [][]byte{
					[]byte("sov1-TOKEN-abcdef"),
					big.NewInt(2).Bytes(),
					[]byte("name"),
					big.NewInt(int64(100)).Bytes(),
					[]byte("12345678901234567890123456789012"),
					[]byte("attributes"),
					[]byte("uri1"),
					[]byte("whiteListedAddress"),
				},
				CallType: vm.ExecOnDestByCaller,
			},
			RecipientAddr: userSender,
		}

		// missing token type
		vmOutput, err := nftCreate.ProcessBuiltinFunction(sender, nil, vmInput)
		requireErrorIsInvalidArgsCrossChain(t, vmOutput, err)

		// missing nonce
		vmInput.VMInput.Arguments[7] = big.NewInt(int64(core.DynamicSFT)).Bytes()
		vmInput.VMInput.Arguments = append(vmInput.VMInput.Arguments, []byte("whiteListedAddress"))
		vmOutput, err = nftCreate.ProcessBuiltinFunction(nil, nil, vmInput)
		requireErrorIsInvalidArgsCrossChain(t, vmOutput, err)

		// missing original creator
		vmInput.VMInput.Arguments[8] = big.NewInt(1).Bytes()
		vmInput.VMInput.Arguments = append(vmInput.VMInput.Arguments, []byte("whiteListedAddress"))
		vmOutput, err = nftCreate.ProcessBuiltinFunction(sender, nil, vmInput)
		requireErrorIsInvalidArgsCrossChain(t, vmOutput, err)
	})

	t.Run("address is not whitelisted", func(t *testing.T) {
		senderInvalid := mock.NewUserAccount([]byte("notWhiteListed"))
		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallerAddr: senderInvalid.AddressBytes(),
				CallValue:  big.NewInt(0),
				Arguments: [][]byte{
					[]byte("sov1-TOKEN-abcdef"),
					big.NewInt(2).Bytes(),
					[]byte("name"),
					big.NewInt(int64(100)).Bytes(),
					[]byte("12345678901234567890123456789012"),
					[]byte("attributes"),
					[]byte("uri1"),
					big.NewInt(int64(core.MetaFungible)).Bytes(),
					big.NewInt(123).Bytes(),
					[]byte("creator"),
				},
			},
			RecipientAddr: senderInvalid.AddressBytes(),
		}

		vmOutput, err := nftCreate.ProcessBuiltinFunction(senderInvalid, nil, vmInput)
		require.Equal(t, err, ErrActionNotAllowed)
		require.Nil(t, vmOutput)
	})

	t.Run("invalid quantity", func(t *testing.T) {
		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallerAddr: userSender,
				CallValue:  big.NewInt(0),
				Arguments: [][]byte{
					[]byte("sov1-TOKEN-abcdef"),
					big.NewInt(2).Bytes(),
					[]byte("name"),
					big.NewInt(int64(100)).Bytes(),
					[]byte("12345678901234567890123456789012"),
					[]byte("attributes"),
					[]byte("uri1"),
					big.NewInt(int64(core.NonFungibleV2)).Bytes(),
					big.NewInt(123).Bytes(),
					[]byte("creator"),
				},
			},
			RecipientAddr: userSender,
		}

		vmOutput, err := nftCreate.ProcessBuiltinFunction(sender, nil, vmInput)
		require.ErrorIs(t, err, ErrInvalidArguments)
		require.True(t, strings.Contains(err.Error(), "invalid quantity"))
		require.Nil(t, vmOutput)
	})

	t.Run("invalid token type", func(t *testing.T) {
		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallerAddr: userSender,
				CallValue:  big.NewInt(0),
				Arguments: [][]byte{
					[]byte("sov1-TOKEN-abcdef"),
					big.NewInt(1).Bytes(),
					[]byte("name"),
					big.NewInt(int64(100)).Bytes(),
					[]byte("12345678901234567890123456789012"),
					[]byte("attributes"),
					[]byte("uri1"),
					big.NewInt(int64(999)).Bytes(),
					big.NewInt(123).Bytes(),
					[]byte("creator"),
				},
			},
			RecipientAddr: userSender,
		}

		vmOutput, err := nftCreate.ProcessBuiltinFunction(sender, nil, vmInput)
		require.ErrorIs(t, err, ErrInvalidArguments)
		require.True(t, strings.Contains(err.Error(), "invalid esdt type"))
		require.Nil(t, vmOutput)
	})
}

func checkESDTNFTMetaData(t *testing.T, tokenType core.ESDTType, quantity *big.Int, esdtMetaData *esdt.MetaData, esdtData *esdt.ESDigitalToken) {
	require.Equal(t, esdtMetaData, esdtData.TokenMetaData)
	require.Equal(t, uint32(tokenType), esdtData.Type)
	require.Equal(t, quantity, esdtData.Value)
}

func requireErrorIsInvalidArgsCrossChain(t *testing.T, vmOutput *vmcommon.VMOutput, err error) {
	require.ErrorIs(t, err, ErrInvalidNumberOfArguments)
	require.True(t, strings.Contains(err.Error(), "for cross chain"))
	require.Nil(t, vmOutput)
}

func readNFTData(t *testing.T, account vmcommon.UserAccountHandler, marshaller vmcommon.Marshalizer, tokenID []byte, nonce uint64, _ []byte) (*esdt.ESDigitalToken, uint64) {
	nonceKey := getNonceKey(tokenID)
	latestNonceBytes, _, err := account.AccountDataHandler().RetrieveValue(nonceKey)
	require.Nil(t, err)
	latestNonce := big.NewInt(0).SetBytes(latestNonceBytes).Uint64()

	data, err := getTokenDataFromAccount(account, tokenID, nonce)
	require.Nil(t, err)

	esdtData := &esdt.ESDigitalToken{}
	err = marshaller.Unmarshal(esdtData, data)
	require.Nil(t, err)

	return esdtData, latestNonce
}

func getTokenDataFromAccount(account vmcommon.UserAccountHandler, tokenID []byte, nonce uint64) ([]byte, error) {
	createdTokenID := []byte(baseESDTKeyPrefix)
	createdTokenID = append(createdTokenID, tokenID...)
	tokenKey := computeESDTNFTTokenKey(createdTokenID, nonce)
	data, _, err := account.AccountDataHandler().RetrieveValue(tokenKey)
	return data, err
}
