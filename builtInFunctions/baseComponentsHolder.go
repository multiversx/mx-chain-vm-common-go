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

	latestEsdtData := getLatestEsdtData(currentESDTData, esdtDataToTransfer)
	latestEsdtData.Value.Set(esdtDataToTransfer.Value)

	_, err = b.esdtStorageHandler.SaveESDTNFTToken(sndAddress, userAccount, esdtTokenKey, nonce, esdtDataToTransfer, false, isReturnWithError)
	if err != nil {
		return err
	}

	isSameShard := b.shardCoordinator.SameShard(sndAddress, dstAddress)
	if !isSameShard {
		err = b.esdtStorageHandler.AddToLiquiditySystemAcc(esdtTokenKey, nonce, transferValue)
		if err != nil {
			return err
		}
	}

	return nil
}

func getLatestEsdtData(currentEsdtData, transferEsdtData *esdt.ESDigitalToken) *esdt.ESDigitalToken {
	currentEsdtDataVersion := big.NewInt(0).SetBytes(currentEsdtData.Reserved).Uint64()
	transferEsdtDataVersion := big.NewInt(0).SetBytes(transferEsdtData.Reserved).Uint64()

	if currentEsdtDataVersion > transferEsdtDataVersion {
		return currentEsdtData
	}

	return transferEsdtData
}
