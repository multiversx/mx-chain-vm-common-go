package builtInFunctions

import (
	"bytes"
	"encoding/hex"
	"errors"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/data/esdt"
	"github.com/ElrondNetwork/elrond-go-core/data/smartContractResult"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/mock"
	"github.com/stretchr/testify/assert"
)

func createNewESDTDataStorageHandler() *esdtDataStorage {
	acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	accounts := &mock.AccountsStub{LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
		return acnt, nil
	}}
	args := ArgsNewESDTDataStorage{
		Accounts:                accounts,
		GlobalSettingsHandler:   &mock.GlobalSettingsHandlerStub{},
		Marshalizer:             &mock.MarshalizerMock{},
		SaveToSystemEnableEpoch: 0,
		EpochNotifier:           &mock.EpochNotifierStub{},
		ShardCoordinator:        &mock.ShardCoordinatorStub{},
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
		Accounts:                accounts,
		GlobalSettingsHandler:   &mock.GlobalSettingsHandlerStub{},
		Marshalizer:             &mock.MarshalizerMock{},
		SaveToSystemEnableEpoch: 0,
		EpochNotifier:           &mock.EpochNotifierStub{},
		ShardCoordinator:        &mock.ShardCoordinatorStub{},
	}
	return args
}

func createNewESDTDataStorageHandlerWithArgs(
	globalSettingsHandler vmcommon.ESDTGlobalSettingsHandler,
	accounts vmcommon.AccountsAdapter,
) *esdtDataStorage {
	args := ArgsNewESDTDataStorage{
		Accounts:                accounts,
		GlobalSettingsHandler:   globalSettingsHandler,
		Marshalizer:             &mock.MarshalizerMock{},
		SaveToSystemEnableEpoch: 10,
		EpochNotifier:           &mock.EpochNotifierStub{},
		ShardCoordinator:        &mock.ShardCoordinatorStub{},
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
	args.EpochNotifier = nil
	e, err = NewESDTDataStorage(args)
	assert.Nil(t, e)
	assert.Equal(t, err, ErrNilEpochHandler)

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
	key := core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier + tokenIdentifier
	nonce := uint64(10)
	esdtDataBytes, _ := args.Marshalizer.Marshal(esdtData)
	tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtDataBytes)

	esdtDataGet, _, err := e.GetESDTNFTTokenOnDestination(userAcc, []byte(key), nonce)
	assert.Nil(t, err)
	assert.Equal(t, esdtData, esdtDataGet)
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
	key := core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier + tokenIdentifier
	nonce := uint64(10)
	esdtDataBytes, _ := args.Marshalizer.Marshal(esdtData)
	tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtDataBytes)

	systemAcc, _ := e.getSystemAccount()
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
	key := core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier + tokenIdentifier
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
	key := core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier + tokenIdentifier
	nonce := uint64(10)
	esdtDataBytes, _ := args.Marshalizer.Marshal(esdtData)
	tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtDataBytes)

	systemAcc, _ := e.getSystemAccount()
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

	err := e.saveESDTMetaDataToSystemAccount(0, []byte("TCK"), 0, nil, true)
	assert.Nil(t, err)

	err = e.saveESDTMetaDataToSystemAccount(0, []byte("TCK"), 1, &esdt.ESDigitalToken{}, true)
	assert.Nil(t, err)
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
	key := core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier + tokenIdentifier
	nonce := uint64(10)
	esdtDataBytes, _ := args.Marshalizer.Marshal(esdtData)
	tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtDataBytes)

	systemAcc, _ := e.getSystemAccount()
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
	key := core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier + tokenIdentifier
	esdtDataBytes, _ := args.Marshalizer.Marshal(esdtData)
	tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtDataBytes)

	esdtData.Value = big.NewInt(0)
	_, err := e.SaveESDTNFTToken([]byte("address"), userAcc, []byte(key), nonce, esdtData, false, false)
	assert.Nil(t, err)

	val, err := userAcc.AccountDataHandler().RetrieveValue(tokenKey)
	assert.Nil(t, val)
	assert.Nil(t, err)

	esdtMetaData, err := e.getESDTMetaDataFromSystemAccount(tokenKey)
	assert.Nil(t, err)
	assert.Equal(t, esdtData.TokenMetaData, esdtMetaData)
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

	shardCoordinator.ComputeIdCalled = func(_ []byte) uint32 {
		return core.MetachainShardId
	}
	val, err = e.WasAlreadySentToDestinationShardAndUpdateState(tickerID, 1, dstAddress)
	assert.True(t, val)
	assert.Nil(t, err)

	shardCoordinator.ComputeIdCalled = func(_ []byte) uint32 {
		return 1
	}
	shardCoordinator.NumberOfShardsCalled = func() uint32 {
		return 5
	}
	val, err = e.WasAlreadySentToDestinationShardAndUpdateState(tickerID, 1, dstAddress)
	assert.False(t, val)
	assert.Nil(t, err)

	systemAcc, _ := e.getSystemAccount()
	metaData := &esdt.MetaData{
		Name: []byte("test"),
	}
	esdtDataOnSystemAcc := &esdt.ESDigitalToken{TokenMetaData: metaData}
	esdtMetaDataBytes, _ := args.Marshalizer.Marshal(esdtDataOnSystemAcc)
	key := core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier + string(tickerID)
	tokenKey := append([]byte(key), big.NewInt(1).Bytes()...)
	_ = systemAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtMetaDataBytes)

	val, err = e.WasAlreadySentToDestinationShardAndUpdateState(tickerID, 1, dstAddress)
	assert.False(t, val)
	assert.Nil(t, err)

	val, err = e.WasAlreadySentToDestinationShardAndUpdateState(tickerID, 1, dstAddress)
	assert.False(t, val)
	assert.Nil(t, err)

	e.flagSendAlwaysEnableEpoch.Reset()
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

func TestEsdtDataStorage_SaveNFTMetaDataToSystemAccount(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewESDTDataStorage()
	shardCoordinator := &mock.ShardCoordinatorStub{}
	args.ShardCoordinator = shardCoordinator
	e, _ := NewESDTDataStorage(args)

	e.flagSaveToSystemAccount.Reset()
	err := e.SaveNFTMetaDataToSystemAccount(nil)
	assert.Nil(t, err)

	_ = e.flagSaveToSystemAccount.SetReturningPrevious()
	err = e.SaveNFTMetaDataToSystemAccount(nil)
	assert.Equal(t, err, ErrNilTransactionHandler)

	scr := &smartContractResult.SmartContractResult{
		SndAddr: []byte("address1"),
		RcvAddr: []byte("address2"),
	}

	err = e.SaveNFTMetaDataToSystemAccount(scr)
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

	err = e.SaveNFTMetaDataToSystemAccount(scr)
	assert.Nil(t, err)

	scr.Data = []byte("function")
	err = e.SaveNFTMetaDataToSystemAccount(scr)
	assert.Nil(t, err)

	scr.Data = []byte("function@01@02@03@04")
	err = e.SaveNFTMetaDataToSystemAccount(scr)
	assert.Nil(t, err)

	scr.Data = []byte(core.BuiltInFunctionESDTNFTTransfer + "@01@02@03@04")
	err = e.SaveNFTMetaDataToSystemAccount(scr)
	assert.NotNil(t, err)

	scr.Data = []byte(core.BuiltInFunctionESDTNFTTransfer + "@01@02@03@00")
	err = e.SaveNFTMetaDataToSystemAccount(scr)
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
	err = e.SaveNFTMetaDataToSystemAccount(scr)
	assert.Nil(t, err)

	key := core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier + string(tickerID)
	tokenKey := append([]byte(key), big.NewInt(1).Bytes()...)
	esdtGetData, _, _ := e.getESDTDigitalTokenDataFromSystemAccount(tokenKey)

	assert.Equal(t, esdtData.TokenMetaData, esdtGetData.TokenMetaData)
}

func TestEsdtDataStorage_SaveNFTMetaDataToSystemAccountWithMultiTransfer(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewESDTDataStorage()
	shardCoordinator := &mock.ShardCoordinatorStub{}
	args.ShardCoordinator = shardCoordinator
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
	err := e.SaveNFTMetaDataToSystemAccount(scr)
	assert.True(t, errors.Is(err, ErrInvalidArguments))

	scr.Data = []byte(core.BuiltInFunctionMultiESDTNFTTransfer + "@02@" + hex.EncodeToString(tickerID) + "@01@01@" + hex.EncodeToString(esdtMarshalled))
	err = e.SaveNFTMetaDataToSystemAccount(scr)
	assert.True(t, errors.Is(err, ErrInvalidArguments))

	scr.Data = []byte(core.BuiltInFunctionMultiESDTNFTTransfer + "@02@" + hex.EncodeToString(tickerID) + "@02@10@" +
		hex.EncodeToString(tickerID) + "@01@" + hex.EncodeToString(esdtMarshalled))
	err = e.SaveNFTMetaDataToSystemAccount(scr)
	assert.Nil(t, err)

	key := core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier + string(tickerID)
	tokenKey := append([]byte(key), big.NewInt(1).Bytes()...)
	esdtGetData, _, _ := e.getESDTDigitalTokenDataFromSystemAccount(tokenKey)

	assert.Equal(t, esdtData.TokenMetaData, esdtGetData.TokenMetaData)

	otherTokenKey := append([]byte(key), big.NewInt(2).Bytes()...)
	esdtGetData, _, err = e.getESDTDigitalTokenDataFromSystemAccount(otherTokenKey)
	assert.Nil(t, esdtGetData)
	assert.Nil(t, err)
}

func TestEsdtDataStorage_checkCollectionFrozen(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewESDTDataStorage()
	shardCoordinator := &mock.ShardCoordinatorStub{}
	args.ShardCoordinator = shardCoordinator
	e, _ := NewESDTDataStorage(args)

	e.flagCheckFrozenCollection.SetValue(false)

	acnt, _ := e.accounts.LoadAccount([]byte("address1"))
	userAcc := acnt.(vmcommon.UserAccountHandler)

	tickerID := []byte("TOKEN-ABCDEF")
	esdtTokenKey := append(e.keyPrefix, tickerID...)
	err := e.checkCollectionIsFrozenForAccount(userAcc, esdtTokenKey, 1, false)
	assert.Nil(t, err)

	e.flagCheckFrozenCollection.SetValue(true)
	err = e.checkCollectionIsFrozenForAccount(userAcc, esdtTokenKey, 0, false)
	assert.Nil(t, err)

	err = e.checkCollectionIsFrozenForAccount(userAcc, esdtTokenKey, 1, true)
	assert.Nil(t, err)

	err = e.checkCollectionIsFrozenForAccount(userAcc, esdtTokenKey, 1, false)
	assert.Nil(t, err)

	tokenData, _ := getESDTDataFromKey(userAcc, esdtTokenKey, e.marshalizer)

	esdtUserMetadata := ESDTUserMetadataFromBytes(tokenData.Properties)
	esdtUserMetadata.Frozen = false
	tokenData.Properties = esdtUserMetadata.ToBytes()
	_ = saveESDTData(userAcc, tokenData, esdtTokenKey, e.marshalizer)

	err = e.checkCollectionIsFrozenForAccount(userAcc, esdtTokenKey, 1, false)
	assert.Nil(t, err)

	esdtUserMetadata.Frozen = true
	tokenData.Properties = esdtUserMetadata.ToBytes()
	_ = saveESDTData(userAcc, tokenData, esdtTokenKey, e.marshalizer)

	err = e.checkCollectionIsFrozenForAccount(userAcc, esdtTokenKey, 1, false)
	assert.Equal(t, err, ErrESDTIsFrozenForAccount)
}
