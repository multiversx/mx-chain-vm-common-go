package builtInFunctions

import (
	"bytes"
	"encoding/hex"
	"errors"
	"math/big"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/data/esdt"
	"github.com/multiversx/mx-chain-core-go/data/smartContractResult"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/assert"
)

func createNewESDTDataStorageHandler() *esdtDataStorage {
	acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	accounts := &mock.AccountsStub{LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
		return acnt, nil
	}}
	args := ArgsNewESDTDataStorage{
		Accounts:              accounts,
		GlobalSettingsHandler: &mock.GlobalSettingsHandlerStub{},
		Marshalizer:           &mock.MarshalizerMock{},
		EnableEpochsHandler: &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return flag == SaveToSystemAccountFlag || flag == SendAlwaysFlag
			},
		},
		ShardCoordinator: &mock.ShardCoordinatorStub{},
	}
	dataStore, _ := NewESDTDataStorage(args)
	return dataStore
}

func createMockArgsForNewESDTDataStorage() ArgsNewESDTDataStorage {
	acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	accounts := &mock.AccountsStub{LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
		return acnt, nil
	}}
	args := ArgsNewESDTDataStorage{
		Accounts:              accounts,
		GlobalSettingsHandler: &mock.GlobalSettingsHandlerStub{},
		Marshalizer:           &mock.MarshalizerMock{},
		EnableEpochsHandler: &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return flag == SaveToSystemAccountFlag || flag == SendAlwaysFlag
			},
		},
		ShardCoordinator: &mock.ShardCoordinatorStub{},
	}
	return args
}

func createNewESDTDataStorageHandlerWithArgs(
	globalSettingsHandler vmcommon.GlobalMetadataHandler,
	accounts vmcommon.AccountsAdapter,
	enableEpochsHandler vmcommon.EnableEpochsHandler,
) *esdtDataStorage {
	args := ArgsNewESDTDataStorage{
		Accounts:              accounts,
		GlobalSettingsHandler: globalSettingsHandler,
		Marshalizer:           &mock.MarshalizerMock{},
		EnableEpochsHandler:   enableEpochsHandler,
		ShardCoordinator:      &mock.ShardCoordinatorStub{},
	}
	dataStore, _ := NewESDTDataStorage(args)
	return dataStore
}

func TestNewESDTDataStorage(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewESDTDataStorage()
	args.Marshalizer = nil
	e, err := NewESDTDataStorage(args)
	assert.Nil(t, e)
	assert.Equal(t, err, ErrNilMarshalizer)

	args = createMockArgsForNewESDTDataStorage()
	args.Accounts = nil
	e, err = NewESDTDataStorage(args)
	assert.Nil(t, e)
	assert.Equal(t, err, ErrNilAccountsAdapter)

	args = createMockArgsForNewESDTDataStorage()
	args.ShardCoordinator = nil
	e, err = NewESDTDataStorage(args)
	assert.Nil(t, e)
	assert.Equal(t, err, ErrNilShardCoordinator)

	args = createMockArgsForNewESDTDataStorage()
	args.GlobalSettingsHandler = nil
	e, err = NewESDTDataStorage(args)
	assert.Nil(t, e)
	assert.Equal(t, err, ErrNilGlobalSettingsHandler)

	args = createMockArgsForNewESDTDataStorage()
	args.EnableEpochsHandler = nil
	e, err = NewESDTDataStorage(args)
	assert.Nil(t, e)
	assert.Equal(t, err, ErrNilEnableEpochsHandler)

	args = createMockArgsForNewESDTDataStorage()
	e, err = NewESDTDataStorage(args)
	assert.Nil(t, err)
	assert.False(t, e.IsInterfaceNil())
}

func TestEsdtDataStorage_GetESDTNFTTokenOnDestinationNoDataInSystemAcc(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewESDTDataStorage()
	e, _ := NewESDTDataStorage(args)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	esdtData := &esdt.ESDigitalToken{
		TokenMetaData: &esdt.MetaData{
			Name: []byte("test"),
		},
		Value: big.NewInt(10),
	}

	tokenIdentifier := "testTkn"
	key := baseESDTKeyPrefix + tokenIdentifier
	nonce := uint64(10)
	esdtDataBytes, _ := args.Marshalizer.Marshal(esdtData)
	tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtDataBytes)

	esdtDataGet, _, err := e.GetESDTNFTTokenOnDestination(userAcc, []byte(key), nonce)
	assert.Nil(t, err)
	assert.Equal(t, esdtData, esdtDataGet)
}

func TestEsdtDataStorage_GetESDTNFTTokenOnDestinationTypeNonFungibleV2(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewESDTDataStorage()
	acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	systemAccLoaded := false
	args.Accounts = &mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			systemAccLoaded = true
			return acnt, nil
		}}
	e, _ := NewESDTDataStorage(args)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	esdtData := &esdt.ESDigitalToken{
		TokenMetaData: &esdt.MetaData{
			Name:    []byte("test"),
			Creator: []byte("creator"),
		},
		Type:  uint32(core.NonFungibleV2),
		Value: big.NewInt(10),
	}

	tokenIdentifier := "testTkn"
	key := baseESDTKeyPrefix + tokenIdentifier
	nonce := uint64(10)
	esdtDataBytes, _ := args.Marshalizer.Marshal(esdtData)
	tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtDataBytes)

	esdtDataGet, _, err := e.GetESDTNFTTokenOnDestination(userAcc, []byte(key), nonce)
	assert.Nil(t, err)
	assert.Equal(t, esdtData, esdtDataGet)
	assert.False(t, systemAccLoaded)
}

func TestEsdtDataStorage_GetESDTNFTTokenOnDestinationGetNodeFromDbErr(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewESDTDataStorage()
	e, _ := NewESDTDataStorage(args)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	userAcc.RetrieveValueCalled = func(key []byte) ([]byte, uint32, error) {
		return nil, 0, core.NewGetNodeFromDBErrWithKey(key, errors.New("error"), "")
	}

	esdtDataGet, _, err := e.GetESDTNFTTokenOnDestination(userAcc, []byte("key"), 1)
	assert.Nil(t, esdtDataGet)
	assert.True(t, core.IsGetNodeFromDBError(err))
}

func TestEsdtDataStorage_GetESDTNFTTokenOnDestinationGetDataFromSystemAcc(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewESDTDataStorage()
	e, _ := NewESDTDataStorage(args)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	esdtData := &esdt.ESDigitalToken{
		Value: big.NewInt(10),
	}

	tokenIdentifier := "testTkn"
	key := baseESDTKeyPrefix + tokenIdentifier
	nonce := uint64(10)
	esdtDataBytes, _ := args.Marshalizer.Marshal(esdtData)
	tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtDataBytes)

	systemAcc, _ := e.getSystemAccount(defaultQueryOptions())
	metaData := &esdt.MetaData{
		Name: []byte("test"),
	}
	esdtDataOnSystemAcc := &esdt.ESDigitalToken{TokenMetaData: metaData}
	esdtMetaDataBytes, _ := args.Marshalizer.Marshal(esdtDataOnSystemAcc)
	_ = systemAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtMetaDataBytes)

	esdtDataGet, _, err := e.GetESDTNFTTokenOnDestination(userAcc, []byte(key), nonce)
	assert.Nil(t, err)
	esdtData.TokenMetaData = metaData
	assert.Equal(t, esdtData, esdtDataGet)
}

func TestEsdtDataStorage_GetESDTNFTTokenOnDestinationWithCustomSystemAccount(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewESDTDataStorage()
	e, _ := NewESDTDataStorage(args)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	esdtData := &esdt.ESDigitalToken{
		Value: big.NewInt(10),
	}

	tokenIdentifier := "testTkn"
	key := baseESDTKeyPrefix + tokenIdentifier
	nonce := uint64(10)
	esdtDataBytes, _ := args.Marshalizer.Marshal(esdtData)
	tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtDataBytes)

	systemAcc, _ := e.getSystemAccount(defaultQueryOptions())
	metaData := &esdt.MetaData{
		Name: []byte("test"),
	}
	esdtDataOnSystemAcc := &esdt.ESDigitalToken{TokenMetaData: metaData}
	esdtMetaDataBytes, _ := args.Marshalizer.Marshal(esdtDataOnSystemAcc)
	_ = systemAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtMetaDataBytes)

	retrieveValueFromCustomAccountCalled := false
	customSystemAccount := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(key []byte) ([]byte, uint32, error) {
					retrieveValueFromCustomAccountCalled = true
					return esdtMetaDataBytes, 0, nil
				},
			}
		},
	}
	esdtDataGet, _, err := e.GetESDTNFTTokenOnDestinationWithCustomSystemAccount(userAcc, []byte(key), nonce, customSystemAccount)
	assert.Nil(t, err)
	esdtData.TokenMetaData = metaData
	assert.Equal(t, esdtData, esdtDataGet)
	assert.True(t, retrieveValueFromCustomAccountCalled)
}

func TestEsdtDataStorage_GetESDTNFTTokenOnDestinationMarshalERR(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewESDTDataStorage()
	e, _ := NewESDTDataStorage(args)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	esdtData := &esdt.ESDigitalToken{
		Value: big.NewInt(10),
		TokenMetaData: &esdt.MetaData{
			Name: []byte("test"),
		},
	}

	tokenIdentifier := "testTkn"
	key := baseESDTKeyPrefix + tokenIdentifier
	nonce := uint64(10)
	esdtDataBytes, _ := args.Marshalizer.Marshal(esdtData)
	esdtDataBytes = append(esdtDataBytes, esdtDataBytes...)
	tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtDataBytes)

	_, _, err := e.GetESDTNFTTokenOnDestination(userAcc, []byte(key), nonce)
	assert.NotNil(t, err)

	_, err = e.GetESDTNFTTokenOnSender(userAcc, []byte(key), nonce)
	assert.NotNil(t, err)
}

func TestEsdtDataStorage_MarshalErrorOnSystemACC(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewESDTDataStorage()
	e, _ := NewESDTDataStorage(args)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	esdtData := &esdt.ESDigitalToken{
		Value: big.NewInt(10),
	}

	tokenIdentifier := "testTkn"
	key := baseESDTKeyPrefix + tokenIdentifier
	nonce := uint64(10)
	esdtDataBytes, _ := args.Marshalizer.Marshal(esdtData)
	tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtDataBytes)

	systemAcc, _ := e.getSystemAccount(defaultQueryOptions())
	metaData := &esdt.MetaData{
		Name: []byte("test"),
	}
	esdtDataOnSystemAcc := &esdt.ESDigitalToken{TokenMetaData: metaData}
	esdtMetaDataBytes, _ := args.Marshalizer.Marshal(esdtDataOnSystemAcc)
	esdtMetaDataBytes = append(esdtMetaDataBytes, esdtMetaDataBytes...)
	_ = systemAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtMetaDataBytes)

	_, _, err := e.GetESDTNFTTokenOnDestination(userAcc, []byte(key), nonce)
	assert.NotNil(t, err)
}

func TestESDTDataStorage_saveDataToSystemAccNotNFTOrMetaData(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewESDTDataStorage()
	e, _ := NewESDTDataStorage(args)

	err := e.saveESDTMetaDataToSystemAccount(nil, 0, []byte("TCK"), 0, nil, true)
	assert.Nil(t, err)

	err = e.saveESDTMetaDataToSystemAccount(nil, 0, []byte("TCK"), 1, &esdt.ESDigitalToken{}, true)
	assert.Nil(t, err)
}

func TestESDTDataStorage_saveESDTMetaDataToSystemAccountGetNodeFromDbErrForSystemAcc(t *testing.T) {
	t.Parallel()

	systemAcc := mock.NewAccountWrapMock([]byte("system acc address"))
	systemAcc.RetrieveValueCalled = func(key []byte) ([]byte, uint32, error) {
		return nil, 0, core.NewGetNodeFromDBErrWithKey(key, errors.New("error"), "")
	}

	args := createMockArgsForNewESDTDataStorage()
	args.Accounts = &mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return systemAcc, nil
		},
	}
	e, _ := NewESDTDataStorage(args)

	esdtData := &esdt.ESDigitalToken{
		TokenMetaData: &esdt.MetaData{},
	}

	err := e.saveESDTMetaDataToSystemAccount(nil, 0, []byte("TCK"), 1, esdtData, true)
	assert.True(t, core.IsGetNodeFromDBError(err))
}

func TestESDTDataStorage_saveESDTMetaDataToSystemAccountGetNodeFromDbErrForUserAcc(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewESDTDataStorage()
	args.EnableEpochsHandler = &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == FixOldTokenLiquidityFlag || flag == SendAlwaysFlag
		},
	}
	e, _ := NewESDTDataStorage(args)

	esdtData := &esdt.ESDigitalToken{
		TokenMetaData: &esdt.MetaData{},
	}

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	userAcc.RetrieveValueCalled = func(key []byte) ([]byte, uint32, error) {
		return nil, 0, core.NewGetNodeFromDBErrWithKey(key, errors.New("error"), "")
	}

	err := e.saveESDTMetaDataToSystemAccount(userAcc, 0, []byte("TCK"), 1, esdtData, true)
	assert.True(t, core.IsGetNodeFromDBError(err))
}

func TestEsdtDataStorage_SaveESDTNFTTokenNoChangeInSystemAcc(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewESDTDataStorage()
	e, _ := NewESDTDataStorage(args)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	esdtData := &esdt.ESDigitalToken{
		Value: big.NewInt(10),
	}

	tokenIdentifier := "testTkn"
	key := baseESDTKeyPrefix + tokenIdentifier
	nonce := uint64(10)
	esdtDataBytes, _ := args.Marshalizer.Marshal(esdtData)
	tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtDataBytes)

	systemAcc, _ := e.getSystemAccount(defaultQueryOptions())
	metaData := &esdt.MetaData{
		Name: []byte("test"),
	}
	esdtDataOnSystemAcc := &esdt.ESDigitalToken{TokenMetaData: metaData}
	esdtMetaDataBytes, _ := args.Marshalizer.Marshal(esdtDataOnSystemAcc)
	_ = systemAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtMetaDataBytes)

	newMetaData := &esdt.MetaData{Name: []byte("newName")}
	transferESDTData := &esdt.ESDigitalToken{Value: big.NewInt(100), TokenMetaData: newMetaData}
	_, err := e.SaveESDTNFTToken([]byte("address"), userAcc, []byte(key), nonce, transferESDTData, false, false)
	assert.Nil(t, err)

	esdtDataGet, _, err := e.GetESDTNFTTokenOnDestination(userAcc, []byte(key), nonce)
	assert.Nil(t, err)
	esdtData.TokenMetaData = metaData
	esdtData.Value = big.NewInt(100)
	assert.Equal(t, esdtData, esdtDataGet)
}

func TestEsdtDataStorage_SaveESDTNFTTokenAlwaysSaveTokenMetaDataEnabled(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewESDTDataStorage()
	args.EnableEpochsHandler = &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == SaveToSystemAccountFlag || flag == SendAlwaysFlag || flag == AlwaysSaveTokenMetaDataFlag
		},
	}
	dataStorage, _ := NewESDTDataStorage(args)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	nonce := uint64(10)

	t.Run("new token should not rewrite metadata", func(t *testing.T) {
		newToken := &esdt.ESDigitalToken{
			Value: big.NewInt(10),
		}
		tokenIdentifier := "newTkn"
		key := baseESDTKeyPrefix + tokenIdentifier
		tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)

		_ = saveESDTData(userAcc, newToken, tokenKey, args.Marshalizer)

		systemAcc, _ := dataStorage.getSystemAccount(defaultQueryOptions())
		metaData := &esdt.MetaData{
			Name: []byte("test"),
		}
		esdtDataOnSystemAcc := &esdt.ESDigitalToken{
			TokenMetaData: metaData,
			Reserved:      []byte{1},
		}
		esdtMetaDataBytes, _ := args.Marshalizer.Marshal(esdtDataOnSystemAcc)
		_ = systemAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtMetaDataBytes)

		newMetaData := &esdt.MetaData{Name: []byte("newName")}
		transferESDTData := &esdt.ESDigitalToken{Value: big.NewInt(100), TokenMetaData: newMetaData}
		_, err := dataStorage.SaveESDTNFTToken([]byte("address"), userAcc, []byte(key), nonce, transferESDTData, false, false)
		assert.Nil(t, err)

		esdtDataGet, _, err := dataStorage.GetESDTNFTTokenOnDestination(userAcc, []byte(key), nonce)
		assert.Nil(t, err)

		expectedESDTData := &esdt.ESDigitalToken{
			Value:         big.NewInt(100),
			TokenMetaData: metaData,
		}
		assert.Equal(t, expectedESDTData, esdtDataGet)
	})
	t.Run("old token should rewrite metadata", func(t *testing.T) {
		newToken := &esdt.ESDigitalToken{
			Value: big.NewInt(10),
		}
		tokenIdentifier := "newTkn"
		key := baseESDTKeyPrefix + tokenIdentifier
		tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)

		_ = saveESDTData(userAcc, newToken, tokenKey, args.Marshalizer)

		systemAcc, _ := dataStorage.getSystemAccount(defaultQueryOptions())
		metaData := &esdt.MetaData{
			Name: []byte("test"),
		}
		esdtDataOnSystemAcc := &esdt.ESDigitalToken{
			TokenMetaData: metaData,
		}
		esdtMetaDataBytes, _ := args.Marshalizer.Marshal(esdtDataOnSystemAcc)
		_ = systemAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtMetaDataBytes)

		newMetaData := &esdt.MetaData{Name: []byte("newName")}
		transferESDTData := &esdt.ESDigitalToken{Value: big.NewInt(100), TokenMetaData: newMetaData}
		esdtDataGet := setAndGetStoredToken(t, dataStorage, userAcc, []byte(key), nonce, transferESDTData)

		expectedESDTData := &esdt.ESDigitalToken{
			Value:         big.NewInt(100),
			TokenMetaData: newMetaData,
		}
		assert.Equal(t, expectedESDTData, esdtDataGet)
	})
	t.Run("old token should not rewrite metadata if the flags are not set", func(t *testing.T) {
		localArgs := createMockArgsForNewESDTDataStorage()
		localEpochsHandler := &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return flag == SaveToSystemAccountFlag || flag == SendAlwaysFlag || flag == AlwaysSaveTokenMetaDataFlag
			},
		}
		localArgs.EnableEpochsHandler = localEpochsHandler
		localDataStorage, _ := NewESDTDataStorage(localArgs)

		newToken := &esdt.ESDigitalToken{
			Value: big.NewInt(10),
		}
		tokenIdentifier := "newTkn"
		key := baseESDTKeyPrefix + tokenIdentifier
		tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)

		_ = saveESDTData(userAcc, newToken, tokenKey, localArgs.Marshalizer)

		systemAcc, _ := localDataStorage.getSystemAccount(defaultQueryOptions())
		metaData := &esdt.MetaData{
			Name: []byte("test"),
		}
		esdtDataOnSystemAcc := &esdt.ESDigitalToken{
			TokenMetaData: metaData,
		}
		esdtMetaDataBytes, _ := localArgs.Marshalizer.Marshal(esdtDataOnSystemAcc)
		_ = systemAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtMetaDataBytes)

		newMetaData := &esdt.MetaData{Name: []byte("newName")}
		transferESDTData := &esdt.ESDigitalToken{Value: big.NewInt(100), TokenMetaData: newMetaData}
		expectedESDTData := &esdt.ESDigitalToken{
			Value:         big.NewInt(100),
			TokenMetaData: metaData,
		}

		localEpochsHandler.IsFlagEnabledCalled = func(flag core.EnableEpochFlag) bool {
			return flag == SaveToSystemAccountFlag || flag == SendAlwaysFlag
		}

		esdtDataGet := setAndGetStoredToken(t, localDataStorage, userAcc, []byte(key), nonce, transferESDTData)
		assert.Equal(t, expectedESDTData, esdtDataGet)

		localEpochsHandler.IsFlagEnabledCalled = func(flag core.EnableEpochFlag) bool {
			return flag == SaveToSystemAccountFlag || flag == AlwaysSaveTokenMetaDataFlag
		}

		esdtDataGet = setAndGetStoredToken(t, localDataStorage, userAcc, []byte(key), nonce, transferESDTData)
		assert.Equal(t, expectedESDTData, esdtDataGet)
	})
}

func setAndGetStoredToken(
	tb testing.TB,
	esdtDataStorage *esdtDataStorage,
	userAcc vmcommon.UserAccountHandler,
	key []byte,
	nonce uint64,
	transferESDTData *esdt.ESDigitalToken,
) *esdt.ESDigitalToken {
	_, err := esdtDataStorage.SaveESDTNFTToken([]byte("address"), userAcc, key, nonce, transferESDTData, false, false)
	assert.Nil(tb, err)

	esdtDataGet, _, err := esdtDataStorage.GetESDTNFTTokenOnDestination(userAcc, key, nonce)
	assert.Nil(tb, err)

	return esdtDataGet
}

func TestEsdtDataStorage_SaveESDTNFTTokenWhenQuantityZero(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewESDTDataStorage()
	e, _ := NewESDTDataStorage(args)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	nonce := uint64(10)
	esdtData := &esdt.ESDigitalToken{
		Value: big.NewInt(10),
		TokenMetaData: &esdt.MetaData{
			Name:  []byte("test"),
			Nonce: nonce,
		},
	}

	tokenIdentifier := "testTkn"
	key := baseESDTKeyPrefix + tokenIdentifier
	esdtDataBytes, _ := args.Marshalizer.Marshal(esdtData)
	tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtDataBytes)

	esdtData.Value = big.NewInt(0)
	_, err := e.SaveESDTNFTToken([]byte("address"), userAcc, []byte(key), nonce, esdtData, false, false)
	assert.Nil(t, err)

	val, _, err := userAcc.AccountDataHandler().RetrieveValue(tokenKey)
	assert.Nil(t, val)
	assert.Nil(t, err)

	esdtMetaData, err := e.getESDTMetaDataFromSystemAccount(tokenKey, defaultQueryOptions())
	assert.Nil(t, err)
	assert.Equal(t, esdtData.TokenMetaData, esdtMetaData)
}

func TestEsdtDataStorage_SaveESDTNFTToken(t *testing.T) {
	t.Parallel()

	t.Run("migrate metadata from system account to user account for NonFungibleV2", func(t *testing.T) {
		t.Parallel()

		tokenIdentifier := "newTkn"
		nonce := uint64(10)
		key := baseESDTKeyPrefix + tokenIdentifier
		tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)

		args := createMockArgsForNewESDTDataStorage()
		args.GlobalSettingsHandler = &mock.GlobalSettingsHandlerStub{
			GetTokenTypeCalled: func(esdtTokenKey []byte) (uint32, error) {
				return uint32(core.NonFungibleV2), nil
			},
		}
		args.EnableEpochsHandler = &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return flag == SaveToSystemAccountFlag || flag == SendAlwaysFlag || flag == DynamicEsdtFlag
			},
		}
		dataStorage, _ := NewESDTDataStorage(args)
		userAcc := mock.NewAccountWrapMock([]byte("addr"))
		nftToken := &esdt.ESDigitalToken{
			Value: big.NewInt(10),
		}
		_ = saveESDTData(userAcc, nftToken, tokenKey, args.Marshalizer)

		systemAcc, _ := dataStorage.getSystemAccount(defaultQueryOptions())
		metaData := &esdt.MetaData{
			Name: []byte("test"),
		}
		esdtDataOnSystemAcc := &esdt.ESDigitalToken{
			TokenMetaData: metaData,
			Reserved:      []byte{1},
		}
		esdtMetaDataBytes, _ := args.Marshalizer.Marshal(esdtDataOnSystemAcc)
		_ = systemAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtMetaDataBytes)

		nftToken.Type = uint32(core.NonFungible)
		nftToken.TokenMetaData = metaData

		_, err := dataStorage.SaveESDTNFTToken([]byte("address"), userAcc, []byte(key), nonce, nftToken, false, false)
		assert.Nil(t, err)

		// metadata has been removed from the system account
		val, _, _ := systemAcc.AccountDataHandler().RetrieveValue(tokenKey)
		assert.Nil(t, val)

		nftToken.Type = uint32(core.NonFungibleV2)
		nftTokenBytes, _ := args.Marshalizer.Marshal(nftToken)
		// metadata has been added to the user account
		val, _, err = userAcc.RetrieveValue(tokenKey)
		assert.Nil(t, err)
		assert.Equal(t, nftTokenBytes, val)
	})
}

func TestEsdtDataStorage_WasAlreadySentToDestinationShard(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewESDTDataStorage()
	shardCoordinator := &mock.ShardCoordinatorStub{}
	args.ShardCoordinator = shardCoordinator
	e, _ := NewESDTDataStorage(args)

	tickerID := []byte("ticker")
	dstAddress := []byte("dstAddress")
	val, err := e.WasAlreadySentToDestinationShardAndUpdateState(tickerID, 0, dstAddress)
	assert.True(t, val)
	assert.Nil(t, err)

	val, err = e.WasAlreadySentToDestinationShardAndUpdateState(tickerID, 1, dstAddress)
	assert.True(t, val)
	assert.Nil(t, err)

	enableEpochsHandler, _ := args.EnableEpochsHandler.(*mock.EnableEpochsHandlerStub)
	enableEpochsHandler.IsFlagEnabledCalled = func(flag core.EnableEpochFlag) bool {
		return flag == SaveToSystemAccountFlag
	}
	shardCoordinator.ComputeIdCalled = func(_ []byte) uint32 {
		return core.MetachainShardId
	}
	val, err = e.WasAlreadySentToDestinationShardAndUpdateState(tickerID, 1, dstAddress)
	assert.True(t, val)
	assert.Nil(t, err)

	enableEpochsHandler.IsFlagEnabledCalled = func(flag core.EnableEpochFlag) bool {
		return flag == SaveToSystemAccountFlag || flag == SendAlwaysFlag
	}

	shardCoordinator.ComputeIdCalled = func(_ []byte) uint32 {
		return 1
	}
	shardCoordinator.NumberOfShardsCalled = func() uint32 {
		return 5
	}
	val, err = e.WasAlreadySentToDestinationShardAndUpdateState(tickerID, 1, dstAddress)
	assert.False(t, val)
	assert.Nil(t, err)

	systemAcc, _ := e.getSystemAccount(defaultQueryOptions())
	metaData := &esdt.MetaData{
		Name: []byte("test"),
	}
	esdtDataOnSystemAcc := &esdt.ESDigitalToken{TokenMetaData: metaData}
	esdtMetaDataBytes, _ := args.Marshalizer.Marshal(esdtDataOnSystemAcc)
	key := baseESDTKeyPrefix + string(tickerID)
	tokenKey := append([]byte(key), big.NewInt(1).Bytes()...)
	_ = systemAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtMetaDataBytes)

	val, err = e.WasAlreadySentToDestinationShardAndUpdateState(tickerID, 1, dstAddress)
	assert.False(t, val)
	assert.Nil(t, err)

	val, err = e.WasAlreadySentToDestinationShardAndUpdateState(tickerID, 1, dstAddress)
	assert.False(t, val)
	assert.Nil(t, err)

	enableEpochsHandler.IsFlagEnabledCalled = func(flag core.EnableEpochFlag) bool {
		return flag == SaveToSystemAccountFlag
	}
	val, err = e.WasAlreadySentToDestinationShardAndUpdateState(tickerID, 1, dstAddress)
	assert.False(t, val)
	assert.Nil(t, err)

	shardCoordinator.NumberOfShardsCalled = func() uint32 {
		return 10
	}
	val, err = e.WasAlreadySentToDestinationShardAndUpdateState(tickerID, 1, dstAddress)
	assert.True(t, val)
	assert.Nil(t, err)
}

func TestEsdtDataStorage_SaveNFTMetaData(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewESDTDataStorage()
	shardCoordinator := &mock.ShardCoordinatorStub{}
	args.ShardCoordinator = shardCoordinator
	e, _ := NewESDTDataStorage(args)

	enableEpochsHandler, _ := args.EnableEpochsHandler.(*mock.EnableEpochsHandlerStub)
	enableEpochsHandler.IsFlagEnabledCalled = func(flag core.EnableEpochFlag) bool {
		return flag == SendAlwaysFlag
	}
	err := e.SaveNFTMetaData(nil)
	assert.Nil(t, err)

	enableEpochsHandler.IsFlagEnabledCalled = func(flag core.EnableEpochFlag) bool {
		return flag == SaveToSystemAccountFlag || flag == SendAlwaysFlag
	}
	err = e.SaveNFTMetaData(nil)
	assert.Nil(t, err)

	enableEpochsHandler.IsFlagEnabledCalled = func(flag core.EnableEpochFlag) bool {
		return flag == SaveToSystemAccountFlag
	}
	err = e.SaveNFTMetaData(nil)
	assert.Equal(t, err, ErrNilTransactionHandler)

	scr := &smartContractResult.SmartContractResult{
		SndAddr: []byte("address1"),
		RcvAddr: []byte("address2"),
	}

	err = e.SaveNFTMetaData(scr)
	assert.Nil(t, err)

	shardCoordinator.ComputeIdCalled = func(address []byte) uint32 {
		if bytes.Equal(address, scr.SndAddr) {
			return 0
		}
		if bytes.Equal(address, scr.RcvAddr) {
			return 1
		}
		return 2
	}
	shardCoordinator.NumberOfShardsCalled = func() uint32 {
		return 3
	}
	shardCoordinator.SelfIdCalled = func() uint32 {
		return 1
	}

	err = e.SaveNFTMetaData(scr)
	assert.Nil(t, err)

	scr.Data = []byte("function")
	err = e.SaveNFTMetaData(scr)
	assert.Nil(t, err)

	scr.Data = []byte("function@01@02@03@04")
	err = e.SaveNFTMetaData(scr)
	assert.Nil(t, err)

	scr.Data = []byte(core.BuiltInFunctionESDTNFTTransfer + "@01@02@03@04")
	err = e.SaveNFTMetaData(scr)
	assert.NotNil(t, err)

	scr.Data = []byte(core.BuiltInFunctionESDTNFTTransfer + "@01@02@03@00")
	err = e.SaveNFTMetaData(scr)
	assert.Nil(t, err)

	tickerID := []byte("TCK")
	esdtData := &esdt.ESDigitalToken{
		Value: big.NewInt(10),
		TokenMetaData: &esdt.MetaData{
			Name: []byte("test"),
		},
	}
	esdtMarshalled, _ := args.Marshalizer.Marshal(esdtData)
	scr.Data = []byte(core.BuiltInFunctionESDTNFTTransfer + "@" + hex.EncodeToString(tickerID) + "@01@01@" + hex.EncodeToString(esdtMarshalled))
	err = e.SaveNFTMetaData(scr)
	assert.Nil(t, err)

	key := baseESDTKeyPrefix + string(tickerID)
	tokenKey := append([]byte(key), big.NewInt(1).Bytes()...)
	esdtGetData, _, _ := e.getESDTDigitalTokenDataFromSystemAccount(tokenKey, defaultQueryOptions())

	assert.Equal(t, esdtData.TokenMetaData, esdtGetData.TokenMetaData)
}

func TestEsdtDataStorage_getESDTDigitalTokenDataFromSystemAccountGetNodeFromDbErr(t *testing.T) {
	t.Parallel()

	systemAcc := mock.NewAccountWrapMock([]byte("system acc address"))
	systemAcc.RetrieveValueCalled = func(key []byte) ([]byte, uint32, error) {
		return nil, 0, core.NewGetNodeFromDBErrWithKey(key, errors.New("error"), "")
	}

	args := createMockArgsForNewESDTDataStorage()
	args.Accounts = &mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return systemAcc, nil
		},
	}
	e, _ := NewESDTDataStorage(args)

	esdtDataGet, _, err := e.getESDTDigitalTokenDataFromSystemAccount([]byte("tokenKey"), defaultQueryOptions())
	assert.Nil(t, esdtDataGet)
	assert.True(t, core.IsGetNodeFromDBError(err))
}

func TestEsdtDataStorage_SaveNFTMetaDataWithMultiTransfer(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewESDTDataStorage()
	shardCoordinator := &mock.ShardCoordinatorStub{}
	args.ShardCoordinator = shardCoordinator
	args.EnableEpochsHandler = &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == SaveToSystemAccountFlag
		},
	}
	e, _ := NewESDTDataStorage(args)

	scr := &smartContractResult.SmartContractResult{
		SndAddr: []byte("address1"),
		RcvAddr: []byte("address2"),
	}

	shardCoordinator.ComputeIdCalled = func(address []byte) uint32 {
		if bytes.Equal(address, scr.SndAddr) {
			return 0
		}
		if bytes.Equal(address, scr.RcvAddr) {
			return 1
		}
		return 2
	}
	shardCoordinator.NumberOfShardsCalled = func() uint32 {
		return 3
	}
	shardCoordinator.SelfIdCalled = func() uint32 {
		return 1
	}

	tickerID := []byte("TCK")
	esdtData := &esdt.ESDigitalToken{
		Value: big.NewInt(10),
		TokenMetaData: &esdt.MetaData{
			Name: []byte("test"),
		},
	}
	esdtMarshalled, _ := args.Marshalizer.Marshal(esdtData)
	scr.Data = []byte(core.BuiltInFunctionMultiESDTNFTTransfer + "@00@" + hex.EncodeToString(tickerID) + "@01@01@" + hex.EncodeToString(esdtMarshalled))
	err := e.SaveNFTMetaData(scr)
	assert.True(t, errors.Is(err, ErrInvalidArguments))

	scr.Data = []byte(core.BuiltInFunctionMultiESDTNFTTransfer + "@02@" + hex.EncodeToString(tickerID) + "@01@01@" + hex.EncodeToString(esdtMarshalled))
	err = e.SaveNFTMetaData(scr)
	assert.True(t, errors.Is(err, ErrInvalidArguments))

	scr.Data = []byte(core.BuiltInFunctionMultiESDTNFTTransfer + "@02@" + hex.EncodeToString(tickerID) + "@02@10@" +
		hex.EncodeToString(tickerID) + "@01@" + hex.EncodeToString(esdtMarshalled))
	err = e.SaveNFTMetaData(scr)
	assert.Nil(t, err)

	key := baseESDTKeyPrefix + string(tickerID)
	tokenKey := append([]byte(key), big.NewInt(1).Bytes()...)
	esdtGetData, _, _ := e.getESDTDigitalTokenDataFromSystemAccount(tokenKey, defaultQueryOptions())

	assert.Equal(t, esdtData.TokenMetaData, esdtGetData.TokenMetaData)

	otherTokenKey := append([]byte(key), big.NewInt(2).Bytes()...)
	esdtGetData, _, err = e.getESDTDigitalTokenDataFromSystemAccount(otherTokenKey, defaultQueryOptions())
	assert.Nil(t, esdtGetData)
	assert.Nil(t, err)
}

func TestEsdtDataStorage_checkCollectionFrozen(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewESDTDataStorage()
	shardCoordinator := &mock.ShardCoordinatorStub{}
	args.ShardCoordinator = shardCoordinator
	e, _ := NewESDTDataStorage(args)

	enableEpochsHandler, _ := args.EnableEpochsHandler.(*mock.EnableEpochsHandlerStub)
	enableEpochsHandler.IsFlagEnabledCalled = func(flag core.EnableEpochFlag) bool {
		return false
	}

	acnt, _ := e.accounts.LoadAccount([]byte("address1"))
	userAcc := acnt.(vmcommon.UserAccountHandler)

	tickerID := []byte("TOKEN-ABCDEF")
	esdtTokenKey := append(e.keyPrefix, tickerID...)
	err := e.checkCollectionIsFrozenForAccount(userAcc, esdtTokenKey, 1, false)
	assert.Nil(t, err)

	enableEpochsHandler.IsFlagEnabledCalled = func(flag core.EnableEpochFlag) bool {
		return flag == CheckFrozenCollectionFlag
	}
	err = e.checkCollectionIsFrozenForAccount(userAcc, esdtTokenKey, 0, false)
	assert.Nil(t, err)

	err = e.checkCollectionIsFrozenForAccount(userAcc, esdtTokenKey, 1, true)
	assert.Nil(t, err)

	err = e.checkCollectionIsFrozenForAccount(userAcc, esdtTokenKey, 1, false)
	assert.Nil(t, err)

	tokenData, _ := getESDTDataFromKey(userAcc, esdtTokenKey, e.marshaller)

	esdtUserMetadata := ESDTUserMetadataFromBytes(tokenData.Properties)
	esdtUserMetadata.Frozen = false
	tokenData.Properties = esdtUserMetadata.ToBytes()
	_ = saveESDTData(userAcc, tokenData, esdtTokenKey, e.marshaller)

	err = e.checkCollectionIsFrozenForAccount(userAcc, esdtTokenKey, 1, false)
	assert.Nil(t, err)

	esdtUserMetadata.Frozen = true
	tokenData.Properties = esdtUserMetadata.ToBytes()
	_ = saveESDTData(userAcc, tokenData, esdtTokenKey, e.marshaller)

	err = e.checkCollectionIsFrozenForAccount(userAcc, esdtTokenKey, 1, false)
	assert.Equal(t, err, ErrESDTIsFrozenForAccount)
}

func TestGetEsdtDataFromKey(t *testing.T) {
	t.Parallel()

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	userAcc.RetrieveValueCalled = func(key []byte) ([]byte, uint32, error) {
		return nil, 0, core.NewGetNodeFromDBErrWithKey(key, errors.New("error"), "")
	}
	tokenData, err := getESDTDataFromKey(userAcc, []byte("esdtTokenKey"), &mock.MarshalizerMock{})
	assert.Nil(t, tokenData)
	assert.True(t, core.IsGetNodeFromDBError(err))
}

func TestEsdtDataStorage_checkCollectionFrozenGetNodeFromDbErr(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewESDTDataStorage()
	args.EnableEpochsHandler = &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == CheckFrozenCollectionFlag
		},
	}
	e, _ := NewESDTDataStorage(args)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	userAcc.RetrieveValueCalled = func(key []byte) ([]byte, uint32, error) {
		return nil, 0, core.NewGetNodeFromDBErrWithKey(key, errors.New("error"), "")
	}

	err := e.checkCollectionIsFrozenForAccount(userAcc, []byte("key"), 1, false)
	assert.True(t, core.IsGetNodeFromDBError(err))
}

func TestEsdtDataStorage_AddToLiquiditySystemAcc(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewESDTDataStorage()
	e, _ := NewESDTDataStorage(args)

	tokenKey := append(e.keyPrefix, []byte("TOKEN-ababab")...)
	nonce := uint64(10)
	err := e.AddToLiquiditySystemAcc(tokenKey, 0, nonce, big.NewInt(10), false)
	assert.Equal(t, err, ErrNilESDTData)

	systemAcc, _ := e.getSystemAccount(defaultQueryOptions())
	esdtData := &esdt.ESDigitalToken{Value: big.NewInt(0)}
	marshalledData, _ := e.marshaller.Marshal(esdtData)

	esdtNFTTokenKey := computeESDTNFTTokenKey(tokenKey, nonce)
	_ = systemAcc.AccountDataHandler().SaveKeyValue(esdtNFTTokenKey, marshalledData)

	err = e.AddToLiquiditySystemAcc(tokenKey, 0, nonce, big.NewInt(10), false)
	assert.Nil(t, err)

	esdtData = &esdt.ESDigitalToken{Value: big.NewInt(10), Reserved: []byte{1}}
	marshalledData, _ = e.marshaller.Marshal(esdtData)

	_ = systemAcc.AccountDataHandler().SaveKeyValue(esdtNFTTokenKey, marshalledData)
	err = e.AddToLiquiditySystemAcc(tokenKey, 0, nonce, big.NewInt(10), false)
	assert.Nil(t, err)

	esdtData, _, _ = e.getESDTDigitalTokenDataFromSystemAccount(esdtNFTTokenKey, defaultQueryOptions())
	assert.Equal(t, esdtData.Value, big.NewInt(20))

	err = e.AddToLiquiditySystemAcc(tokenKey, 0, nonce, big.NewInt(-20), false)
	assert.Nil(t, err)

	esdtData, _, _ = e.getESDTDigitalTokenDataFromSystemAccount(esdtNFTTokenKey, defaultQueryOptions())
	assert.Nil(t, esdtData)
}

func TestEsdtDataStorage_IsNFTWithMetadataOnAccount(t *testing.T) {
	t.Parallel()

	t.Run("returns false if not NonFungibleV2", func(t *testing.T) {
		t.Parallel()

		esdtData := &esdt.ESDigitalToken{
			Type: uint32(core.NonFungible),
		}
		assert.False(t, isNFTWithMetadataOnAccount(esdtData))
	})
	t.Run("returns false if MetaData is nil", func(t *testing.T) {
		t.Parallel()

		esdtData := &esdt.ESDigitalToken{
			TokenMetaData: nil,
			Type:          uint32(core.NonFungibleV2),
		}
		assert.False(t, isNFTWithMetadataOnAccount(esdtData))
	})
	t.Run("returns false if Creator is empty", func(t *testing.T) {
		t.Parallel()

		esdtData := &esdt.ESDigitalToken{
			TokenMetaData: &esdt.MetaData{
				Creator: nil,
			},
			Type: uint32(core.NonFungibleV2),
		}
		assert.False(t, isNFTWithMetadataOnAccount(esdtData))
	})
	t.Run("returns true for NonFungibleV2 and existing MetaData", func(t *testing.T) {
		t.Parallel()

		esdtData := &esdt.ESDigitalToken{
			TokenMetaData: &esdt.MetaData{
				Creator: []byte("creator"),
			},
			Type: uint32(core.NonFungibleV2),
		}
		assert.True(t, isNFTWithMetadataOnAccount(esdtData))
	})
}

func TestEsdtDataStorage_ShouldSaveMetadataInSystemAccount(t *testing.T) {
	t.Parallel()

	t.Run("returns false if SaveToSystemAccountFlag flag disabled", func(t *testing.T) {
		t.Parallel()

		args := createMockArgsForNewESDTDataStorage()
		args.EnableEpochsHandler = &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return false
			},
		}
		e, _ := NewESDTDataStorage(args)

		assert.False(t, e.shouldSaveMetadataInSystemAccount(uint32(core.NonFungibleV2)))
	})

	t.Run("returns true if dynamic token type", func(t *testing.T) {
		t.Parallel()

		args := createMockArgsForNewESDTDataStorage()
		args.EnableEpochsHandler = &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return flag == SaveToSystemAccountFlag
			},
		}
		e, _ := NewESDTDataStorage(args)

		assert.True(t, e.shouldSaveMetadataInSystemAccount(uint32(core.DynamicNFT)))
		assert.True(t, e.shouldSaveMetadataInSystemAccount(uint32(core.DynamicSFT)))
		assert.True(t, e.shouldSaveMetadataInSystemAccount(uint32(core.DynamicMeta)))
	})

	t.Run("returns false if NonFungibleV2", func(t *testing.T) {
		t.Parallel()

		args := createMockArgsForNewESDTDataStorage()
		args.EnableEpochsHandler = &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return flag == SaveToSystemAccountFlag
			},
		}
		e, _ := NewESDTDataStorage(args)

		assert.False(t, e.shouldSaveMetadataInSystemAccount(uint32(core.NonFungibleV2)))
		assert.True(t, e.shouldSaveMetadataInSystemAccount(uint32(core.NonFungible)))
	})
}

func TestEsdtDataStorage_GetMetaDataFromSystemAccount(t *testing.T) {
	t.Parallel()

	key := []byte("tokenKey")
	nonce := uint64(10)
	keyNonce := append(key, big.NewInt(int64(nonce)).Bytes()...)

	args := createMockArgsForNewESDTDataStorage()
	acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)

	args.Accounts = &mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		}}
	e, _ := NewESDTDataStorage(args)

	metaData := &esdt.MetaData{
		Name: []byte("test"),
	}
	esdtData := &esdt.ESDigitalToken{
		TokenMetaData: metaData,
	}
	esdtDataBytes, _ := e.marshaller.Marshal(esdtData)
	_ = acnt.SaveKeyValue(keyNonce, esdtDataBytes)

	retrievedMetaData, err := e.GetMetaDataFromSystemAccount(key, nonce)
	assert.Nil(t, err)
	assert.Equal(t, esdtData, retrievedMetaData)
}

func TestEsdtDataStorage_SaveMetaDataToSystemAccount(t *testing.T) {
	t.Parallel()

	key := []byte("tokenKey")
	nonce := uint64(10)
	keyNonce := append(key, big.NewInt(int64(nonce)).Bytes()...)

	args := createMockArgsForNewESDTDataStorage()
	acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	args.Accounts = &mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		}}
	e, _ := NewESDTDataStorage(args)

	metaData := &esdt.MetaData{
		Name: []byte("test"),
	}
	esdtData := &esdt.ESDigitalToken{
		TokenMetaData: metaData,
	}

	err := e.SaveMetaDataToSystemAccount(key, nonce, esdtData)
	assert.Nil(t, err)

	retrievedVal, _, err := acnt.AccountDataHandler().RetrieveValue(keyNonce)
	assert.Nil(t, err)
	esdtDataBytes, _ := e.marshaller.Marshal(esdtData)
	assert.Equal(t, esdtDataBytes, retrievedVal)
}

func TestEsdtDataStorage_SaveESDTNFTToken_migrateTypeAndMetadata(t *testing.T) {
	t.Parallel()

	t.Run("migrate token type for NonFungibleV2", func(t *testing.T) {
		t.Parallel()

		saveESDTNFTTokenMigrateTypeAndMetadata(t, core.NonFungibleV2)
	})
	t.Run("migrate metadata for SemiFungible", func(t *testing.T) {
		t.Parallel()

		saveESDTNFTTokenMigrateTypeAndMetadata(t, core.SemiFungible)
	})
	t.Run("migrate metadata for MetaFungible", func(t *testing.T) {
		t.Parallel()

		saveESDTNFTTokenMigrateTypeAndMetadata(t, core.MetaFungible)
	})
	t.Run("migrate metadata for DynamicNFT", func(t *testing.T) {
		t.Parallel()

		saveESDTNFTTokenMigrateTypeAndMetadata(t, core.DynamicNFT)
	})
	t.Run("migrate metadata for DynamicMeta", func(t *testing.T) {
		t.Parallel()

		saveESDTNFTTokenMigrateTypeAndMetadata(t, core.DynamicMeta)
	})
	t.Run("migrate metadata for DynamicSFT", func(t *testing.T) {
		t.Parallel()

		saveESDTNFTTokenMigrateTypeAndMetadata(t, core.DynamicSFT)
	})

}

func saveESDTNFTTokenMigrateTypeAndMetadata(t *testing.T, tokenType core.ESDTType) {
	args := createMockArgsForNewESDTDataStorage()
	args.EnableEpochsHandler = &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == SaveToSystemAccountFlag || flag == SendAlwaysFlag || flag == DynamicEsdtFlag
		},
	}
	args.GlobalSettingsHandler = &mock.GlobalSettingsHandlerStub{
		GetTokenTypeCalled: func(esdtTokenKey []byte) (uint32, error) {
			return uint32(tokenType), nil
		},
	}
	e, _ := NewESDTDataStorage(args)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	esdtData := &esdt.ESDigitalToken{
		Value: big.NewInt(10),
		Type:  uint32(core.NonFungible),
	}

	tokenIdentifier := "testTkn"
	key := baseESDTKeyPrefix + tokenIdentifier
	nonce := uint64(10)
	esdtDataBytes, _ := args.Marshalizer.Marshal(esdtData)
	tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtDataBytes)

	systemAcc, _ := e.getSystemAccount(defaultQueryOptions())
	metaData := &esdt.MetaData{
		Name: []byte("test"),
	}
	esdtDataOnSystemAcc := &esdt.ESDigitalToken{TokenMetaData: metaData}
	esdtMetaDataBytes, _ := args.Marshalizer.Marshal(esdtDataOnSystemAcc)
	_ = systemAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtMetaDataBytes)

	retrievedEsdtData, _, err := e.GetESDTNFTTokenOnDestination(userAcc, []byte(key), nonce)
	assert.Nil(t, err)
	assert.Equal(t, uint32(core.NonFungible), retrievedEsdtData.Type)

	esdtData.TokenMetaData = metaData
	_, err = e.SaveESDTNFTToken([]byte("address"), userAcc, []byte(key), nonce, esdtData, false, false)
	assert.Nil(t, err)

	retrievedEsdtData, _, err = e.GetESDTNFTTokenOnDestination(userAcc, []byte(key), nonce)
	assert.Nil(t, err)
	assert.Equal(t, uint32(tokenType), retrievedEsdtData.Type)

	if tokenType == core.NonFungibleV2 {
		retrievedMetaData, err := e.GetMetaDataFromSystemAccount([]byte(key), nonce)
		assert.Nil(t, err)
		assert.Nil(t, retrievedMetaData)
	} else {
		retrievedMetaData, err := e.GetMetaDataFromSystemAccount([]byte(key), nonce)
		assert.Nil(t, err)
		assert.Equal(t, &esdt.ESDigitalToken{TokenMetaData: metaData}, retrievedMetaData)
	}
}
