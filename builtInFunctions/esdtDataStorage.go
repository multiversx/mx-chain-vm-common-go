package builtInFunctions

import (
	"math/big"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data/esdt"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/atomic"
)

//TODO: resolve GetESDTToken on blockchain hook

type esdtDataStorage struct {
	accounts                vmcommon.AccountsAdapter
	globalSettingsHandler   vmcommon.ESDTGlobalSettingsHandler
	marshalizer             vmcommon.Marshalizer
	keyPrefix               []byte
	flagSaveToSystemAccount atomic.Flag
	saveToSystemEnableEpoch uint32
	shardCoordinator        vmcommon.Coordinator
}

// ArgsNewESDTDataStorage defines the argument list for new esdt data storage handler
type ArgsNewESDTDataStorage struct {
	Accounts                vmcommon.AccountsAdapter
	GlobalSettingsHandler   vmcommon.ESDTGlobalSettingsHandler
	Marshalizer             vmcommon.Marshalizer
	SaveToSystemEnableEpoch uint32
	EpochNotifier           vmcommon.EpochNotifier
	ShardCoordinator        vmcommon.Coordinator
}

// NewESDTDataStorage creates a new esdt data storage handler
func NewESDTDataStorage(args ArgsNewESDTDataStorage) (*esdtDataStorage, error) {
	if check.IfNil(args.Accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if check.IfNil(args.GlobalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
	}
	if check.IfNil(args.Marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(args.EpochNotifier) {
		return nil, ErrNilEpochHandler
	}
	if check.IfNil(args.ShardCoordinator) {
		return nil, ErrNilShardCoordinator
	}

	e := &esdtDataStorage{
		accounts:                args.Accounts,
		globalSettingsHandler:   args.GlobalSettingsHandler,
		marshalizer:             args.Marshalizer,
		keyPrefix:               []byte(core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier),
		flagSaveToSystemAccount: atomic.Flag{},
		saveToSystemEnableEpoch: args.SaveToSystemEnableEpoch,
		shardCoordinator:        args.ShardCoordinator,
	}
	args.EpochNotifier.RegisterNotifyHandler(e)

	return e, nil
}

// GetESDTNFTTokenOnSender gets the nft token on sender account
func (e *esdtDataStorage) GetESDTNFTTokenOnSender(
	accnt vmcommon.UserAccountHandler,
	esdtTokenKey []byte,
	nonce uint64,
) (*esdt.ESDigitalToken, error) {
	esdtData, isNew, err := e.GetESDTNFTTokenOnDestination(accnt, esdtTokenKey, nonce)
	if err != nil {
		return nil, err
	}
	if isNew {
		return nil, ErrNewNFTDataOnSenderAddress
	}

	return esdtData, nil
}

// GetESDTNFTTokenOnDestination gets the nft token on destination account
func (e *esdtDataStorage) GetESDTNFTTokenOnDestination(
	accnt vmcommon.UserAccountHandler,
	esdtTokenKey []byte,
	nonce uint64,
) (*esdt.ESDigitalToken, bool, error) {
	esdtNFTTokenKey := computeESDTNFTTokenKey(esdtTokenKey, nonce)
	esdtData := &esdt.ESDigitalToken{
		Value: big.NewInt(0),
		Type:  uint32(core.Fungible),
	}
	marshaledData, err := accnt.AccountDataHandler().RetrieveValue(esdtNFTTokenKey)
	if err != nil || len(marshaledData) == 0 {
		return esdtData, true, nil
	}

	err = e.marshalizer.Unmarshal(esdtData, marshaledData)
	if err != nil {
		return nil, false, err
	}

	if !e.flagSaveToSystemAccount.IsSet() || nonce == 0 {
		return esdtData, false, nil
	}

	esdtMetaData, err := e.getESDTMetaDataFromSystemAccount(esdtNFTTokenKey)
	if err != nil {
		return nil, false, err
	}
	if esdtMetaData != nil {
		esdtData.TokenMetaData = esdtMetaData
	}

	return esdtData, false, nil
}

func (e *esdtDataStorage) getESDTDigitalTokenDataFromSystemAccount(
	tokenKey []byte,
) (*esdt.ESDigitalToken, vmcommon.UserAccountHandler, error) {
	systemAcc, err := e.getSystemAccount()
	if err != nil {
		return nil, nil, err
	}

	marshaledData, err := systemAcc.AccountDataHandler().RetrieveValue(tokenKey)
	if err != nil || len(marshaledData) == 0 {
		return nil, systemAcc, nil
	}

	esdtData := &esdt.ESDigitalToken{}
	err = e.marshalizer.Unmarshal(esdtData, marshaledData)
	if err != nil {
		return nil, nil, err
	}

	return esdtData, systemAcc, nil
}

func (e *esdtDataStorage) getESDTMetaDataFromSystemAccount(
	tokenKey []byte,
) (*esdt.MetaData, error) {
	esdtData, _, err := e.getESDTDigitalTokenDataFromSystemAccount(tokenKey)
	if err != nil {
		return nil, err
	}
	if esdtData == nil {
		return nil, nil
	}

	return esdtData.TokenMetaData, nil
}

// SaveESDTNFTToken saves the nft token to the account and system account
func (e *esdtDataStorage) SaveESDTNFTToken(
	senderAddress []byte,
	acnt vmcommon.UserAccountHandler,
	esdtTokenKey []byte,
	nonce uint64,
	esdtData *esdt.ESDigitalToken,
	isReturnWithError bool,
) ([]byte, error) {
	err := checkFrozeAndPause(acnt.AddressBytes(), esdtTokenKey, esdtData, e.globalSettingsHandler, isReturnWithError)
	if err != nil {
		return nil, err
	}

	esdtNFTTokenKey := computeESDTNFTTokenKey(esdtTokenKey, nonce)
	err = checkFrozeAndPause(acnt.AddressBytes(), esdtNFTTokenKey, esdtData, e.globalSettingsHandler, isReturnWithError)
	if err != nil {
		return nil, err
	}

	if esdtData.Value.Cmp(zero) <= 0 {
		return nil, acnt.AccountDataHandler().SaveKeyValue(esdtNFTTokenKey, nil)
	}

	if !e.flagSaveToSystemAccount.IsSet() {
		marshaledData, err := e.marshalizer.Marshal(esdtData)
		if err != nil {
			return nil, err
		}

		return marshaledData, acnt.AccountDataHandler().SaveKeyValue(esdtNFTTokenKey, marshaledData)
	}

	senderShardID := e.shardCoordinator.ComputeId(senderAddress)
	err = e.saveESDTMetaDataToSystemAccount(senderShardID, esdtNFTTokenKey, nonce, esdtData, false)
	if err != nil {
		return nil, err
	}

	esdtDataOnAccount := &esdt.ESDigitalToken{
		Type:       esdtData.Type,
		Value:      esdtData.Value,
		Properties: esdtData.Properties,
	}
	marshaledData, err := e.marshalizer.Marshal(esdtDataOnAccount)
	if err != nil {
		return nil, err
	}

	return marshaledData, acnt.AccountDataHandler().SaveKeyValue(esdtNFTTokenKey, marshaledData)
}

// UpdateNFTMetaData updates the nft on system account and deletes information about which shard was it sent
func (e *esdtDataStorage) UpdateNFTMetaData(
	esdtTokenKey []byte,
	nonce uint64,
	esdtData *esdt.ESDigitalToken,
) error {
	if !e.flagSaveToSystemAccount.IsSet() {
		return nil
	}

	esdtNFTTokenKey := computeESDTNFTTokenKey(esdtTokenKey, nonce)
	return e.saveESDTMetaDataToSystemAccount(e.shardCoordinator.SelfId(), esdtNFTTokenKey, nonce, esdtData, true)
}

func (e *esdtDataStorage) saveESDTMetaDataToSystemAccount(
	senderShardID uint32,
	esdtNFTTokenKey []byte,
	nonce uint64,
	esdtData *esdt.ESDigitalToken,
	isUpdate bool,
) error {
	if nonce == 0 {
		return nil
	}
	if esdtData.TokenMetaData == nil {
		return nil
	}

	systemAcc, err := e.getSystemAccount()
	if err != nil {
		return err
	}

	currentSaveData, err := systemAcc.AccountDataHandler().RetrieveValue(esdtNFTTokenKey)
	if !isUpdate && len(currentSaveData) > 0 {
		return nil
	}

	esdtDataOnSystemAcc := &esdt.ESDigitalToken{
		Type:          esdtData.Type,
		Value:         big.NewInt(0),
		TokenMetaData: esdtData.TokenMetaData,
		Properties:    make([]byte, e.shardCoordinator.NumberOfShards()),
	}
	selfID := e.shardCoordinator.SelfId()
	if selfID != core.MetachainShardId {
		esdtDataOnSystemAcc.Properties[selfID] = 1
	}
	if senderShardID != core.MetachainShardId {
		esdtDataOnSystemAcc.Properties[senderShardID] = 1
	}

	marshaledData, err := e.marshalizer.Marshal(esdtDataOnSystemAcc)
	if err != nil {
		return err
	}

	return systemAcc.AccountDataHandler().SaveKeyValue(esdtNFTTokenKey, marshaledData)
}

func (e *esdtDataStorage) getSystemAccount() (vmcommon.UserAccountHandler, error) {
	systemSCAccount, err := e.accounts.LoadAccount(vmcommon.SystemAccountAddress)
	if err != nil {
		return nil, err
	}

	userAcc, ok := systemSCAccount.(vmcommon.UserAccountHandler)
	if !ok {
		return nil, ErrWrongTypeAssertion
	}

	return userAcc, nil
}

// WasAlreadySentToDestinationShard checks whether NFT metadata was sent to destination shard or not
// saves destination shard as pending until it is confirmed
func (e *esdtDataStorage) WasAlreadySentToDestinationShard(
	tickerID []byte,
	nonce uint64,
	dstAddress []byte,
) (bool, error) {
	if !e.flagSaveToSystemAccount.IsSet() {
		return false, nil
	}
	if nonce == 0 {
		return true, nil
	}
	dstShardID := e.shardCoordinator.ComputeId(dstAddress)
	if dstShardID == e.shardCoordinator.SelfId() {
		return true, nil
	}
	if dstShardID == core.MetachainShardId {
		return true, nil
	}

	esdtTokenKey := append(e.keyPrefix, tickerID...)
	esdtNFTTokenKey := computeESDTNFTTokenKey(esdtTokenKey, nonce)

	esdtData, systemAcc, err := e.getESDTDigitalTokenDataFromSystemAccount(esdtNFTTokenKey)
	if err != nil {
		return false, err
	}
	if esdtData == nil {
		return false, nil
	}

	if uint32(len(esdtData.Properties)) < e.shardCoordinator.NumberOfShards() {
		newVector := make([]byte, e.shardCoordinator.NumberOfShards())
		for i, val := range esdtData.Properties {
			newVector[i] = val
		}
		esdtData.Properties = newVector
	}

	if esdtData.Properties[dstShardID] > 0 {
		return true, nil
	}

	esdtData.Properties[dstShardID] = 1
	marshaledData, err := e.marshalizer.Marshal(esdtData)
	if err != nil {
		return false, err
	}

	err = systemAcc.AccountDataHandler().SaveKeyValue(esdtNFTTokenKey, marshaledData)
	return false, err
}

// SaveNFTMetaDataToSystemAccount this saves the NFT metadata to the system account even if there was an error in processing
func (e *esdtDataStorage) SaveNFTMetaDataToSystemAccount() {

}

// EpochConfirmed is called whenever a new epoch is confirmed
func (e *esdtDataStorage) EpochConfirmed(epoch uint32, _ uint64) {
	e.flagSaveToSystemAccount.Toggle(epoch >= e.saveToSystemEnableEpoch)
	log.Debug("ESDT NFT save to system account", "enabled", e.flagSaveToSystemAccount.IsSet())
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtDataStorage) IsInterfaceNil() bool {
	return e == nil
}
