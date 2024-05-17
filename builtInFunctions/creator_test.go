package builtInFunctions

import (
	"errors"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createMockArguments() ArgsCreateBuiltInFunctionContainer {
	gasMap := make(map[string]map[string]uint64)
	fillGasMapInternal(gasMap, 1)

	args := ArgsCreateBuiltInFunctionContainer{
		GasMap:                                gasMap,
		MapDNSAddresses:                       make(map[string]struct{}),
		MapDNSV2Addresses:                     make(map[string]struct{}),
		EnableUserNameChange:                  false,
		Marshalizer:                           &mock.MarshalizerMock{},
		Accounts:                              &mock.AccountsStub{},
		ShardCoordinator:                      mock.NewMultiShardsCoordinatorMock(1),
		EnableEpochsHandler:                   &mock.EnableEpochsHandlerStub{},
		GuardedAccountHandler:                 &mock.GuardedAccountHandlerStub{},
		MaxNumOfAddressesForTransferRole:      100,
		MapWhiteListedCrossChainMintAddresses: getWhiteListedAddress(),
	}

	return args
}

func fillGasMapInternal(gasMap map[string]map[string]uint64, value uint64) map[string]map[string]uint64 {
	gasMap[core.BaseOperationCostString] = fillGasMapBaseOperationCosts(value)
	gasMap[core.BuiltInCostString] = fillGasMapBuiltInCosts(value)

	return gasMap
}

func fillGasMapBaseOperationCosts(value uint64) map[string]uint64 {
	gasMap := make(map[string]uint64)
	gasMap["StorePerByte"] = value
	gasMap["DataCopyPerByte"] = value
	gasMap["ReleasePerByte"] = value
	gasMap["PersistPerByte"] = value
	gasMap["CompilePerByte"] = value
	gasMap["AoTPreparePerByte"] = value
	gasMap["GetCode"] = value
	return gasMap
}

func fillGasMapBuiltInCosts(value uint64) map[string]uint64 {
	gasMap := make(map[string]uint64)
	gasMap["ClaimDeveloperRewards"] = value
	gasMap["ChangeOwnerAddress"] = value
	gasMap["SaveUserName"] = value
	gasMap["SaveKeyValue"] = value
	gasMap["ESDTTransfer"] = value
	gasMap["ESDTBurn"] = value
	gasMap["ChangeOwnerAddress"] = value
	gasMap["ClaimDeveloperRewards"] = value
	gasMap["SaveUserName"] = value
	gasMap["SaveKeyValue"] = value
	gasMap["ESDTTransfer"] = value
	gasMap["ESDTBurn"] = value
	gasMap["ESDTLocalMint"] = value
	gasMap["ESDTLocalBurn"] = value
	gasMap["ESDTNFTCreate"] = value
	gasMap["ESDTNFTAddQuantity"] = value
	gasMap["ESDTNFTBurn"] = value
	gasMap["ESDTNFTTransfer"] = value
	gasMap["ESDTNFTChangeCreateOwner"] = value
	gasMap["ESDTNFTAddUri"] = value
	gasMap["ESDTNFTUpdateAttributes"] = value
	gasMap["ESDTNFTMultiTransfer"] = value
	gasMap["SetGuardian"] = value
	gasMap["GuardAccount"] = value
	gasMap["UnGuardAccount"] = value
	gasMap["TrieLoadPerNode"] = value
	gasMap["TrieStorePerNode"] = value
	gasMap["ESDTModifyRoyalties"] = value
	gasMap["ESDTModifyCreator"] = value
	gasMap["ESDTNFTRecreate"] = value
	gasMap["ESDTNFTSetNewURIs"] = value
	gasMap["ESDTNFTUpdate"] = value

	return gasMap
}

func TestCreateBuiltInFunctionContainer_Errors(t *testing.T) {
	args := createMockArguments()
	args.GasMap[core.BuiltInCostString]["ClaimDeveloperRewards"] = 0

	f, err := NewBuiltInFunctionsCreator(args)
	assert.Nil(t, f)
	assert.NotNil(t, err)

	args = createMockArguments()
	args.ShardCoordinator = nil
	_, err = NewBuiltInFunctionsCreator(args)
	assert.Equal(t, err, ErrNilShardCoordinator)

	args = createMockArguments()
	args.EnableEpochsHandler = nil
	_, err = NewBuiltInFunctionsCreator(args)
	assert.Equal(t, err, ErrNilEnableEpochsHandler)

	args = createMockArguments()
	args.EnableEpochsHandler = &mock.EnableEpochsHandlerStub{
		IsFlagDefinedCalled: func(flag core.EnableEpochFlag) bool {
			return false
		},
	}
	_, err = NewBuiltInFunctionsCreator(args)
	assert.True(t, errors.Is(err, core.ErrInvalidEnableEpochsHandler))

	args = createMockArguments()
	args.Marshalizer = nil
	_, err = NewBuiltInFunctionsCreator(args)
	assert.Equal(t, err, ErrNilMarshalizer)

	args = createMockArguments()
	args.Accounts = nil
	_, err = NewBuiltInFunctionsCreator(args)
	assert.Equal(t, err, ErrNilAccountsAdapter)

	args = createMockArguments()
	args.GuardedAccountHandler = nil
	_, err = NewBuiltInFunctionsCreator(args)
	assert.Equal(t, err, ErrNilGuardedAccountHandler)

	args = createMockArguments()
	f, err = NewBuiltInFunctionsCreator(args)
	assert.Nil(t, err)
	assert.NotNil(t, f)
}

func TestBuiltInFuncCreator_IsInterfaceNil(t *testing.T) {
	t.Parallel()

	var instance *builtInFuncCreator
	require.True(t, instance.IsInterfaceNil())

	instance, _ = NewBuiltInFunctionsCreator(createMockArguments())
	require.False(t, instance.IsInterfaceNil())
}

func TestCreateBuiltInContainer_GasScheduleChange(t *testing.T) {
	args := createMockArguments()
	f, _ := NewBuiltInFunctionsCreator(args)

	fillGasMapInternal(args.GasMap, 5)
	args.GasMap[core.BuiltInCostString]["ClaimDeveloperRewards"] = 0
	f.GasScheduleChange(args.GasMap)
	assert.Equal(t, f.gasConfig.BuiltInCost.ClaimDeveloperRewards, uint64(1))

	args.GasMap[core.BuiltInCostString]["ClaimDeveloperRewards"] = 5
	f.GasScheduleChange(args.GasMap)
	assert.Equal(t, f.gasConfig.BuiltInCost.ClaimDeveloperRewards, uint64(5))
}

func TestCreateBuiltInContainer_Create(t *testing.T) {
	args := createMockArguments()
	f, _ := NewBuiltInFunctionsCreator(args)

	err := f.CreateBuiltInFunctionContainer()
	assert.Nil(t, err)
	assert.Equal(t, 42, f.BuiltInFunctionContainer().Len())

	err = f.SetPayableHandler(nil)
	assert.Equal(t, ErrNilPayableHandler, err)

	err = f.SetPayableHandler(&mock.PayableHandlerStub{})
	assert.Nil(t, err)

	err = f.SetBlockchainHook(nil)
	assert.Equal(t, ErrNilBlockchainHook, err)

	numSetBlockDataHandlerCalls := 0
	for funcName := range f.builtInFunctions.Keys() {
		builtInFunc, _ := f.builtInFunctions.Get(funcName)
		_, ok := builtInFunc.(vmcommon.BlockchainDataProvider)
		if !ok {
			continue
		}

		builtInFunc = &mock.BuiltInFunctionStub{
			SetBlockchainHookCalled: func(blockDataHandler vmcommon.BlockchainDataHook) error {
				numSetBlockDataHandlerCalls++
				return nil
			},
		}
		err = f.builtInFunctions.Replace(funcName, builtInFunc)
		assert.Nil(t, err)
	}

	err = f.SetBlockchainHook(&disabledBlockchainHook{})
	assert.Nil(t, err)
	assert.Equal(t, 6, numSetBlockDataHandlerCalls)

	fillGasMapInternal(args.GasMap, 5)
	f.GasScheduleChange(args.GasMap)
	assert.Equal(t, f.gasConfig.BuiltInCost.ClaimDeveloperRewards, uint64(5))

	nftStorageHandler := f.NFTStorageHandler()
	assert.False(t, check.IfNil(nftStorageHandler))
}
