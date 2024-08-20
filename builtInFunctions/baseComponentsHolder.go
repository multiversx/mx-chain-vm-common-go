package builtInFunctions

import (
	"errors"
	"math/big"

	"github.com/multiversx/mx-chain-core-go/data/esdt"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

type baseComponentsHolder struct {
	esdtStorageHandler    vmcommon.ESDTNFTStorageHandler
	globalSettingsHandler vmcommon.GlobalMetadataHandler
	shardCoordinator      vmcommon.Coordinator
	enableEpochsHandler   vmcommon.EnableEpochsHandler
}

func (b *baseComponentsHolder) addNFTToDestination(
	sndAddress []byte,
	dstAddress []byte,
	userAccount vmcommon.UserAccountHandler,
	esdtDataToTransfer *esdt.ESDigitalToken,
	esdtTokenKey []byte,
	nonce uint64,
	isReturnWithError bool,
) error {
	currentESDTData, isNew, err := b.esdtStorageHandler.GetESDTNFTTokenOnDestination(userAccount, esdtTokenKey, nonce)
	if err != nil && !errors.Is(err, ErrNFTTokenDoesNotExist) {
		return err
	}
	err = checkFrozeAndPause(dstAddress, esdtTokenKey, currentESDTData, b.globalSettingsHandler, isReturnWithError)
	if err != nil {
		return err
	}

	transferValue := big.NewInt(0).Set(esdtDataToTransfer.Value)
	esdtDataToTransfer.Value.Add(esdtDataToTransfer.Value, currentESDTData.Value)

	latestEsdtData := esdtDataToTransfer
	if !isNew {
		latestEsdtData = getLatestEsdtData(currentESDTData, esdtDataToTransfer, b.enableEpochsHandler)
	}
	latestEsdtData.Value.Set(esdtDataToTransfer.Value)

	_, err = b.esdtStorageHandler.SaveESDTNFTToken(sndAddress, userAccount, esdtTokenKey, nonce, latestEsdtData, false, isReturnWithError)
	if err != nil {
		return err
	}

	isSameShard := b.shardCoordinator.SameShard(sndAddress, dstAddress)
	if !isSameShard {
		err = b.esdtStorageHandler.AddToLiquiditySystemAcc(esdtTokenKey, latestEsdtData.Type, nonce, transferValue, false)
		if err != nil {
			return err
		}
	}

	return nil
}

func getLatestEsdtData(currentEsdtData, transferEsdtData *esdt.ESDigitalToken, enableEpochsHandler vmcommon.EnableEpochsHandler) *esdt.ESDigitalToken {
	if !enableEpochsHandler.IsFlagEnabled(DynamicEsdtFlag) {
		return transferEsdtData
	}

	currentEsdtDataVersion := big.NewInt(0).SetBytes(currentEsdtData.Reserved).Uint64()
	transferEsdtDataVersion := big.NewInt(0).SetBytes(transferEsdtData.Reserved).Uint64()

	if currentEsdtDataVersion > transferEsdtDataVersion {
		return currentEsdtData
	}

	return mergeEsdtData(currentEsdtData, transferEsdtData)
}

func mergeEsdtData(currentEsdtData, transferEsdtData *esdt.ESDigitalToken) *esdt.ESDigitalToken {
	if currentEsdtData.TokenMetaData == nil {
		currentEsdtData.TokenMetaData = &esdt.MetaData{}
	}
	currentEsdtData.Reserved = transferEsdtData.Reserved

	if transferEsdtData.TokenMetaData.Nonce > 0 {
		currentEsdtData.TokenMetaData.Nonce = transferEsdtData.TokenMetaData.Nonce
	}
	if len(transferEsdtData.TokenMetaData.Name) != 0 {
		currentEsdtData.TokenMetaData.Name = transferEsdtData.TokenMetaData.Name
	}
	if len(transferEsdtData.TokenMetaData.Creator) != 0 {
		currentEsdtData.TokenMetaData.Creator = transferEsdtData.TokenMetaData.Creator
	}
	if transferEsdtData.TokenMetaData.Royalties > 0 {
		currentEsdtData.TokenMetaData.Royalties = transferEsdtData.TokenMetaData.Royalties
	}
	if len(transferEsdtData.TokenMetaData.Hash) != 0 {
		currentEsdtData.TokenMetaData.Hash = transferEsdtData.TokenMetaData.Hash
	}
	if len(transferEsdtData.TokenMetaData.URIs) != 0 {
		currentEsdtData.TokenMetaData.URIs = transferEsdtData.TokenMetaData.URIs
	}
	if len(transferEsdtData.TokenMetaData.Attributes) != 0 {
		currentEsdtData.TokenMetaData.Attributes = transferEsdtData.TokenMetaData.Attributes
	}

	return currentEsdtData
}
