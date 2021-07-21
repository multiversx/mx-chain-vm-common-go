package builtInFunctions

import (
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/mitchellh/mapstructure"
)

// ArgsCreateBuiltInFunctionContainer -
type ArgsCreateBuiltInFunctionContainer struct {
	GasMap                              map[string]map[string]uint64
	MapDNSAddresses                     map[string]struct{}
	EnableUserNameChange                bool
	Marshalizer                         vmcommon.Marshalizer
	Accounts                            vmcommon.AccountsAdapter
	ShardCoordinator                    vmcommon.Coordinator
	EpochNotifier                       vmcommon.EpochNotifier
	ESDTNFTImprovementV1ActivationEpoch uint32
}

type builtInFuncFactory struct {
	mapDNSAddresses                     map[string]struct{}
	enableUserNameChange                bool
	marshalizer                         vmcommon.Marshalizer
	accounts                            vmcommon.AccountsAdapter
	builtInFunctions                    vmcommon.BuiltInFunctionContainer
	gasConfig                           *vmcommon.GasCost
	shardCoordinator                    vmcommon.Coordinator
	epochNotifier                       vmcommon.EpochNotifier
	esdtNFTImprovementV1ActivationEpoch uint32
}

// NewBuiltInFunctionsFactory creates a factory which will instantiate the built in functions contracts
func NewBuiltInFunctionsFactory(args ArgsCreateBuiltInFunctionContainer) (*builtInFuncFactory, error) {
	if check.IfNil(args.Marshalizer) {
		return nil, ErrNilMarshalizer
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
		return nil, ErrNilEpochHandler
	}

	b := &builtInFuncFactory{
		mapDNSAddresses:                     args.MapDNSAddresses,
		enableUserNameChange:                args.EnableUserNameChange,
		marshalizer:                         args.Marshalizer,
		accounts:                            args.Accounts,
		shardCoordinator:                    args.ShardCoordinator,
		epochNotifier:                       args.EpochNotifier,
		esdtNFTImprovementV1ActivationEpoch: args.ESDTNFTImprovementV1ActivationEpoch,
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
func (b *builtInFuncFactory) GasScheduleChange(gasSchedule map[string]map[string]uint64) {
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

// CreateBuiltInFunctionContainer will create the list of built-in functions
func (b *builtInFuncFactory) CreateBuiltInFunctionContainer() (vmcommon.BuiltInFunctionContainer, error) {

	b.builtInFunctions = NewBuiltInFunctionContainer()
	var newFunc vmcommon.BuiltinFunction
	newFunc = NewClaimDeveloperRewardsFunc(b.gasConfig.BuiltInCost.ClaimDeveloperRewards)
	err := b.builtInFunctions.Add(vmcommon.BuiltInFunctionClaimDeveloperRewards, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc = NewChangeOwnerAddressFunc(b.gasConfig.BuiltInCost.ChangeOwnerAddress)
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionChangeOwnerAddress, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewSaveUserNameFunc(b.gasConfig.BuiltInCost.SaveUserName, b.mapDNSAddresses, b.enableUserNameChange)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionSetUserName, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewSaveKeyValueStorageFunc(b.gasConfig.BaseOperationCost, b.gasConfig.BuiltInCost.SaveKeyValue)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionSaveKeyValue, newFunc)
	if err != nil {
		return nil, err
	}

	pauseFunc, err := NewESDTPauseFunc(b.accounts, true)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionESDTPause, pauseFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTTransferFunc(b.gasConfig.BuiltInCost.ESDTTransfer, b.marshalizer, pauseFunc, b.shardCoordinator)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionESDTTransfer, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTBurnFunc(b.gasConfig.BuiltInCost.ESDTBurn, b.marshalizer, pauseFunc)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionESDTBurn, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTFreezeWipeFunc(b.marshalizer, true, false)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionESDTFreeze, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTFreezeWipeFunc(b.marshalizer, false, false)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionESDTUnFreeze, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTFreezeWipeFunc(b.marshalizer, false, true)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionESDTWipe, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTPauseFunc(b.accounts, false)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionESDTUnPause, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTRolesFunc(b.marshalizer, false)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionUnSetESDTRole, newFunc)
	if err != nil {
		return nil, err
	}

	setRoleFunc, err := NewESDTRolesFunc(b.marshalizer, true)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionSetESDTRole, setRoleFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTLocalBurnFunc(b.gasConfig.BuiltInCost.ESDTLocalBurn, b.marshalizer, pauseFunc, setRoleFunc)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionESDTLocalBurn, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTLocalMintFunc(b.gasConfig.BuiltInCost.ESDTLocalMint, b.marshalizer, pauseFunc, setRoleFunc)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionESDTLocalMint, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTNFTAddQuantityFunc(b.gasConfig.BuiltInCost.ESDTNFTAddQuantity, b.marshalizer, pauseFunc, setRoleFunc)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionESDTNFTAddQuantity, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTNFTBurnFunc(b.gasConfig.BuiltInCost.ESDTNFTBurn, b.marshalizer, pauseFunc, setRoleFunc)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionESDTNFTBurn, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTNFTCreateFunc(b.gasConfig.BuiltInCost.ESDTNFTCreate, b.gasConfig.BaseOperationCost, b.marshalizer, pauseFunc, setRoleFunc)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionESDTNFTCreate, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTNFTTransferFunc(b.gasConfig.BuiltInCost.ESDTNFTTransfer, b.marshalizer, pauseFunc, b.accounts, b.shardCoordinator, b.gasConfig.BaseOperationCost)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionESDTNFTTransfer, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTNFTCreateRoleTransfer(b.marshalizer, b.accounts, b.shardCoordinator)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionESDTNFTCreateRoleTransfer, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTNFTUpdateAttributesFunc(b.gasConfig.BuiltInCost.ESDTNFTUpdateAttributes, b.gasConfig.BaseOperationCost, b.marshalizer, pauseFunc, setRoleFunc, b.esdtNFTImprovementV1ActivationEpoch, b.epochNotifier)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionESDTNFTUpdateAttributes, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTNFTAddUriFunc(b.gasConfig.BuiltInCost.ESDTNFTAddURI, b.gasConfig.BaseOperationCost, b.marshalizer, pauseFunc, setRoleFunc, b.esdtNFTImprovementV1ActivationEpoch, b.epochNotifier)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionESDTNFTAddURI, newFunc)
	if err != nil {
		return nil, err
	}

	newFunc, err = NewESDTNFTMultiTransferFunc(b.gasConfig.BuiltInCost.ESDTNFTMultiTransfer, b.marshalizer, pauseFunc, b.accounts, b.shardCoordinator, b.gasConfig.BaseOperationCost, b.esdtNFTImprovementV1ActivationEpoch, b.epochNotifier)
	if err != nil {
		return nil, err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionMultiESDTNFTTransfer, newFunc)
	if err != nil {
		return nil, err
	}

	return b.builtInFunctions, nil
}

func createGasConfig(gasMap map[string]map[string]uint64) (*vmcommon.GasCost, error) {
	baseOps := &vmcommon.BaseOperationCost{}
	err := mapstructure.Decode(gasMap[vmcommon.BaseOperationCostString], baseOps)
	if err != nil {
		return nil, err
	}

	err = check.ForZeroUintFields(*baseOps)
	if err != nil {
		return nil, err
	}

	builtInOps := &vmcommon.BuiltInCost{}
	err = mapstructure.Decode(gasMap[vmcommon.BuiltInCostString], builtInOps)
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
		vmcommon.BuiltInFunctionMultiESDTNFTTransfer,
		vmcommon.BuiltInFunctionESDTNFTTransfer,
		vmcommon.BuiltInFunctionESDTTransfer}

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
func (b *builtInFuncFactory) IsInterfaceNil() bool {
	return b == nil
}
