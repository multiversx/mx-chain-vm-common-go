package builtInFunctions

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/atomic"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go-core/data/esdt"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/parsers"
)

const existsOnShard = byte(1)

type esdtDataStorage struct {
	accounts              vmcommon.AccountsAdapter
	globalSettingsHandler vmcommon.ESDTGlobalSettingsHandler
	marshaller            vmcommon.Marshalizer
	keyPrefix             []byte
	shardCoordinator      vmcommon.Coordinator
	txDataParser          vmcommon.CallArgsParser

	flagSaveToSystemAccount          atomic.Flag
	saveToSystemEnableEpoch          uint32
	flagCheckFrozenCollection        atomic.Flag
	checkFrozenCollectionEnableEpoch uint32
	flagSendAlwaysEnableEpoch        atomic.Flag
	sendAlwaysEnableEpoch            uint32
	flagFixOldTokenLiquidity         atomic.Flag
	fixOldTokenLiquidityEnableEpoch  uint32
}

// ArgsNewESDTDataStorage defines the argument list for new esdt data storage handler
type ArgsNewESDTDataStorage struct {
	Accounts                        vmcommon.AccountsAdapter
	GlobalSettingsHandler           vmcommon.ESDTGlobalSettingsHandler
	Marshalizer                     vmcommon.Marshalizer
	SaveToSystemEnableEpoch         uint32
	SendAlwaysEnableEpoch           uint32
	FixOldTokenLiquidityEnableEpoch uint32
	EpochNotifier                   vmcommon.EpochNotifier
	ShardCoordinator                vmcommon.Coordinator
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
		accounts:              args.Accounts,
		globalSettingsHandler: args.GlobalSettingsHandler,
		marshaller:            args.Marshalizer,
		keyPrefix:             []byte(baseESDTKeyPrefix),
		shardCoordinator:      args.ShardCoordinator,
		txDataParser:          parsers.NewCallArgsParser(),

		flagSaveToSystemAccount:          atomic.Flag{},
		saveToSystemEnableEpoch:          args.SaveToSystemEnableEpoch,
		flagCheckFrozenCollection:        atomic.Flag{},
		checkFrozenCollectionEnableEpoch: args.SaveToSystemEnableEpoch,
		flagSendAlwaysEnableEpoch:        atomic.Flag{},
		sendAlwaysEnableEpoch:            args.SendAlwaysEnableEpoch,
		flagFixOldTokenLiquidity:         atomic.Flag{},
		fixOldTokenLiquidityEnableEpoch:  args.FixOldTokenLiquidityEnableEpoch,
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

	err = e.marshaller.Unmarshal(esdtData, marshaledData)
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
	err = e.marshaller.Unmarshal(esdtData, marshaledData)
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

// CheckCollectionIsFrozenForAccount returns
func (e *esdtDataStorage) checkCollectionIsFrozenForAccount(
	accnt vmcommon.UserAccountHandler,
	esdtTokenKey []byte,
	nonce uint64,
	isReturnWithError bool,
) error {
	if !e.flagCheckFrozenCollection.IsSet() {
		return nil
	}
	if nonce == 0 || isReturnWithError {
		return nil
	}

	esdtData := &esdt.ESDigitalToken{
		Value: big.NewInt(0),
		Type:  uint32(core.Fungible),
	}
	marshaledData, err := accnt.AccountDataHandler().RetrieveValue(esdtTokenKey)
	if err != nil || len(marshaledData) == 0 {
		return nil
	}

	err = e.marshaller.Unmarshal(esdtData, marshaledData)
	if err != nil {
		return err
	}

	esdtUserMetaData := ESDTUserMetadataFromBytes(esdtData.Properties)
	if esdtUserMetaData.Frozen {
		return ErrESDTIsFrozenForAccount
	}

	return nil
}

func (e *esdtDataStorage) checkFrozenPauseProperties(
	acnt vmcommon.UserAccountHandler,
	esdtTokenKey []byte,
	nonce uint64,
	esdtData *esdt.ESDigitalToken,
	isReturnWithError bool,
) error {
	err := checkFrozeAndPause(acnt.AddressBytes(), esdtTokenKey, esdtData, e.globalSettingsHandler, isReturnWithError)
	if err != nil {
		return err
	}

	esdtNFTTokenKey := computeESDTNFTTokenKey(esdtTokenKey, nonce)
	err = checkFrozeAndPause(acnt.AddressBytes(), esdtNFTTokenKey, esdtData, e.globalSettingsHandler, isReturnWithError)
	if err != nil {
		return err
	}

	err = e.checkCollectionIsFrozenForAccount(acnt, esdtTokenKey, nonce, isReturnWithError)
	if err != nil {
		return err
	}

	return nil
}

// AddToLiquiditySystemAcc will increase/decrease the liquidity for ESDT Tokens on the metadata
func (e *esdtDataStorage) AddToLiquiditySystemAcc(
	esdtTokenKey []byte,
	nonce uint64,
	transferValue *big.Int,
) error {
	if !e.flagSaveToSystemAccount.IsSet() || !e.flagSendAlwaysEnableEpoch.IsSet() || nonce == 0 {
		return nil
	}

	esdtNFTTokenKey := computeESDTNFTTokenKey(esdtTokenKey, nonce)
	esdtData, systemAcc, err := e.getESDTDigitalTokenDataFromSystemAccount(esdtNFTTokenKey)
	if err != nil {
		return err
	}

	if esdtData == nil {
		return ErrNilESDTData
	}

	// old style metaData - nothing to do
	if len(esdtData.Reserved) == 0 {
		return nil
	}

	if e.flagFixOldTokenLiquidity.IsSet() {
		// old tokens which were transferred intra shard before the activation of this flag
		if esdtData.Value.Cmp(zero) == 0 && transferValue.Cmp(zero) < 0 {
			esdtData.Reserved = nil
			return e.marshalAndSaveData(systemAcc, esdtData, esdtNFTTokenKey)
		}
	}

	esdtData.Value.Add(esdtData.Value, transferValue)
	if esdtData.Value.Cmp(zero) < 0 {
		return ErrInvalidLiquidityForESDT
	}

	if esdtData.Value.Cmp(zero) == 0 {
		err = systemAcc.AccountDataHandler().SaveKeyValue(esdtNFTTokenKey, nil)
		if err != nil {
			return err
		}

		return e.accounts.SaveAccount(systemAcc)
	}

	err = e.marshalAndSaveData(systemAcc, esdtData, esdtNFTTokenKey)
	if err != nil {
		return err
	}

	return nil
}

// SaveESDTNFTToken saves the nft token to the account and system account
func (e *esdtDataStorage) SaveESDTNFTToken(
	senderAddress []byte,
	acnt vmcommon.UserAccountHandler,
	esdtTokenKey []byte,
	nonce uint64,
	esdtData *esdt.ESDigitalToken,
	mustUpdate bool,
	isReturnWithError bool,
) ([]byte, error) {
	err := e.checkFrozenPauseProperties(acnt, esdtTokenKey, nonce, esdtData, isReturnWithError)
	if err != nil {
		return nil, err
	}

	esdtNFTTokenKey := computeESDTNFTTokenKey(esdtTokenKey, nonce)
	senderShardID := e.shardCoordinator.ComputeId(senderAddress)
	if e.flagSaveToSystemAccount.IsSet() {
		err = e.saveESDTMetaDataToSystemAccount(acnt, senderShardID, esdtNFTTokenKey, nonce, esdtData, mustUpdate)
		if err != nil {
			return nil, err
		}
	}

	if esdtData.Value.Cmp(zero) <= 0 {
		return nil, acnt.AccountDataHandler().SaveKeyValue(esdtNFTTokenKey, nil)
	}

	if !e.flagSaveToSystemAccount.IsSet() {
		marshaledData, err := e.marshaller.Marshal(esdtData)
		if err != nil {
			return nil, err
		}

		return marshaledData, acnt.AccountDataHandler().SaveKeyValue(esdtNFTTokenKey, marshaledData)
	}

	esdtDataOnAccount := &esdt.ESDigitalToken{
		Type:       esdtData.Type,
		Value:      esdtData.Value,
		Properties: esdtData.Properties,
	}
	marshaledData, err := e.marshaller.Marshal(esdtDataOnAccount)
	if err != nil {
		return nil, err
	}

	return marshaledData, acnt.AccountDataHandler().SaveKeyValue(esdtNFTTokenKey, marshaledData)
}

func (e *esdtDataStorage) saveESDTMetaDataToSystemAccount(
	userAcc vmcommon.UserAccountHandler,
	senderShardID uint32,
	esdtNFTTokenKey []byte,
	nonce uint64,
	esdtData *esdt.ESDigitalToken,
	mustUpdate bool,
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
	if !mustUpdate && len(currentSaveData) > 0 {
		return nil
	}

	esdtDataOnSystemAcc := &esdt.ESDigitalToken{
		Type:          esdtData.Type,
		Value:         big.NewInt(0),
		TokenMetaData: esdtData.TokenMetaData,
		Properties:    make([]byte, e.shardCoordinator.NumberOfShards()),
	}
	if len(currentSaveData) == 0 && e.flagSendAlwaysEnableEpoch.IsSet() {
		esdtDataOnSystemAcc.Properties = nil
		esdtDataOnSystemAcc.Reserved = []byte{1}

		err = e.setReservedToNilForOldToken(esdtDataOnSystemAcc, userAcc, esdtNFTTokenKey)
		if err != nil {
			return err
		}
	}

	if !e.flagSendAlwaysEnableEpoch.IsSet() {
		selfID := e.shardCoordinator.SelfId()
		if selfID != core.MetachainShardId {
			esdtDataOnSystemAcc.Properties[selfID] = existsOnShard
		}
		if senderShardID != core.MetachainShardId {
			esdtDataOnSystemAcc.Properties[senderShardID] = existsOnShard
		}
	}

	return e.marshalAndSaveData(systemAcc, esdtDataOnSystemAcc, esdtNFTTokenKey)
}

func (e *esdtDataStorage) setReservedToNilForOldToken(
	esdtDataOnSystemAcc *esdt.ESDigitalToken,
	userAcc vmcommon.UserAccountHandler,
	esdtNFTTokenKey []byte,
) error {
	if !e.flagFixOldTokenLiquidity.IsSet() {
		return nil
	}

	if check.IfNil(userAcc) {
		return ErrNilUserAccount
	}
	dataOnUserAcc, errNotCritical := userAcc.AccountDataHandler().RetrieveValue(esdtNFTTokenKey)
	shouldIgnoreToken := errNotCritical != nil || len(dataOnUserAcc) == 0
	if shouldIgnoreToken {
		return nil
	}

	esdtDataOnUserAcc := &esdt.ESDigitalToken{}
	err := e.marshaller.Unmarshal(esdtDataOnUserAcc, dataOnUserAcc)
	if err != nil {
		return err
	}

	// tokens which were last moved before flagOptimizeNFTStore keep the esdt metaData on the user account
	// these are not compatible with the new liquidity model,so we set the reserved field to nil
	if esdtDataOnUserAcc.TokenMetaData != nil {
		esdtDataOnSystemAcc.Reserved = nil
	}

	return nil
}

func (e *esdtDataStorage) marshalAndSaveData(
	systemAcc vmcommon.UserAccountHandler,
	esdtData *esdt.ESDigitalToken,
	esdtNFTTokenKey []byte,
) error {
	marshaledData, err := e.marshaller.Marshal(esdtData)
	if err != nil {
		return err
	}

	err = systemAcc.AccountDataHandler().SaveKeyValue(esdtNFTTokenKey, marshaledData)
	if err != nil {
		return err
	}

	return e.accounts.SaveAccount(systemAcc)
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

//TODO: merge properties in case of shard merge

// WasAlreadySentToDestinationShardAndUpdateState checks whether NFT metadata was sent to destination shard or not
// and saves the destination shard as sent
func (e *esdtDataStorage) WasAlreadySentToDestinationShardAndUpdateState(
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

	if e.flagSendAlwaysEnableEpoch.IsSet() {
		return false, nil
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
		newSlice := make([]byte, e.shardCoordinator.NumberOfShards())
		for i, val := range esdtData.Properties {
			newSlice[i] = val
		}
		esdtData.Properties = newSlice
	}

	if esdtData.Properties[dstShardID] > 0 {
		return true, nil
	}

	esdtData.Properties[dstShardID] = existsOnShard
	return false, e.marshalAndSaveData(systemAcc, esdtData, esdtNFTTokenKey)
}

// SaveNFTMetaDataToSystemAccount this saves the NFT metadata to the system account even if there was an error in processing
func (e *esdtDataStorage) SaveNFTMetaDataToSystemAccount(
	tx data.TransactionHandler,
) error {
	if !e.flagSaveToSystemAccount.IsSet() {
		return nil
	}
	if e.flagSendAlwaysEnableEpoch.IsSet() {
		return nil
	}
	if check.IfNil(tx) {
		return ErrNilTransactionHandler
	}

	sndShardID := e.shardCoordinator.ComputeId(tx.GetSndAddr())
	dstShardID := e.shardCoordinator.ComputeId(tx.GetRcvAddr())
	isCrossShardTxAtDest := sndShardID != dstShardID && e.shardCoordinator.SelfId() == dstShardID
	if !isCrossShardTxAtDest {
		return nil
	}

	function, arguments, err := e.txDataParser.ParseData(string(tx.GetData()))
	if err != nil {
		return nil
	}
	if len(arguments) < 4 {
		return nil
	}

	switch function {
	case core.BuiltInFunctionESDTNFTTransfer:
		return e.addMetaDataToSystemAccountFromNFTTransfer(sndShardID, arguments)
	case core.BuiltInFunctionMultiESDTNFTTransfer:
		return e.addMetaDataToSystemAccountFromMultiTransfer(sndShardID, arguments)
	default:
		return nil
	}
}

func (e *esdtDataStorage) addMetaDataToSystemAccountFromNFTTransfer(
	sndShardID uint32,
	arguments [][]byte,
) error {
	if !bytes.Equal(arguments[3], zeroByteArray) {
		esdtTransferData := &esdt.ESDigitalToken{}
		err := e.marshaller.Unmarshal(esdtTransferData, arguments[3])
		if err != nil {
			return err
		}
		esdtTokenKey := append(e.keyPrefix, arguments[0]...)
		nonce := big.NewInt(0).SetBytes(arguments[1]).Uint64()
		esdtNFTTokenKey := computeESDTNFTTokenKey(esdtTokenKey, nonce)

		return e.saveESDTMetaDataToSystemAccount(nil, sndShardID, esdtNFTTokenKey, nonce, esdtTransferData, true)
	}
	return nil
}

func (e *esdtDataStorage) addMetaDataToSystemAccountFromMultiTransfer(
	sndShardID uint32,
	arguments [][]byte,
) error {
	numOfTransfers := big.NewInt(0).SetBytes(arguments[0]).Uint64()
	if numOfTransfers == 0 {
		return fmt.Errorf("%w, 0 tokens to transfer", ErrInvalidArguments)
	}
	minNumOfArguments := numOfTransfers*argumentsPerTransfer + 1
	if uint64(len(arguments)) < minNumOfArguments {
		return fmt.Errorf("%w, invalid number of arguments", ErrInvalidArguments)
	}

	startIndex := uint64(1)
	for i := uint64(0); i < numOfTransfers; i++ {
		tokenStartIndex := startIndex + i*argumentsPerTransfer
		tokenID := arguments[tokenStartIndex]
		nonce := big.NewInt(0).SetBytes(arguments[tokenStartIndex+1]).Uint64()

		if nonce > 0 && len(arguments[tokenStartIndex+2]) > vmcommon.MaxLengthForValueToOptTransfer {
			esdtTransferData := &esdt.ESDigitalToken{}
			marshaledNFTTransfer := arguments[tokenStartIndex+2]
			err := e.marshaller.Unmarshal(esdtTransferData, marshaledNFTTransfer)
			if err != nil {
				return fmt.Errorf("%w for token %s", err, string(tokenID))
			}

			esdtTokenKey := append(e.keyPrefix, tokenID...)
			esdtNFTTokenKey := computeESDTNFTTokenKey(esdtTokenKey, nonce)
			err = e.saveESDTMetaDataToSystemAccount(nil, sndShardID, esdtNFTTokenKey, nonce, esdtTransferData, true)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// EpochConfirmed is called whenever a new epoch is confirmed
func (e *esdtDataStorage) EpochConfirmed(epoch uint32, _ uint64) {
	e.flagSaveToSystemAccount.SetValue(epoch >= e.saveToSystemEnableEpoch)
	log.Debug("ESDT NFT save to system account", "enabled", e.flagSaveToSystemAccount.IsSet())

	e.flagCheckFrozenCollection.SetValue(epoch >= e.checkFrozenCollectionEnableEpoch)
	log.Debug("ESDT NFT check frozen collection", "enabled", e.flagCheckFrozenCollection.IsSet())

	e.flagSendAlwaysEnableEpoch.SetValue(epoch >= e.sendAlwaysEnableEpoch)
	log.Debug("ESDT send metadata always", "enabled", e.flagSendAlwaysEnableEpoch.IsSet())

	e.flagFixOldTokenLiquidity.SetValue(epoch >= e.fixOldTokenLiquidityEnableEpoch)
	log.Debug("ESDT fix old token liquidity", "enabled", e.flagFixOldTokenLiquidity.IsSet())
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtDataStorage) IsInterfaceNil() bool {
	return e == nil
}
