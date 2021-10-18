package builtInFunctions

import (
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/data/esdt"
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
	esdtMetaDataBytes, _ := args.Marshalizer.Marshal(metaData)
	_ = systemAcc.AccountDataHandler().SaveKeyValue(tokenKey, esdtMetaDataBytes)

	esdtDataGet, _, err := e.GetESDTNFTTokenOnDestination(userAcc, []byte(key), nonce)
	assert.Nil(t, err)
	esdtData.TokenMetaData = metaData
	assert.Equal(t, esdtData, esdtDataGet)
}
