package builtInFunctions

import (
	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/mitchellh/mapstructure"
)

// ArgsCreateBuiltInFunctionContainer defines the input arguments to create built in functions container
type ArgsCreateBuiltInFunctionContainer struct {
	GasMap                              map[string]map[string]uint64
	MapDNSAddresses                     map[string]struct{}
	EnableUserNameChange                bool
	Marshalizer                         vmcommon.Marshalizer
	Accounts                            vmcommon.AccountsAdapter
	ShardCoordinator                    vmcommon.Coordinator
	EpochNotifier                       vmcommon.EpochNotifier
	PubKeyConverter                     core.PubkeyConverter
	BlockChainEpochHook                 BlockChainEpochHook
	ESDTNFTImprovementV1ActivationEpoch uint32
	ESDTTransferRoleEnableEpoch         uint32
	GlobalMintBurnDisableEpoch          uint32
	ESDTTransferToMetaEnableEpoch       uint32
	NFTCreateMultiShardEnableEpoch      uint32
	SaveNFTToSystemAccountEnableEpoch   uint32
	SetGuardianEnableEpoch              uint32
	GuardianActivationEpochs            uint32
	FreezeAccountEnableEpoch            uint32
}

type builtInFuncCreator struct {
	mapDNSAddresses                     map[string]struct{}
	enableUserNameChange                bool
	marshalizer                         vmcommon.Marshalizer
	accounts                            vmcommon.AccountsAdapter
	builtInFunctions                    vmcommon.BuiltInFunctionContainer
	gasConfig                           *vmcommon.GasCost
	shardCoordinator                    vmcommon.Coordinator
	epochNotifier                       vmcommon.EpochNotifier
	esdtStorageHandler                  vmcommon.ESDTNFTStorageHandler
	pubKeyConverter                     core.PubkeyConverter
	blockChainEpochHook                 BlockChainEpochHook
	esdtNFTImprovementV1ActivationEpoch uint32
	esdtTransferRoleEnableEpoch         uint32
	globalMintBurnDisableEpoch          uint32
	esdtTransferToMetaEnableEpoch       uint32
	nftCreateMultiShardEnableEpoch      uint32
	saveNFTToSystemAccountEnableEpoch   uint32
	setGuardianEnableEpoch              uint32
	guardianActivationEpochs            uint32
	freezeAccountEnableEpoch            uint32
}

// NewBuiltInFunctionsCreator creates a component which will instantiate the built in functions contracts
func NewBuiltInFunctionsCreator(args ArgsCreateBuiltInFunctionContainer) (*builtInFuncCreator, error) {
	if check.IfNil(args.Marshalizer) {
		return nil, ErrNilMarshaller
	}
	if check.IfNil(args.Accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if args.MapDNSAddresses == nil {
		return nil, ErrNilDnsAddresses
	}
	if check.IfNil(args.ShardCoordinator) {
		return nil, ErrNilShardCoordinator
	}
	if check.IfNil(args.EpochNotifier) {
		return nil, ErrNilEpochNotifier
	}
	if check.IfNil(args.BlockChainEpochHook) {
		return nil, ErrNilBlockChainHook
	}
	if check.IfNil(args.PubKeyConverter) {
		return nil, ErrNilPubKeyConverter
	}

	b := &builtInFuncCreator{
		mapDNSAddresses:                     args.MapDNSAddresses,
		enableUserNameChange:                args.EnableUserNameChange,
		marshalizer:                         args.Marshalizer,
		accounts:                            args.Accounts,
		shardCoordinator:                    args.ShardCoordinator,
		epochNotifier:                       args.EpochNotifier,
		pubKeyConverter:                     args.PubKeyConverter,
		blockChainEpochHook:                 args.BlockChainEpochHook,
		esdtNFTImprovementV1ActivationEpoch: args.ESDTNFTImprovementV1ActivationEpoch,
		esdtTransferRoleEnableEpoch:         args.ESDTTransferRoleEnableEpoch,
		globalMintBurnDisableEpoch:          args.GlobalMintBurnDisableEpoch,
		esdtTransferToMetaEnableEpoch:       args.ESDTTransferToMetaEnableEpoch,
		nftCreateMultiShardEnableEpoch:      args.NFTCreateMultiShardEnableEpoch,
		saveNFTToSystemAccountEnableEpoch:   args.SaveNFTToSystemAccountEnableEpoch,
		setGuardianEnableEpoch:              args.SetGuardianEnableEpoch,
		guardianActivationEpochs:            args.GuardianActivationEpochs,
		freezeAccountEnableEpoch:            args.FreezeAccountEnableEpoch,
	}

	var err error
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

// CreateBuiltInFunctionContainer will create the list of built-in functions
func (b *builtInFuncCreator) CreateBuiltInFunctionContainer() (vmcommon.BuiltInFunctionContainer, error) {

	b.builtInFunctions = NewBuiltInFunctionContainer()
	var newFunc vmcommon.BuiltinFunction
	newFunc = NewClaimDeveloperRewardsFunc(b.gasConfig.BuiltInCost.ClaimDeveloperRewards)
	err := b.builtInFunctions.Add(core.BuiltInFunctionClaimDeveloperRewards, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc = NewChangeOwnerAddressFunc(b.gasConfig.BuiltInCost.ChangeOwnerAddress)
	err = b.builtInFunctions.Add(core.BuiltInFunctionChangeOwnerAddress, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewSaveUserNameFunc(b.gasConfig.BuiltInCost.SaveUserName, b.mapDNSAddresses, b.enableUserNameChange)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionSetUserName, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewSaveKeyValueStorageFunc(b.gasConfig.BaseOperationCost, b.gasConfig.BuiltInCost.SaveKeyValue)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionSaveKeyValue, newFunc)
	if err != nil {
		return nil, err
	}

	globalSettingsFunc, err := NewESDTGlobalSettingsFunc(b.accounts, true, core.BuiltInFunctionESDTPause, 0, b.epochNotifier)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTPause, globalSettingsFunc)
	if err != nil {
		return nil, err
	}

	setRoleFunc, err := NewESDTRolesFunc(b.marshalizer, true)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionSetESDTRole, setRoleFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTTransferFunc(b.gasConfig.BuiltInCost.ESDTTransfer, b.marshalizer, globalSettingsFunc, b.shardCoordinator, setRoleFunc, b.esdtTransferToMetaEnableEpoch, b.epochNotifier)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTTransfer, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTBurnFunc(b.gasConfig.BuiltInCost.ESDTBurn, b.marshalizer, globalSettingsFunc, b.globalMintBurnDisableEpoch, b.epochNotifier)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTBurn, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTFreezeWipeFunc(b.marshalizer, true, false)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTFreeze, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTFreezeWipeFunc(b.marshalizer, false, false)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTUnFreeze, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTFreezeWipeFunc(b.marshalizer, false, true)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTWipe, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTGlobalSettingsFunc(b.accounts, false, core.BuiltInFunctionESDTUnPause, 0, b.epochNotifier)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTUnPause, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTRolesFunc(b.marshalizer, false)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionUnSetESDTRole, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTLocalBurnFunc(b.gasConfig.BuiltInCost.ESDTLocalBurn, b.marshalizer, globalSettingsFunc, setRoleFunc)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTLocalBurn, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTLocalMintFunc(b.gasConfig.BuiltInCost.ESDTLocalMint, b.marshalizer, globalSettingsFunc, setRoleFunc)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTLocalMint, newFunc)
	if err != nil {
		return nil, err
	}

	args := ArgsNewESDTDataStorage{
		Accounts:                b.accounts,
		GlobalSettingsHandler:   globalSettingsFunc,
		Marshalizer:             b.marshalizer,
		SaveToSystemEnableEpoch: b.saveNFTToSystemAccountEnableEpoch,
		EpochNotifier:           b.epochNotifier,
		ShardCoordinator:        b.shardCoordinator,
	}
	b.esdtStorageHandler, err = NewESDTDataStorage(args)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTNFTAddQuantityFunc(b.gasConfig.BuiltInCost.ESDTNFTAddQuantity, b.esdtStorageHandler, globalSettingsFunc, setRoleFunc, b.saveNFTToSystemAccountEnableEpoch, b.epochNotifier)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTNFTAddQuantity, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTNFTBurnFunc(b.gasConfig.BuiltInCost.ESDTNFTBurn, b.esdtStorageHandler, globalSettingsFunc, setRoleFunc)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTNFTBurn, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTNFTCreateFunc(b.gasConfig.BuiltInCost.ESDTNFTCreate, b.gasConfig.BaseOperationCost, b.marshalizer, globalSettingsFunc, setRoleFunc, b.esdtStorageHandler, b.accounts, b.saveNFTToSystemAccountEnableEpoch, b.epochNotifier)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTNFTCreate, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTNFTTransferFunc(b.gasConfig.BuiltInCost.ESDTNFTTransfer, b.marshalizer, globalSettingsFunc, b.accounts, b.shardCoordinator, b.gasConfig.BaseOperationCost, setRoleFunc, b.esdtTransferToMetaEnableEpoch, b.saveNFTToSystemAccountEnableEpoch, b.esdtStorageHandler, b.epochNotifier)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTNFTTransfer, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTNFTCreateRoleTransfer(b.marshalizer, b.accounts, b.shardCoordinator)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTNFTCreateRoleTransfer, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTNFTUpdateAttributesFunc(b.gasConfig.BuiltInCost.ESDTNFTUpdateAttributes, b.gasConfig.BaseOperationCost, b.esdtStorageHandler, globalSettingsFunc, setRoleFunc, b.esdtNFTImprovementV1ActivationEpoch, b.epochNotifier)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTNFTUpdateAttributes, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTNFTAddUriFunc(b.gasConfig.BuiltInCost.ESDTNFTAddURI, b.gasConfig.BaseOperationCost, b.esdtStorageHandler, globalSettingsFunc, setRoleFunc, b.esdtNFTImprovementV1ActivationEpoch, b.epochNotifier)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTNFTAddURI, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTNFTMultiTransferFunc(b.gasConfig.BuiltInCost.ESDTNFTMultiTransfer, b.marshalizer, globalSettingsFunc, b.accounts, b.shardCoordinator, b.gasConfig.BaseOperationCost, b.esdtNFTImprovementV1ActivationEpoch, b.epochNotifier, setRoleFunc, b.esdtTransferToMetaEnableEpoch, b.esdtStorageHandler)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionMultiESDTNFTTransfer, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTGlobalSettingsFunc(b.accounts, true, core.BuiltInFunctionESDTSetLimitedTransfer, b.esdtTransferRoleEnableEpoch, b.epochNotifier)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTSetLimitedTransfer, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTGlobalSettingsFunc(b.accounts, false, core.BuiltInFunctionESDTUnSetLimitedTransfer, b.esdtTransferRoleEnableEpoch, b.epochNotifier)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionESDTUnSetLimitedTransfer, newFunc)
	if err != nil {
		return nil, err
	}

	argsSetGuardian := SetGuardianArgs{
		BaseAccountFreezerArgs:   b.createBaseAccountFreezerArgs(b.gasConfig.BuiltInCost.SetGuardian),
		PubKeyConverter:          b.pubKeyConverter,
		GuardianActivationEpochs: b.guardianActivationEpochs,
		SetGuardianEnableEpoch:   b.setGuardianEnableEpoch,
	}
	newFunc, err = NewSetGuardianFunc(argsSetGuardian)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(BuiltInFunctionSetGuardian, newFunc)
	if err != nil {
		return nil, err
	}

	argsFreezeAccount := b.createFreezeAccountArgs(true)
	newFunc, err = NewFreezeAccountFunc(argsFreezeAccount)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(BuiltInFunctionFreezeAccount, newFunc)
	if err != nil {
		return nil, err
	}

	argsUnfreezeAccount := b.createFreezeAccountArgs(false)
	newFunc, err = NewFreezeAccountFunc(argsUnfreezeAccount)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(BuiltInFunctionUnfreezeAccount, newFunc)
	if err != nil {
		return nil, err
	}

	return b.builtInFunctions, nil
}

func (b *builtInFuncCreator) createBaseAccountFreezerArgs(funcGasCost uint64) BaseAccountFreezerArgs {
	return BaseAccountFreezerArgs{
		BlockChainHook: b.blockChainEpochHook,
		Marshaller:     b.marshalizer,
		EpochNotifier:  b.epochNotifier,
		FuncGasCost:    funcGasCost,
	}
}

func (b *builtInFuncCreator) createFreezeAccountArgs(freeze bool) FreezeAccountArgs {
	return FreezeAccountArgs{
		BaseAccountFreezerArgs:   b.createBaseAccountFreezerArgs(b.gasConfig.BuiltInCost.FreezeAccount),
		FreezeAccountEnableEpoch: b.freezeAccountEnableEpoch,
		Freeze:                   freeze,
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

// SetPayableHandler sets the payable interface to the needed functions
func SetPayableHandler(container vmcommon.BuiltInFunctionContainer, payableHandler vmcommon.PayableHandler) error {
	listOfTransferFunc := []string{
		core.BuiltInFunctionMultiESDTNFTTransfer,
		core.BuiltInFunctionESDTNFTTransfer,
		core.BuiltInFunctionESDTTransfer}

	for _, transferFunc := range listOfTransferFunc {
		builtInFunc, err := container.Get(transferFunc)
		if err != nil {
			return err
		}

		esdtTransferFunc, ok := builtInFunc.(vmcommon.AcceptPayableHandler)
		if !ok {
			return ErrWrongTypeAssertion
		}

		err = esdtTransferFunc.SetPayableHandler(payableHandler)
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