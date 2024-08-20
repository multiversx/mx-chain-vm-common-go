package builtInFunctions

import (
	"math/big"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/data/esdt"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/assert"
)

func TestBaseComponentsHolder_addNFTToDestination(t *testing.T) {
	t.Parallel()

	t.Run("different shards should save liquidity to system account", func(t *testing.T) {
		t.Parallel()

		saveCalled := false
		addToLiquiditySystemAccCalled := false
		b := &baseComponentsHolder{
			esdtStorageHandler: &mock.ESDTNFTStorageHandlerStub{
				GetESDTNFTTokenOnDestinationCalled: func(_ vmcommon.UserAccountHandler, _ []byte, _ uint64) (*esdt.ESDigitalToken, bool, error) {
					return &esdt.ESDigitalToken{
						Value: big.NewInt(100),
					}, false, nil
				},
				SaveESDTNFTTokenCalled: func(_ []byte, _ vmcommon.UserAccountHandler, _ []byte, _ uint64, esdtData *esdt.ESDigitalToken, mustUpdateAllFields bool, isReturnWithError bool) ([]byte, error) {
					assert.Equal(t, big.NewInt(200), esdtData.Value)
					saveCalled = true
					return nil, nil
				},
				AddToLiquiditySystemAccCalled: func(esdtTokenKey []byte, _ uint32, nonce uint64, transferValue *big.Int, _ bool) error {
					assert.Equal(t, big.NewInt(100), transferValue)
					addToLiquiditySystemAccCalled = true
					return nil
				},
			},
			globalSettingsHandler: &mock.GlobalSettingsHandlerStub{
				IsPausedCalled: func(_ []byte) bool {
					return false
				},
			},
			shardCoordinator: &mock.ShardCoordinatorStub{
				SameShardCalled: func(_, _ []byte) bool {
					return false
				},
			},
			enableEpochsHandler: &mock.EnableEpochsHandlerStub{},
		}

		acc := &mock.UserAccountStub{}
		esdtDataToTransfer := &esdt.ESDigitalToken{
			Type:       0,
			Value:      big.NewInt(100),
			Properties: make([]byte, 0),
		}
		err := b.addNFTToDestination([]byte("sndAddr"), []byte("dstAddr"), acc, esdtDataToTransfer, []byte("esdtTokenKey"), 0, false)
		assert.Nil(t, err)
		assert.True(t, addToLiquiditySystemAccCalled)
		assert.True(t, saveCalled)
	})
}

func TestBaseComponentsHolder_getLatestEsdtData(t *testing.T) {
	t.Parallel()

	t.Run("flag disabled should return transfer esdt data", func(t *testing.T) {
		t.Parallel()

		enableEpochsHandler := &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(_ core.EnableEpochFlag) bool {
				return false
			},
		}
		currentEsdtData := &esdt.ESDigitalToken{
			Reserved: []byte{1},
			Value:    big.NewInt(100),
		}
		transferEsdtData := &esdt.ESDigitalToken{
			Reserved: []byte{2},
			Value:    big.NewInt(200),
		}

		latestEsdtData := getLatestEsdtData(currentEsdtData, transferEsdtData, enableEpochsHandler)
		assert.Equal(t, transferEsdtData, latestEsdtData)
	})
	t.Run("flag enabled and current esdt data version is higher should return current esdt data", func(t *testing.T) {
		t.Parallel()
		enableEpochsHandler := &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(_ core.EnableEpochFlag) bool {
				return true
			},
		}
		currentEsdtData := &esdt.ESDigitalToken{
			Reserved: []byte{2},
			Value:    big.NewInt(100),
		}
		transferEsdtData := &esdt.ESDigitalToken{
			Reserved: []byte{1},
			Value:    big.NewInt(200),
		}

		latestEsdtData := getLatestEsdtData(currentEsdtData, transferEsdtData, enableEpochsHandler)
		assert.Equal(t, currentEsdtData, latestEsdtData)
	})
	t.Run("flag enabled and transfer esdt data version is higher should merge with transfer esdt data", func(t *testing.T) {
		t.Parallel()

		enableEpochsHandler := &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(_ core.EnableEpochFlag) bool {
				return true
			},
		}

		name := []byte("name")
		creator := []byte("creator")
		newCreator := []byte("newCreator")
		royalties := uint32(25)
		newRoyalties := uint32(11)
		hash := []byte("hash")
		uris := [][]byte{[]byte("uri1"), []byte("uri2")}
		attributes := []byte("attributes")
		newAttributes := []byte("newAttributes")
		currentEsdtData := &esdt.ESDigitalToken{
			Reserved: []byte{1},
			TokenMetaData: &esdt.MetaData{
				Nonce:      0,
				Name:       name,
				Creator:    creator,
				Royalties:  royalties,
				Hash:       hash,
				URIs:       uris,
				Attributes: attributes,
			},
		}
		transferEsdtData := &esdt.ESDigitalToken{
			Reserved: []byte{2},
			TokenMetaData: &esdt.MetaData{
				Creator:    newCreator,
				Royalties:  newRoyalties,
				Attributes: newAttributes,
			},
		}

		latestEsdtData := getLatestEsdtData(currentEsdtData, transferEsdtData, enableEpochsHandler)
		assert.Equal(t, []byte{2}, latestEsdtData.Reserved)
		assert.Equal(t, newCreator, latestEsdtData.TokenMetaData.Creator)
		assert.Equal(t, newRoyalties, latestEsdtData.TokenMetaData.Royalties)
		assert.Equal(t, newAttributes, latestEsdtData.TokenMetaData.Attributes)

		assert.Equal(t, name, latestEsdtData.TokenMetaData.Name)
		assert.Equal(t, hash, latestEsdtData.TokenMetaData.Hash)
		assert.Equal(t, uris, latestEsdtData.TokenMetaData.URIs)
	})
}
