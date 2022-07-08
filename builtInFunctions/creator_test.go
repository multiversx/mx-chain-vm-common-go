package builtInFunctions

import (
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-vm-common/mock"
	"github.com/stretchr/testify/assert"
)

func createMockArguments() ArgsCreateBuiltInFunctionContainer {
	gasMap := make(map[string]map[string]uint64)
	fillGasMapInternal(gasMap, 1)

	args := ArgsCreateBuiltInFunctionContainer{
		GasMap:                gasMap,
		MapDNSAddresses:       make(map[string]struct{}),
		EnableUserNameChange:  false,
		Marshalizer:           &mock.MarshalizerMock{},
		Accounts:              &mock.AccountsStub{},
		ShardCoordinator:      mock.NewMultiShardsCoordinatorMock(1),
		EpochNotifier:         &mock.EpochNotifierStub{},
		GuardedAccountHandler: &mock.GuardedAccountHandlerStub{},
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
	gasMap["FreezeAccount"] = value
	gasMap["UnfreezeAccount"] = value

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
	args.EpochNotifier = nil
	_, err = NewBuiltInFunctionsCreator(args)
	assert.Equal(t, err, ErrNilEpochHandler)

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
	f, err = NewBuiltInFunctionsCreator(args)
	assert.Equal(t, err, ErrNilGuardedAccountHandler)

	args = createMockArguments()
	f, err = NewBuiltInFunctionsCreator(args)
	assert.Nil(t, err)
	assert.False(t, f.IsInterfaceNil())
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

	container, err := f.CreateBuiltInFunctionContainer()
	assert.Nil(t, err)
	assert.Equal(t, container.Len(), 32)

	err = SetPayableHandler(container, nil)
	assert.NotNil(t, err)

	err = SetPayableHandler(container, &mock.PayableHandlerStub{})
	assert.Nil(t, err)

	fillGasMapInternal(args.GasMap, 5)
	f.GasScheduleChange(args.GasMap)
	assert.Equal(t, f.gasConfig.BuiltInCost.ClaimDeveloperRewards, uint64(5))

	nftStorageHandler := f.NFTStorageHandler()
	assert.False(t, check.IfNil(nftStorageHandler))
}
