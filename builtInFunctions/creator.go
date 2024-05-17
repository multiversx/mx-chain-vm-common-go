package builtInFunctions

import (
	"github.com/mitchellh/mapstructure"
	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

var _ vmcommon.BuiltInFunctionFactory = (*builtInFuncCreator)(nil)

var trueHandler = func() bool { return true }
var falseHandler = func() bool { return false }

const deleteUserNameFuncName = "DeleteUserName" // all builtInFunction names are upper case

// ArgsCreateBuiltInFunctionContainer defines the input arguments to create built in functions container
type ArgsCreateBuiltInFunctionContainer struct {
	GasMap                                map[string]map[string]uint64
	MapDNSAddresses                       map[string]struct{}
	MapDNSV2Addresses                     map[string]struct{}
	MapWhiteListedCrossChainMintAddresses map[string]struct{}
	EnableUserNameChange                  bool
	Marshalizer                           vmcommon.Marshalizer
	Accounts                              vmcommon.AccountsAdapter
	ShardCoordinator                      vmcommon.Coordinator
	EnableEpochsHandler                   vmcommon.EnableEpochsHandler
	GuardedAccountHandler                 vmcommon.GuardedAccountHandler
	MaxNumOfAddressesForTransferRole      uint32
	ConfigAddress                         []byte
	SelfESDTPrefix                        []byte
}

type builtInFuncCreator struct {
	mapDNSAddresses                       map[string]struct{}
	mapDNSV2Addresses                     map[string]struct{}
	mapWhiteListedCrossChainMintAddresses map[string]struct{}
	enableUserNameChange                  bool
	marshaller                            vmcommon.Marshalizer
	accounts                              vmcommon.AccountsAdapter
	builtInFunctions                      vmcommon.BuiltInFunctionContainer
	gasConfig                             *vmcommon.GasCost
	shardCoordinator                      vmcommon.Coordinator
	esdtStorageHandler                    vmcommon.ESDTNFTStorageHandler
	esdtGlobalSettingsHandler             vmcommon.ESDTGlobalSettingsHandler
	enableEpochsHandler                   vmcommon.EnableEpochsHandler
	guardedAccountHandler                 vmcommon.GuardedAccountHandler
	maxNumOfAddressesForTransferRole      uint32
	configAddress                         []byte
	selfESDTPrefix                        []byte
}

// NewBuiltInFunctionsCreator creates a component which will instantiate the built in functions contracts
func NewBuiltInFunctionsCreator(args ArgsCreateBuiltInFunctionContainer) (*builtInFuncCreator, error) {
	if check.IfNil(args.Marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(args.Accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if args.MapDNSAddresses == nil {
		return nil, ErrNilDnsAddresses
	}
	if args.MapDNSV2Addresses == nil {
		return nil, ErrNilDnsAddresses
	}
	if check.IfNil(args.ShardCoordinator) {
		return nil, ErrNilShardCoordinator
	}
	if check.IfNil(args.EnableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}
	err := core.CheckHandlerCompatibility(args.EnableEpochsHandler, allFlags)
	if err != nil {
		return nil, err
	}
	if check.IfNil(args.GuardedAccountHandler) {
		return nil, ErrNilGuardedAccountHandler
	}

	b := &builtInFuncCreator{
		mapDNSAddresses:                       args.MapDNSAddresses,
		mapDNSV2Addresses:                     args.MapDNSV2Addresses,
		enableUserNameChange:                  args.EnableUserNameChange,
		marshaller:                            args.Marshalizer,
		accounts:                              args.Accounts,
		shardCoordinator:                      args.ShardCoordinator,
		enableEpochsHandler:                   args.EnableEpochsHandler,
		guardedAccountHandler:                 args.GuardedAccountHandler,
		maxNumOfAddressesForTransferRole:      args.MaxNumOfAddressesForTransferRole,
		configAddress:                         args.ConfigAddress,
		selfESDTPrefix:                        args.SelfESDTPrefix,
		mapWhiteListedCrossChainMintAddresses: args.MapWhiteListedCrossChainMintAddresses,
	}

	b.gasConfig, err = createGasConfig(args.GasMap)
	if err != nil {
		return nil, err
	}
	b.builtInFunctions = NewBuiltInFunctionContainer()

	return b, nil
}

// GasScheduleChange is called when gas schedule is changed, thus all contracts must be updated
func (b *builtInFuncCreator) GasScheduleChange(gasSchedule map[string]map[string]uint64) {
	newGasConfig, err := createGasConfig(gasSchedule)
	if err != nil {
		return
	}

	b.gasConfig = newGasConfig
	for key := range b.builtInFunctions.Keys() {
		builtInFunc, errGet := b.builtInFunctions.Get(key)
		if errGet != nil {
			return
		}

		builtInFunc.SetNewGasConfig(b.gasConfig)
	}
}

// NFTStorageHandler will return the esdt storage handler from the built in functions factory
func (b *builtInFuncCreator) NFTStorageHandler() vmcommon.SimpleESDTNFTStorageHandler {
	return b.esdtStorageHandler
}

// ESDTGlobalSettingsHandler will return the esdt global settings handler from the built in functions factory
func (b *builtInFuncCreator) ESDTGlobalSettingsHandler() vmcommon.ESDTGlobalSettingsHandler {
	return b.esdtGlobalSettingsHandler
}

// BuiltInFunctionContainer will return the built in function container
func (b *builtInFuncCreator) BuiltInFunctionContainer() vmcommon.BuiltInFunctionContainer {
	return b.builtInFunctions
}

// CreateBuiltInFunctionContainer will create the list of built-in functions
func (b *builtInFuncCreator) CreateBuiltInFunctionContainer() error {

	b.builtInFunctions = NewBuiltInFunctionContainer()
	var newFunc vmcommon.BuiltinFunction
	newFunc = NewClaimDeveloperRewardsFunc(b.gasConfig.BuiltInCost.ClaimDeveloperRewards)
	err := b.builtInFunctions.Add(core.BuiltInFunctionClaimDeveloperRewards, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewChangeOwnerAddressFunc(b.gasConfig.BuiltInCost.ChangeOwnerAddress, b.enableEpochsHandler)
	if err != nil {
		return err
	}

	err = b.builtInFunctions.Add(core.BuiltInFunctionChangeOwnerAddress, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewSaveUserNameFunc(b.gasConfig.BuiltInCost.SaveUserName, b.mapDNSAddresses, b.mapDNSV2Addresses, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionSetUserName, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDeleteUserNameFunc(b.gasConfig.BuiltInCost.SaveUserName, b.mapDNSV2Addresses, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(deleteUserNameFuncName, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewSaveKeyValueStorageFunc(b.gasConfig.BaseOperationCost, b.gasConfig.BuiltInCost.SaveKeyValue, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionSaveKeyValue, newFunc)
	if err != nil {
		return err
	}

	globalSettingsFunc, err := NewESDTGlobalSettingsFunc(
		b.accounts,
		b.marshaller,
		true,
		core.BuiltInFunctionESDTPause,
		trueHandler,
	)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTPause, globalSettingsFunc)
	if err != nil {
		return err
	}
	b.esdtGlobalSettingsHandler = globalSettingsFunc

	setRoleFunc, err := NewESDTRolesFunc(b.marshaller, true)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionSetESDTRole, setRoleFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewESDTTransferFunc(
		b.gasConfig.BuiltInCost.ESDTTransfer,
		b.marshaller,
		globalSettingsFunc,
		b.shardCoordinator,
		setRoleFunc,
		b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTTransfer, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewESDTBurnFunc(b.gasConfig.BuiltInCost.ESDTBurn, b.marshaller, globalSettingsFunc, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTBurn, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewESDTGlobalSettingsFunc(
		b.accounts,
		b.marshaller,
		false,
		core.BuiltInFunctionESDTUnPause,
		trueHandler,
	)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTUnPause, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewESDTRolesFunc(b.marshaller, false)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionUnSetESDTRole, newFunc)
	if err != nil {
		return err
	}

	crossChainTokenCheckerHandler, err := NewCrossChainTokenChecker(b.selfESDTPrefix, b.mapWhiteListedCrossChainMintAddresses)
	if err != nil {
		return err
	}

	argsEsdtLocalBurn := ESDTLocalMintBurnFuncArgs{
		FuncGasCost:            b.gasConfig.BuiltInCost.ESDTLocalBurn,
		Marshaller:             b.marshaller,
		GlobalSettingsHandler:  globalSettingsFunc,
		RolesHandler:           setRoleFunc,
		EnableEpochsHandler:    b.enableEpochsHandler,
		CrossChainTokenChecker: crossChainTokenCheckerHandler,
	}
	newFunc, err = NewESDTLocalBurnFunc(argsEsdtLocalBurn)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTLocalBurn, newFunc)
	if err != nil {
		return err
	}

	argsLocalMint := ESDTLocalMintBurnFuncArgs{
		FuncGasCost:            b.gasConfig.BuiltInCost.ESDTLocalMint,
		Marshaller:             b.marshaller,
		GlobalSettingsHandler:  globalSettingsFunc,
		RolesHandler:           setRoleFunc,
		EnableEpochsHandler:    b.enableEpochsHandler,
		CrossChainTokenChecker: crossChainTokenCheckerHandler,
	}
	newFunc, err = NewESDTLocalMintFunc(argsLocalMint)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTLocalMint, newFunc)
	if err != nil {
		return err
	}

	args := ArgsNewESDTDataStorage{
		Accounts:              b.accounts,
		GlobalSettingsHandler: globalSettingsFunc,
		Marshalizer:           b.marshaller,
		EnableEpochsHandler:   b.enableEpochsHandler,
		ShardCoordinator:      b.shardCoordinator,
	}
	b.esdtStorageHandler, err = NewESDTDataStorage(args)
	if err != nil {
		return err
	}

	newFunc, err = NewESDTNFTAddQuantityFunc(b.gasConfig.BuiltInCost.ESDTNFTAddQuantity, b.esdtStorageHandler, globalSettingsFunc, setRoleFunc, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTNFTAddQuantity, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewESDTNFTBurnFunc(b.gasConfig.BuiltInCost.ESDTNFTBurn, b.esdtStorageHandler, globalSettingsFunc, setRoleFunc)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTNFTBurn, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewESDTNFTCreateFunc(
		b.gasConfig.BuiltInCost.ESDTNFTCreate,
		b.gasConfig.BaseOperationCost,
		b.marshaller,
		globalSettingsFunc,
		setRoleFunc,
		b.esdtStorageHandler,
		b.accounts,
		b.enableEpochsHandler,
		crossChainTokenCheckerHandler,
	)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTNFTCreate, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewESDTFreezeWipeFunc(b.esdtStorageHandler, b.enableEpochsHandler, b.marshaller, true, false)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTFreeze, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewESDTFreezeWipeFunc(b.esdtStorageHandler, b.enableEpochsHandler, b.marshaller, false, false)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTUnFreeze, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewESDTFreezeWipeFunc(b.esdtStorageHandler, b.enableEpochsHandler, b.marshaller, false, true)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTWipe, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewESDTNFTTransferFunc(b.gasConfig.BuiltInCost.ESDTNFTTransfer,
		b.marshaller,
		globalSettingsFunc,
		b.accounts,
		b.shardCoordinator,
		b.gasConfig.BaseOperationCost,
		setRoleFunc,
		b.esdtStorageHandler,
		b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTNFTTransfer, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewESDTNFTCreateRoleTransfer(b.marshaller, b.accounts, b.shardCoordinator)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTNFTCreateRoleTransfer, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewESDTNFTUpdateAttributesFunc(b.gasConfig.BuiltInCost.ESDTNFTUpdateAttributes, b.gasConfig.BaseOperationCost, b.esdtStorageHandler, globalSettingsFunc, setRoleFunc, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTNFTUpdateAttributes, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewESDTNFTAddUriFunc(b.gasConfig.BuiltInCost.ESDTNFTAddURI, b.gasConfig.BaseOperationCost, b.esdtStorageHandler, globalSettingsFunc, setRoleFunc, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTNFTAddURI, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewESDTNFTMultiTransferFunc(b.gasConfig.BuiltInCost.ESDTNFTMultiTransfer,
		b.marshaller,
		globalSettingsFunc,
		b.accounts,
		b.shardCoordinator,
		b.gasConfig.BaseOperationCost,
		b.enableEpochsHandler,
		setRoleFunc,
		b.esdtStorageHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionMultiESDTNFTTransfer, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewESDTGlobalSettingsFunc(
		b.accounts,
		b.marshaller,
		true,
		core.BuiltInFunctionESDTSetLimitedTransfer,
		func() bool {
			return b.enableEpochsHandler.IsFlagEnabled(ESDTTransferRoleFlag)
		},
	)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTSetLimitedTransfer, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewESDTGlobalSettingsFunc(
		b.accounts,
		b.marshaller,
		false,
		core.BuiltInFunctionESDTUnSetLimitedTransfer,
		func() bool {
			return b.enableEpochsHandler.IsFlagEnabled(ESDTTransferRoleFlag)
		},
	)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTUnSetLimitedTransfer, newFunc)
	if err != nil {
		return err
	}

	argsNewDeleteFunc := ArgsNewESDTDeleteMetadata{
		FuncGasCost:         b.gasConfig.BuiltInCost.ESDTNFTBurn,
		Marshalizer:         b.marshaller,
		Accounts:            b.accounts,
		AllowedAddress:      b.configAddress,
		Delete:              true,
		EnableEpochsHandler: b.enableEpochsHandler,
	}
	newFunc, err = NewESDTDeleteMetadataFunc(argsNewDeleteFunc)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(vmcommon.ESDTDeleteMetadata, newFunc)
	if err != nil {
		return err
	}

	argsNewDeleteFunc.Delete = false
	newFunc, err = NewESDTDeleteMetadataFunc(argsNewDeleteFunc)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(vmcommon.ESDTAddMetadata, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewESDTGlobalSettingsFunc(
		b.accounts,
		b.marshaller,
		true,
		vmcommon.BuiltInFunctionESDTSetBurnRoleForAll,
		func() bool {
			return b.enableEpochsHandler.IsFlagEnabled(SendAlwaysFlag)
		},
	)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionESDTSetBurnRoleForAll, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewESDTGlobalSettingsFunc(
		b.accounts,
		b.marshaller,
		false,
		vmcommon.BuiltInFunctionESDTUnSetBurnRoleForAll,
		func() bool {
			return b.enableEpochsHandler.IsFlagEnabled(SendAlwaysFlag)
		},
	)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionESDTUnSetBurnRoleForAll, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewESDTTransferRoleAddressFunc(b.accounts, b.marshaller, b.maxNumOfAddressesForTransferRole, false, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionESDTTransferRoleDeleteAddress, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewESDTTransferRoleAddressFunc(b.accounts, b.marshaller, b.maxNumOfAddressesForTransferRole, true, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionESDTTransferRoleAddAddress, newFunc)
	if err != nil {
		return err
	}

	argsSetGuardian := SetGuardianArgs{
		BaseAccountGuarderArgs: b.createBaseAccountGuarderArgs(b.gasConfig.BuiltInCost.SetGuardian),
	}
	newFunc, err = NewSetGuardianFunc(argsSetGuardian)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionSetGuardian, newFunc)
	if err != nil {
		return err
	}

	argsGuardAccount := b.createGuardAccountArgs()
	newFunc, err = NewGuardAccountFunc(argsGuardAccount)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionGuardAccount, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewUnGuardAccountFunc(argsGuardAccount)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionUnGuardAccount, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewMigrateDataTrieFunc(b.gasConfig.BuiltInCost, b.enableEpochsHandler, b.accounts)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionMigrateDataTrie, newFunc)
	if err != nil {
		return err
	}

	activeHandler := func() bool {
		return b.enableEpochsHandler.IsFlagEnabled(DynamicEsdtFlag)
	}
	newFunc, err = NewESDTSetTokenTypeFunc(b.accounts, globalSettingsFunc, b.marshaller, activeHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.ESDTSetTokenType, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewESDTMetaDataRecreateFunc(b.gasConfig.BuiltInCost.ESDTNFTRecreate, b.gasConfig.BaseOperationCost, b.accounts, globalSettingsFunc, b.esdtStorageHandler, setRoleFunc, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.ESDTMetaDataRecreate, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewESDTMetaDataUpdateFunc(b.gasConfig.BuiltInCost.ESDTNFTUpdate, b.gasConfig.BaseOperationCost, b.accounts, globalSettingsFunc, b.esdtStorageHandler, setRoleFunc, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.ESDTMetaDataUpdate, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewESDTSetNewURIsFunc(b.gasConfig.BuiltInCost.ESDTNFTRecreate, b.gasConfig.BaseOperationCost, b.accounts, globalSettingsFunc, b.esdtStorageHandler, setRoleFunc, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.ESDTSetNewURIs, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewESDTModifyRoyaltiesFunc(b.gasConfig.BuiltInCost.ESDTModifyRoyalties, b.accounts, globalSettingsFunc, b.esdtStorageHandler, setRoleFunc, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.ESDTModifyRoyalties, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewESDTModifyCreatorFunc(b.gasConfig.BuiltInCost.ESDTModifyRoyalties, b.accounts, globalSettingsFunc, b.esdtStorageHandler, setRoleFunc, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.ESDTModifyCreator, newFunc)
	if err != nil {
		return err
	}

	return nil
}

func (b *builtInFuncCreator) createBaseAccountGuarderArgs(funcGasCost uint64) BaseAccountGuarderArgs {
	return BaseAccountGuarderArgs{
		Marshaller:            b.marshaller,
		FuncGasCost:           funcGasCost,
		GuardedAccountHandler: b.guardedAccountHandler,
		EnableEpochsHandler:   b.enableEpochsHandler,
	}
}

func (b *builtInFuncCreator) createGuardAccountArgs() GuardAccountArgs {
	return GuardAccountArgs{
		BaseAccountGuarderArgs: b.createBaseAccountGuarderArgs(b.gasConfig.BuiltInCost.GuardAccount),
	}
}

func createGasConfig(gasMap map[string]map[string]uint64) (*vmcommon.GasCost, error) {
	baseOps := &vmcommon.BaseOperationCost{}
	err := mapstructure.Decode(gasMap[core.BaseOperationCostString], baseOps)
	if err != nil {
		return nil, err
	}

	err = check.ForZeroUintFields(*baseOps)
	if err != nil {
		return nil, err
	}

	builtInOps := &vmcommon.BuiltInCost{}
	err = mapstructure.Decode(gasMap[core.BuiltInCostString], builtInOps)
	if err != nil {
		return nil, err
	}

	err = check.ForZeroUintFields(*builtInOps)
	if err != nil {
		return nil, err
	}

	gasCost := vmcommon.GasCost{
		BaseOperationCost: *baseOps,
		BuiltInCost:       *builtInOps,
	}

	return &gasCost, nil
}

// SetBlockchainHook sets the blockchain hook to the needed functions
func (b *builtInFuncCreator) SetBlockchainHook(blockchainHook vmcommon.BlockchainDataHook) error {
	if check.IfNil(blockchainHook) {
		return ErrNilBlockchainHook
	}

	builtInFuncs := b.builtInFunctions.Keys()
	for funcName := range builtInFuncs {
		builtInFunc, err := b.builtInFunctions.Get(funcName)
		if err != nil {
			return err
		}

		esdtBlockchainDataProvider, ok := builtInFunc.(vmcommon.BlockchainDataProvider)
		if !ok {
			continue
		}

		err = esdtBlockchainDataProvider.SetBlockchainHook(blockchainHook)
		if err != nil {
			return err
		}
	}

	return nil
}

// SetPayableHandler sets the payableCheck interface to the needed functions
func (b *builtInFuncCreator) SetPayableHandler(payableHandler vmcommon.PayableHandler) error {
	payableChecker, err := NewPayableCheckFunc(
		payableHandler,
		b.enableEpochsHandler,
	)
	if err != nil {
		return err
	}

	listOfTransferFunc := []string{
		core.BuiltInFunctionMultiESDTNFTTransfer,
		core.BuiltInFunctionESDTNFTTransfer,
		core.BuiltInFunctionESDTTransfer,
	}

	for _, transferFunc := range listOfTransferFunc {
		builtInFunc, err := b.builtInFunctions.Get(transferFunc)
		if err != nil {
			return err
		}

		esdtTransferFunc, ok := builtInFunc.(vmcommon.AcceptPayableChecker)
		if !ok {
			return ErrWrongTypeAssertion
		}

		err = esdtTransferFunc.SetPayableChecker(payableChecker)
		if err != nil {
			return err
		}
	}

	return nil
}

// IsInterfaceNil returns true if underlying object is nil
func (b *builtInFuncCreator) IsInterfaceNil() bool {
	return b == nil
}
