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
	marshaller            vmcommon.Marshalizer
}

func (b *baseComponentsHolder) addNFTToDestination(
	sndAddress []byte,
	dstAddress []byte,
	userAccount vmcommon.UserAccountHandler,
	esdtDataToTransfer *esdt.ESDigitalToken,
	esdtTokenKey []byte,
	nonce uint64,
	isReturnWithError bool,
	isSenderESDTSCAddr bool,
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

	if isNew && !metaDataOnUserAccount(esdtDataToTransfer.Type) {
		esdtDataInSystemAcc, err := b.esdtStorageHandler.GetMetaDataFromSystemAccount(esdtTokenKey, nonce)
		if err != nil {
			return err
		}
		if esdtDataInSystemAcc != nil {
			currentESDTData.TokenMetaData = esdtDataInSystemAcc.TokenMetaData
			currentESDTData.Reserved = esdtDataInSystemAcc.Reserved
		}
	}

	latestEsdtData, err := getLatestMetaData(currentESDTData, esdtDataToTransfer, b.enableEpochsHandler, b.marshaller)
	if err != nil {
		return err
	}
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
	if !isSameShard || isSenderESDTSCAddr {
		err = b.esdtStorageHandler.AddToLiquiditySystemAcc(esdtTokenKey, latestEsdtData.Type, nonce, transferValue, false)
		if err != nil {
			return err
		}
	}

	return nil
}

func getLatestMetaData(currentEsdtData, transferEsdtData *esdt.ESDigitalToken, enableEpochsHandler vmcommon.EnableEpochsHandler, marshaller vmcommon.Marshalizer) (*esdt.ESDigitalToken, error) {
	if !enableEpochsHandler.IsFlagEnabled(DynamicEsdtFlag) {
		return transferEsdtData, nil
	}

	return mergeEsdtData(currentEsdtData, transferEsdtData, enableEpochsHandler, marshaller)
}

func mergeEsdtData(currentEsdtData, transferEsdtData *esdt.ESDigitalToken, enableEpochsHandler vmcommon.EnableEpochsHandler, marshaller vmcommon.Marshalizer) (*esdt.ESDigitalToken, error) {
	if currentEsdtData.TokenMetaData == nil {
		return transferEsdtData, nil
	}

	currentMetaDataVersion, wasCurrentMetaDataUpdated, err := getMetaDataVersion(currentEsdtData, enableEpochsHandler, marshaller)
	if err != nil {
		return nil, err
	}
	transferredMetaDataVersion, wasTransferMetaDataUpdated, err := getMetaDataVersion(transferEsdtData, enableEpochsHandler, marshaller)
	if err != nil {
		return nil, err
	}

	if !wasCurrentMetaDataUpdated && !wasTransferMetaDataUpdated {
		return transferEsdtData, nil
	}

	if currentMetaDataVersion.Name > transferredMetaDataVersion.Name {
		transferEsdtData.TokenMetaData.Name = currentEsdtData.TokenMetaData.Name
		transferredMetaDataVersion.Name = currentMetaDataVersion.Name
	}
	if currentMetaDataVersion.Creator > transferredMetaDataVersion.Creator {
		transferEsdtData.TokenMetaData.Creator = currentEsdtData.TokenMetaData.Creator
		transferredMetaDataVersion.Creator = currentMetaDataVersion.Creator
	}
	if currentMetaDataVersion.Royalties > transferredMetaDataVersion.Royalties {
		transferEsdtData.TokenMetaData.Royalties = currentEsdtData.TokenMetaData.Royalties
		transferredMetaDataVersion.Royalties = currentMetaDataVersion.Royalties
	}
	if currentMetaDataVersion.Hash > transferredMetaDataVersion.Hash {
		transferEsdtData.TokenMetaData.Hash = currentEsdtData.TokenMetaData.Hash
		transferredMetaDataVersion.Hash = currentMetaDataVersion.Hash
	}
	if currentMetaDataVersion.URIs > transferredMetaDataVersion.URIs {
		transferEsdtData.TokenMetaData.URIs = currentEsdtData.TokenMetaData.URIs
		transferredMetaDataVersion.URIs = currentMetaDataVersion.URIs
	}
	if currentMetaDataVersion.Attributes > transferredMetaDataVersion.Attributes {
		transferEsdtData.TokenMetaData.Attributes = currentEsdtData.TokenMetaData.Attributes
		transferredMetaDataVersion.Attributes = currentMetaDataVersion.Attributes
	}

	err = changeEsdtVersion(transferEsdtData, transferredMetaDataVersion, enableEpochsHandler, marshaller)
	if err != nil {
		return nil, err
	}

	return transferEsdtData, nil
}
