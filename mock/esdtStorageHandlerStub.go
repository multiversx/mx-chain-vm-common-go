package mock

import (
	"math/big"

	"github.com/multiversx/mx-chain-core-go/data"
	"github.com/multiversx/mx-chain-core-go/data/esdt"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

// ESDTNFTStorageHandlerStub -
type ESDTNFTStorageHandlerStub struct {
	SaveESDTNFTTokenCalled                                    func(senderAddress []byte, acnt vmcommon.UserAccountHandler, esdtTokenKey []byte, nonce uint64, esdtData *esdt.ESDigitalToken, mustUpdateAllFields bool, isReturnWithError bool) ([]byte, error)
	GetESDTNFTTokenOnSenderCalled                             func(acnt vmcommon.UserAccountHandler, esdtTokenKey []byte, nonce uint64) (*esdt.ESDigitalToken, error)
	GetESDTNFTTokenOnDestinationCalled                        func(acnt vmcommon.UserAccountHandler, esdtTokenKey []byte, nonce uint64) (*esdt.ESDigitalToken, bool, error)
	GetESDTNFTTokenOnDestinationWithCustomSystemAccountCalled func(accnt vmcommon.UserAccountHandler, esdtTokenKey []byte, nonce uint64, systemAccount vmcommon.UserAccountHandler) (*esdt.ESDigitalToken, bool, error)
	WasAlreadySentToDestinationShardAndUpdateStateCalled      func(tickerID []byte, nonce uint64, dstAddress []byte) (bool, error)
	SaveNFTMetaDataCalled                                     func(tx data.TransactionHandler) error
	AddToLiquiditySystemAccCalled                             func(esdtTokenKey []byte, tokenType uint32, nonce uint64, transferValue *big.Int) error
	GetMetaDataFromSystemAccountCalled                        func([]byte, uint64) (*esdt.MetaData, error)
	SaveMetaDataToSystemAccountCalled                         func(tokenKey []byte, nonce uint64, esdtData *esdt.ESDigitalToken) error
}

// SaveESDTNFTToken -
func (stub *ESDTNFTStorageHandlerStub) SaveESDTNFTToken(senderAddress []byte, acnt vmcommon.UserAccountHandler, esdtTokenKey []byte, nonce uint64, esdtData *esdt.ESDigitalToken, mustUpdateAllFields bool, isReturnWithError bool) ([]byte, error) {
	if stub.SaveESDTNFTTokenCalled != nil {
		return stub.SaveESDTNFTTokenCalled(senderAddress, acnt, esdtTokenKey, nonce, esdtData, mustUpdateAllFields, isReturnWithError)
	}
	return nil, nil
}

// GetESDTNFTTokenOnSender -
func (stub *ESDTNFTStorageHandlerStub) GetESDTNFTTokenOnSender(acnt vmcommon.UserAccountHandler, esdtTokenKey []byte, nonce uint64) (*esdt.ESDigitalToken, error) {
	if stub.GetESDTNFTTokenOnSenderCalled != nil {
		return stub.GetESDTNFTTokenOnSenderCalled(acnt, esdtTokenKey, nonce)
	}
	return nil, nil
}

// GetMetaDataFromSystemAccount -
func (stub *ESDTNFTStorageHandlerStub) GetMetaDataFromSystemAccount(key []byte, nonce uint64) (*esdt.MetaData, error) {
	if stub.GetMetaDataFromSystemAccountCalled != nil {
		return stub.GetMetaDataFromSystemAccountCalled(key, nonce)
	}
	return nil, nil
}

// SaveMetaDataToSystemAccount -
func (stub *ESDTNFTStorageHandlerStub) SaveMetaDataToSystemAccount(tokenKey []byte, nonce uint64, esdtData *esdt.ESDigitalToken) error {
	if stub.SaveMetaDataToSystemAccountCalled != nil {
		return stub.SaveMetaDataToSystemAccountCalled(tokenKey, nonce, esdtData)
	}
	return nil
}

// GetESDTNFTTokenOnDestination -
func (stub *ESDTNFTStorageHandlerStub) GetESDTNFTTokenOnDestination(acnt vmcommon.UserAccountHandler, esdtTokenKey []byte, nonce uint64) (*esdt.ESDigitalToken, bool, error) {
	if stub.GetESDTNFTTokenOnDestinationCalled != nil {
		return stub.GetESDTNFTTokenOnDestinationCalled(acnt, esdtTokenKey, nonce)
	}
	return nil, false, nil
}

// GetESDTNFTTokenOnDestinationWithCustomSystemAccount -
func (stub *ESDTNFTStorageHandlerStub) GetESDTNFTTokenOnDestinationWithCustomSystemAccount(accnt vmcommon.UserAccountHandler, esdtTokenKey []byte, nonce uint64, systemAccount vmcommon.UserAccountHandler) (*esdt.ESDigitalToken, bool, error) {
	if stub.GetESDTNFTTokenOnDestinationWithCustomSystemAccountCalled != nil {
		return stub.GetESDTNFTTokenOnDestinationWithCustomSystemAccountCalled(accnt, esdtTokenKey, nonce, systemAccount)
	}
	return nil, false, nil
}

// WasAlreadySentToDestinationShardAndUpdateState -
func (stub *ESDTNFTStorageHandlerStub) WasAlreadySentToDestinationShardAndUpdateState(tickerID []byte, nonce uint64, dstAddress []byte) (bool, error) {
	if stub.WasAlreadySentToDestinationShardAndUpdateStateCalled != nil {
		return stub.WasAlreadySentToDestinationShardAndUpdateStateCalled(tickerID, nonce, dstAddress)
	}
	return false, nil
}

// SaveNFTMetaData -
func (stub *ESDTNFTStorageHandlerStub) SaveNFTMetaData(tx data.TransactionHandler) error {
	if stub.SaveNFTMetaDataCalled != nil {
		return stub.SaveNFTMetaDataCalled(tx)
	}
	return nil
}

// AddToLiquiditySystemAcc -
func (stub *ESDTNFTStorageHandlerStub) AddToLiquiditySystemAcc(esdtTokenKey []byte, tokenType uint32, nonce uint64, transferValue *big.Int) error {
	if stub.AddToLiquiditySystemAccCalled != nil {
		return stub.AddToLiquiditySystemAccCalled(esdtTokenKey, tokenType, nonce, transferValue)
	}
	return nil
}

// IsInterfaceNil -
func (stub *ESDTNFTStorageHandlerStub) IsInterfaceNil() bool {
	return stub == nil
}
