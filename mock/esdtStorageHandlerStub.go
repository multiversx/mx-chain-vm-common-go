package mock

import (
	"math/big"

	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go-core/data/esdt"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

// ESDTNFTStorageHandlerStub -
type ESDTNFTStorageHandlerStub struct {
	SaveESDTNFTTokenCalled                                    func(senderAddress []byte, acnt vmcommon.UserAccountHandler, esdtTokenKey []byte, nonce uint64, esdtData *esdt.ESDigitalToken, isCreation bool, isReturnWithError bool) ([]byte, error)
	GetESDTNFTTokenOnSenderCalled                             func(acnt vmcommon.UserAccountHandler, esdtTokenKey []byte, nonce uint64) (*esdt.ESDigitalToken, error)
	GetESDTNFTTokenOnDestinationCalled                        func(acnt vmcommon.UserAccountHandler, esdtTokenKey []byte, nonce uint64) (*esdt.ESDigitalToken, bool, error)
	GetESDTNFTTokenOnDestinationWithCustomSystemAccountCalled func(accnt vmcommon.UserAccountHandler, esdtTokenKey []byte, nonce uint64, systemAccount vmcommon.UserAccountHandler) (*esdt.ESDigitalToken, bool, error)
	WasAlreadySentToDestinationShardAndUpdateStateCalled      func(tickerID []byte, nonce uint64, dstAddress []byte) (bool, error)
	SaveNFTMetaDataToSystemAccountCalled                      func(tx data.TransactionHandler) error
	AddToLiquiditySystemAccCalled                             func(esdtTokenKey []byte, nonce uint64, transferValue *big.Int) error
}

// SaveESDTNFTToken -
func (stub *ESDTNFTStorageHandlerStub) SaveESDTNFTToken(senderAddress []byte, acnt vmcommon.UserAccountHandler, esdtTokenKey []byte, nonce uint64, esdtData *esdt.ESDigitalToken, isCreation bool, isReturnWithError bool) ([]byte, error) {
	if stub.SaveESDTNFTTokenCalled != nil {
		return stub.SaveESDTNFTTokenCalled(senderAddress, acnt, esdtTokenKey, nonce, esdtData, isCreation, isReturnWithError)
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

// SaveNFTMetaDataToSystemAccount -
func (stub *ESDTNFTStorageHandlerStub) SaveNFTMetaDataToSystemAccount(tx data.TransactionHandler) error {
	if stub.SaveNFTMetaDataToSystemAccountCalled != nil {
		return stub.SaveNFTMetaDataToSystemAccountCalled(tx)
	}
	return nil
}

// AddToLiquiditySystemAcc -
func (stub *ESDTNFTStorageHandlerStub) AddToLiquiditySystemAcc(esdtTokenKey []byte, nonce uint64, transferValue *big.Int) error {
	if stub.AddToLiquiditySystemAccCalled != nil {
		return stub.AddToLiquiditySystemAccCalled(esdtTokenKey, nonce, transferValue)
	}
	return nil
}

// IsInterfaceNil -
func (stub *ESDTNFTStorageHandlerStub) IsInterfaceNil() bool {
	return stub == nil
}
