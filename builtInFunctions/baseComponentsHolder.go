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
	currentESDTData, _, err := b.esdtStorageHandler.GetESDTNFTTokenOnDestination(userAccount, esdtTokenKey, nonce)
	if err != nil && !errors.Is(err, ErrNFTTokenDoesNotExist) {
		return err
	}
	err = checkFrozeAndPause(dstAddress, esdtTokenKey, currentESDTData, b.globalSettingsHandler, isReturnWithError)
	if err != nil {
		return err
	}

	transferValue := big.NewInt(0).Set(esdtDataToTransfer.Value)
	esdtDataToTransfer.Value.Add(esdtDataToTransfer.Value, currentESDTData.Value)

	latestEsdtData := getLatestEsdtData(currentESDTData, esdtDataToTransfer, b.enableEpochsHandler)
	latestEsdtData.Value.Set(esdtDataToTransfer.Value)

	properties := vmcommon.NftSaveArgs{
		MustUpdateAllFields:         false,
		IsReturnWithError:           isReturnWithError,
		KeepMetaDataOnZeroLiquidity: false,
	}

	_, err = b.esdtStorageHandler.SaveESDTNFTToken(sndAddress, userAccount, esdtTokenKey, nonce, latestEsdtData, properties)
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

	if transferEsdtDataVersion >= currentEsdtDataVersion {
		return transferEsdtData
	}

	return mergeEsdtData(currentEsdtData, transferEsdtData)
}

func mergeEsdtData(currentEsdtData, transferEsdtData *esdt.ESDigitalToken) *esdt.ESDigitalToken {
	if currentEsdtData.TokenMetaData == nil {
		return transferEsdtData
	}

	transferEsdtData.Reserved = currentEsdtData.Reserved
	transferEsdtData.Type = currentEsdtData.Type

	wasAnyFieldUpdated := false
	hadNilTokenMetaData := false
	if transferEsdtData.TokenMetaData == nil {
		hadNilTokenMetaData = true
		transferEsdtData.TokenMetaData = &esdt.MetaData{}
	}

	if currentEsdtData.TokenMetaData.Nonce > 0 {
		wasAnyFieldUpdated = true
		transferEsdtData.TokenMetaData.Nonce = currentEsdtData.TokenMetaData.Nonce
	}
	if len(currentEsdtData.TokenMetaData.Name) != 0 {
		wasAnyFieldUpdated = true
		transferEsdtData.TokenMetaData.Name = currentEsdtData.TokenMetaData.Name
	}
	if len(currentEsdtData.TokenMetaData.Creator) != 0 {
		wasAnyFieldUpdated = true
		transferEsdtData.TokenMetaData.Creator = currentEsdtData.TokenMetaData.Creator
	}
	if currentEsdtData.TokenMetaData.Royalties > 0 {
		wasAnyFieldUpdated = true
		transferEsdtData.TokenMetaData.Royalties = currentEsdtData.TokenMetaData.Royalties
	}
	if len(currentEsdtData.TokenMetaData.Hash) != 0 {
		wasAnyFieldUpdated = true
		transferEsdtData.TokenMetaData.Hash = currentEsdtData.TokenMetaData.Hash
	}
	if len(currentEsdtData.TokenMetaData.URIs) != 0 {
		wasAnyFieldUpdated = true
		transferEsdtData.TokenMetaData.URIs = currentEsdtData.TokenMetaData.URIs
	}
	if len(currentEsdtData.TokenMetaData.Attributes) != 0 {
		wasAnyFieldUpdated = true
		transferEsdtData.TokenMetaData.Attributes = currentEsdtData.TokenMetaData.Attributes
	}

	if !wasAnyFieldUpdated && hadNilTokenMetaData {
		transferEsdtData.TokenMetaData = nil
	}

	return transferEsdtData
}
