package builtInFunctions

import (
	"bytes"
	"fmt"
	"math/big"
	"sync"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data/esdt"
	"github.com/multiversx/mx-chain-core-go/data/vm"
	logger "github.com/multiversx/mx-chain-logger-go"

	"github.com/multiversx/mx-chain-vm-common-go"
)

var (
	log         = logger.GetOrCreate("builtInFunctions")
	noncePrefix = []byte(core.ProtectedKeyPrefix + core.ESDTNFTLatestNonceIdentifier)
)

const minNumOfArgsForCrossChainMint = 10

type esdtNFTCreateInput struct {
	esdtType              uint32
	quantity              *big.Int
	nonce                 uint64
	originalCreator       []byte
	uris                  [][]byte
	isCrossChainOperation bool
}

type esdtNFTCrossChainData struct {
	esdtType        uint32
	nonce           uint64
	originalCreator []byte
}

type esdtNFTCreate struct {
	baseAlwaysActiveHandler
	keyPrefix                     []byte
	accounts                      vmcommon.AccountsAdapter
	marshaller                    vmcommon.Marshalizer
	globalSettingsHandler         vmcommon.GlobalMetadataHandler
	rolesHandler                  vmcommon.ESDTRoleHandler
	funcGasCost                   uint64
	gasConfig                     vmcommon.BaseOperationCost
	esdtStorageHandler            vmcommon.ESDTNFTStorageHandler
	enableEpochsHandler           vmcommon.EnableEpochsHandler
	mutExecution                  sync.RWMutex
	crossChainTokenCheckerHandler CrossChainTokenCheckerHandler
	validEsdtTypes                map[uint32]struct{}
}

// ESDTNFTCreateFuncArgs is a struct placeholder for args needed to create the esdt nft create func
type ESDTNFTCreateFuncArgs struct {
	FuncGasCost                   uint64
	Marshaller                    vmcommon.Marshalizer
	RolesHandler                  vmcommon.ESDTRoleHandler
	EnableEpochsHandler           vmcommon.EnableEpochsHandler
	EsdtStorageHandler            vmcommon.ESDTNFTStorageHandler
	Accounts                      vmcommon.AccountsAdapter
	GasConfig                     vmcommon.BaseOperationCost
	GlobalSettingsHandler         vmcommon.GlobalMetadataHandler
	CrossChainTokenCheckerHandler CrossChainTokenCheckerHandler
}

// NewESDTNFTCreateFunc returns the esdt NFT create built-in function component
func NewESDTNFTCreateFunc(args ESDTNFTCreateFuncArgs) (*esdtNFTCreate, error) {
	if check.IfNil(args.Marshaller) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(args.GlobalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
	}
	if check.IfNil(args.RolesHandler) {
		return nil, ErrNilRolesHandler
	}
	if check.IfNil(args.EsdtStorageHandler) {
		return nil, ErrNilESDTNFTStorageHandler
	}
	if check.IfNil(args.EnableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}
	if check.IfNil(args.Accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if check.IfNil(args.CrossChainTokenCheckerHandler) {
		return nil, ErrNilCrossChainTokenChecker
	}

	e := &esdtNFTCreate{
		keyPrefix:                     []byte(baseESDTKeyPrefix),
		marshaller:                    args.Marshaller,
		globalSettingsHandler:         args.GlobalSettingsHandler,
		rolesHandler:                  args.RolesHandler,
		funcGasCost:                   args.FuncGasCost,
		gasConfig:                     args.GasConfig,
		esdtStorageHandler:            args.EsdtStorageHandler,
		enableEpochsHandler:           args.EnableEpochsHandler,
		mutExecution:                  sync.RWMutex{},
		accounts:                      args.Accounts,
		crossChainTokenCheckerHandler: args.CrossChainTokenCheckerHandler,
		validEsdtTypes:                getAllESDTTypes(),
	}

	return e, nil
}

func getAllESDTTypes() map[uint32]struct{} {
	esdtTypes := make(map[uint32]struct{})

	esdtTypes[uint32(core.NonFungibleV2)] = struct{}{}
	esdtTypes[uint32(core.SemiFungible)] = struct{}{}
	esdtTypes[uint32(core.MetaFungible)] = struct{}{}
	esdtTypes[uint32(core.DynamicNFT)] = struct{}{}
	esdtTypes[uint32(core.DynamicSFT)] = struct{}{}
	esdtTypes[uint32(core.DynamicMeta)] = struct{}{}

	return esdtTypes
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtNFTCreate) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.ESDTNFTCreate
	e.gasConfig = gasCost.BaseOperationCost
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves ESDT NFT create function call
// Requires at least 7 arguments:
// arg0 - token identifier
// arg1 - initial quantity
// arg2 - NFT name
// arg3 - Royalties - max 10000
// arg4 - hash
// arg5 - attributes
// arg6+ - multiple entries of URI (minimum 1)
// In case of cross chain operation, we need 3 more args:
// extraArg1 - token type
// extraArg2 - token nonce
// extraArg3 - creator from originating chain
// For ExecOnDestByCaller, last arg should be sc address caller
func (e *esdtNFTCreate) ProcessBuiltinFunction(
	acntSnd, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	e.mutExecution.RLock()
	defer e.mutExecution.RUnlock()

	err := checkESDTNFTCreateBurnAddInput(acntSnd, vmInput, e.funcGasCost)
	if err != nil {
		return nil, err
	}

	minNumOfArgs := 7
	if vmInput.CallType == vm.ExecOnDestByCaller {
		minNumOfArgs = 8
	}
	argsLen := len(vmInput.Arguments)
	if argsLen < minNumOfArgs {
		return nil, fmt.Errorf("%w, wrong number of arguments", ErrInvalidArguments)
	}

	accountWithRoles := acntSnd
	uris := vmInput.Arguments[6:]
	if vmInput.CallType == vm.ExecOnDestByCaller {
		scAddressWithRoles := vmInput.Arguments[argsLen-1]
		uris = vmInput.Arguments[6 : argsLen-1]

		if len(scAddressWithRoles) != len(vmInput.CallerAddr) {
			return nil, ErrInvalidAddressLength
		}
		if bytes.Equal(scAddressWithRoles, vmInput.CallerAddr) {
			return nil, ErrInvalidRcvAddr
		}

		accountWithRoles, err = e.getAccount(scAddressWithRoles)
		if err != nil {
			return nil, err
		}
	}

	tokenID := vmInput.Arguments[0]
	err = e.rolesHandler.CheckAllowedToExecute(accountWithRoles, vmInput.Arguments[0], []byte(core.ESDTRoleNFTCreate))
	if err != nil {
		return nil, err
	}

	createInput, err := e.getESDTNFTCreateInput(vmInput, tokenID, uris, accountWithRoles)
	if err != nil {
		return nil, err
	}

	esdtType, quantity, nonce, originalCreator, uris, isCrossChainToken :=
		createInput.esdtType,
		createInput.quantity,
		createInput.nonce,
		createInput.originalCreator,
		createInput.uris,
		createInput.isCrossChainOperation

	totalLength := uint64(0)
	for _, arg := range vmInput.Arguments {
		totalLength += uint64(len(arg))
	}
	gasToUse := totalLength*e.gasConfig.StorePerByte + e.funcGasCost
	if vmInput.GasProvided < gasToUse {
		return nil, ErrNotEnoughGas
	}

	royalties := uint32(big.NewInt(0).SetBytes(vmInput.Arguments[3]).Uint64())
	if royalties > core.MaxRoyalty {
		return nil, fmt.Errorf("%w, invalid max royality value", ErrInvalidArguments)
	}

	esdtTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	if quantity.Cmp(big.NewInt(1)) > 0 {
		err = e.rolesHandler.CheckAllowedToExecute(accountWithRoles, vmInput.Arguments[0], []byte(core.ESDTRoleNFTAddQuantity))
		if err != nil {
			return nil, err
		}
	}
	isValueLengthCheckFlagEnabled := e.enableEpochsHandler.IsFlagEnabled(ValueLengthCheckFlag)
	if isValueLengthCheckFlagEnabled && len(vmInput.Arguments[1]) > maxLenForAddNFTQuantity {
		return nil, fmt.Errorf("%w max length for quantity in nft create is %d", ErrInvalidArguments, maxLenForAddNFTQuantity)
	}

	nextNonce := nonce
	if !isCrossChainToken {
		nextNonce = nonce + 1
	}

	esdtData := &esdt.ESDigitalToken{
		Type:  esdtType,
		Value: quantity,
		TokenMetaData: &esdt.MetaData{
			Nonce:      nextNonce,
			Name:       vmInput.Arguments[2],
			Creator:    originalCreator,
			Royalties:  royalties,
			Hash:       vmInput.Arguments[4],
			Attributes: vmInput.Arguments[5],
			URIs:       uris,
		},
	}

	properties := vmcommon.NftSaveArgs{
		MustUpdateAllFields:         true,
		IsReturnWithError:           vmInput.ReturnCallAfterError,
		KeepMetaDataOnZeroLiquidity: false,
	}
	_, err = e.esdtStorageHandler.SaveESDTNFTToken(accountWithRoles.AddressBytes(), accountWithRoles, esdtTokenKey, nextNonce, esdtData, properties)
	if err != nil {
		return nil, err
	}
	err = e.esdtStorageHandler.AddToLiquiditySystemAcc(esdtTokenKey, esdtData.Type, nextNonce, quantity, false)
	if err != nil {
		return nil, err
	}

	if !isCrossChainToken {
		err = saveLatestNonce(accountWithRoles, tokenID, nextNonce)
		if err != nil {
			return nil, err
		}
	}

	if vmInput.CallType == vm.ExecOnDestByCaller {
		err = e.accounts.SaveAccount(accountWithRoles)
		if err != nil {
			return nil, err
		}
	}

	vmOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: vmInput.GasProvided - gasToUse,
		ReturnData:   [][]byte{big.NewInt(0).SetUint64(nextNonce).Bytes()},
	}

	esdtDataBytes, err := e.marshaller.Marshal(esdtData)
	if err != nil {
		log.Warn("esdtNFTCreate.ProcessBuiltinFunction: cannot marshall esdt data for log", "error", err)
	}

	addESDTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionESDTNFTCreate), vmInput.Arguments[0], nextNonce, quantity, vmInput.CallerAddr, esdtDataBytes)

	return vmOutput, nil
}

func (e *esdtNFTCreate) getTokenType(tokenID []byte) (uint32, error) {
	if !e.enableEpochsHandler.IsFlagEnabled(DynamicEsdtFlag) {
		return uint32(core.NonFungible), nil
	}

	esdtTokenKey := append([]byte(baseESDTKeyPrefix), tokenID...)
	return e.globalSettingsHandler.GetTokenType(esdtTokenKey)
}

func (e *esdtNFTCreate) getAccount(address []byte) (vmcommon.UserAccountHandler, error) {
	account, err := e.accounts.LoadAccount(address)
	if err != nil {
		return nil, err
	}

	userAcc, ok := account.(vmcommon.UserAccountHandler)
	if !ok {
		return nil, ErrWrongTypeAssertion
	}

	return userAcc, nil
}

func getLatestNonce(acnt vmcommon.UserAccountHandler, tokenID []byte) (uint64, error) {
	nonceKey := getNonceKey(tokenID)
	nonceData, _, err := acnt.AccountDataHandler().RetrieveValue(nonceKey)
	if err != nil {
		return 0, err
	}

	if len(nonceData) == 0 {
		return 0, nil
	}

	return big.NewInt(0).SetBytes(nonceData).Uint64(), nil
}

func (e *esdtNFTCreate) getESDTNFTCreateInput(
	vmInput *vmcommon.ContractCallInput,
	tokenID []byte,
	originalURIs [][]byte,
	accountWithRoles vmcommon.UserAccountHandler,
) (*esdtNFTCreateInput, error) {
	args := vmInput.Arguments

	var uris = originalURIs
	var esdtType uint32
	var nonce uint64
	var originalCreator []byte
	var err error
	quantity := big.NewInt(0).SetBytes(vmInput.Arguments[1])
	if quantity.Cmp(zero) <= 0 {
		return nil, fmt.Errorf("%w, invalid quantity", ErrInvalidArguments)
	}

	isCrossChainToken := e.crossChainTokenCheckerHandler.IsCrossChainOperation(tokenID)
	if !isCrossChainToken {
		esdtType, err = e.getTokenType(tokenID)
		if err != nil {
			return nil, err
		}

		nonce, err = getLatestNonce(accountWithRoles, tokenID)
		if err != nil {
			return nil, err
		}

		originalCreator = vmInput.CallerAddr
	} else {
		esdtData, err := getCrossChainESDTData(args, vmInput.CallType)
		if err != nil {
			return nil, err
		}

		err = e.validateEsdtType(esdtData.esdtType)
		if err != nil {
			return nil, err
		}

		err = e.validateQuantity(quantity, esdtData.esdtType)
		if err != nil {
			return nil, err
		}

		esdtType, nonce, originalCreator =
			esdtData.esdtType,
			esdtData.nonce,
			esdtData.originalCreator
		uris = uris[:len(uris)-3]
	}

	return &esdtNFTCreateInput{
		esdtType:              esdtType,
		quantity:              quantity,
		nonce:                 nonce,
		originalCreator:       originalCreator,
		uris:                  uris,
		isCrossChainOperation: isCrossChainToken,
	}, nil
}

func getCrossChainESDTData(args [][]byte, callType vm.CallType) (*esdtNFTCrossChainData, error) {
	minRequiredArgs := minNumOfArgsForCrossChainMint
	if callType == vm.ExecOnDestByCaller {
		minRequiredArgs++
	}

	argsLen := len(args)
	if argsLen < minRequiredArgs {
		return nil, fmt.Errorf("%w for cross chain token mint, received: %d, expected: %d, 2 extra arguments should be the nonce and original creator",
			ErrInvalidNumberOfArguments, argsLen, minRequiredArgs)
	}

	if !(callType == vm.ExecOnDestByCaller) {
		return &esdtNFTCrossChainData{
			esdtType:        uint32(getUIn46FromBytes(args[argsLen-3])),
			nonce:           getUIn46FromBytes(args[argsLen-2]),
			originalCreator: args[argsLen-1],
		}, nil
	}

	return &esdtNFTCrossChainData{
		esdtType:        uint32(getUIn46FromBytes(args[argsLen-4])),
		nonce:           getUIn46FromBytes(args[argsLen-3]),
		originalCreator: args[argsLen-2],
	}, nil
}

func getUIn46FromBytes(value []byte) uint64 {
	return big.NewInt(0).SetBytes(value).Uint64()
}

func saveLatestNonce(acnt vmcommon.UserAccountHandler, tokenID []byte, nonce uint64) error {
	nonceKey := getNonceKey(tokenID)
	return acnt.AccountDataHandler().SaveKeyValue(nonceKey, big.NewInt(0).SetUint64(nonce).Bytes())
}

func computeESDTNFTTokenKey(esdtTokenKey []byte, nonce uint64) []byte {
	return append(esdtTokenKey, big.NewInt(0).SetUint64(nonce).Bytes()...)
}

func checkESDTNFTCreateBurnAddInput(
	account vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
	funcGasCost uint64,
) error {
	err := checkBasicESDTArguments(vmInput)
	if err != nil {
		return err
	}
	if !bytes.Equal(vmInput.CallerAddr, vmInput.RecipientAddr) {
		return ErrInvalidRcvAddr
	}
	if check.IfNil(account) && vmInput.CallType != vm.ExecOnDestByCaller {
		return ErrNilUserAccount
	}
	if vmInput.GasProvided < funcGasCost {
		return ErrNotEnoughGas
	}
	return nil
}

func getNonceKey(tokenID []byte) []byte {
	return append(noncePrefix, tokenID...)
}

func (e *esdtNFTCreate) validateEsdtType(esdtType uint32) error {
	if _, isValid := e.validEsdtTypes[esdtType]; !isValid {
		return fmt.Errorf("%w, invalid esdt type %d (%s)", ErrInvalidArguments, esdtType, core.ESDTType(esdtType).String())
	}
	return nil
}

func isNonFungibleTokenType(esdtType uint32) bool {
	switch core.ESDTType(esdtType) {
	case core.NonFungible, core.NonFungibleV2, core.DynamicNFT:
		return true
	default:
		return false
	}
}

func (e *esdtNFTCreate) validateQuantity(quantity *big.Int, esdtType uint32) error {
	if isNonFungibleTokenType(esdtType) && quantity.Cmp(big.NewInt(1)) != 0 {
		return fmt.Errorf("%w, invalid quantity for esdt type %d (%s)", ErrInvalidArguments, esdtType, core.ESDTType(esdtType).String())
	}
	return nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtNFTCreate) IsInterfaceNil() bool {
	return e == nil
}
