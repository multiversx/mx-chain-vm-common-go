package builtInFunctions

import (
	"bytes"
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
}

// ArgsNewESDTDataStorage defines the argument list for new esdt data storage handler
type ArgsNewESDTDataStorage struct {
	Accounts                vmcommon.AccountsAdapter
	GlobalSettingsHandler   vmcommon.ESDTGlobalSettingsHandler
	Marshalizer             vmcommon.Marshalizer
	SaveToSystemEnableEpoch uint32
	EpochNotifier           vmcommon.EpochNotifier
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

	e := &esdtDataStorage{
		accounts:                args.Accounts,
		globalSettingsHandler:   args.GlobalSettingsHandler,
		marshalizer:             args.Marshalizer,
		keyPrefix:               []byte(core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier),
		flagSaveToSystemAccount: atomic.Flag{},
		saveToSystemEnableEpoch: args.SaveToSystemEnableEpoch,
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
	esdtData := &esdt.ESDigitalToken{Value: big.NewInt(0), Type: uint32(core.Fungible)}
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

func (e *esdtDataStorage) getESDTMetaDataFromSystemAccount(
	tokenKey []byte,
) (*esdt.MetaData, error) {
	systemAcc, err := e.getSystemAccount()
	if err != nil {
		return nil, err
	}

	marshaledData, err := systemAcc.AccountDataHandler().RetrieveValue(tokenKey)
	if err != nil || len(marshaledData) == 0 {
		return nil, nil
	}

	esdtMetaData := &esdt.MetaData{}
	err = e.marshalizer.Unmarshal(esdtMetaData, marshaledData)
	if err != nil {
		return nil, err
	}

	return esdtMetaData, nil
}

// SaveESDTNFTToken saves the nft token to the account and system account
func (e *esdtDataStorage) SaveESDTNFTToken(
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

	err = e.saveESDTMetaDataToSystemAccount(esdtNFTTokenKey, nonce, esdtData.TokenMetaData)
	if err != nil {
		return nil, err
	}

	esdtData.TokenMetaData = nil
	marshaledData, err := e.marshalizer.Marshal(esdtData)
	if err != nil {
		return nil, err
	}

	return marshaledData, acnt.AccountDataHandler().SaveKeyValue(esdtNFTTokenKey, marshaledData)
}

func (e *esdtDataStorage) saveESDTMetaDataToSystemAccount(
	esdtNFTTokenKey []byte,
	nonce uint64,
	esdtMetaData *esdt.MetaData,
) error {
	if nonce == 0 {
		return nil
	}

	marshaledData, err := e.marshalizer.Marshal(esdtMetaData)
	if err != nil {
		return err
	}

	systemAcc, err := e.getSystemAccount()
	if err != nil {
		return err
	}

	currentSaveData, err := systemAcc.AccountDataHandler().RetrieveValue(esdtNFTTokenKey)
	if bytes.Equal(marshaledData, currentSaveData) {
		return nil
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

// EpochConfirmed is called whenever a new epoch is confirmed
func (e *esdtDataStorage) EpochConfirmed(epoch uint32, _ uint64) {
	e.flagSaveToSystemAccount.Toggle(epoch >= e.saveToSystemEnableEpoch)
	log.Debug("ESDT NFT save to system account", "enabled", e.flagSaveToSystemAccount.IsSet())
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtDataStorage) IsInterfaceNil() bool {
	return e == nil
}
