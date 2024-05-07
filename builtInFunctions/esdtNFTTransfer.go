package builtInFunctions

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data/esdt"
	"github.com/multiversx/mx-chain-core-go/data/vm"

	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

const baseESDTKeyPrefix = core.ProtectedKeyPrefix + core.ESDTKeyIdentifier

var oneValue = big.NewInt(1)
var zeroByteArray = []byte{0}

type esdtNFTTransfer struct {
	baseAlwaysActiveHandler
	*baseComponentsHolder
	keyPrefix      []byte
	marshaller     vmcommon.Marshalizer
	payableHandler vmcommon.PayableChecker
	funcGasCost    uint64
	accounts       vmcommon.AccountsAdapter
	gasConfig      vmcommon.BaseOperationCost
	mutExecution   sync.RWMutex
	rolesHandler   vmcommon.ESDTRoleHandler
}

// NewESDTNFTTransferFunc returns the esdt NFT transfer built-in function component
func NewESDTNFTTransferFunc(
	funcGasCost uint64,
	marshaller vmcommon.Marshalizer,
	globalSettingsHandler vmcommon.GlobalMetadataHandler,
	accounts vmcommon.AccountsAdapter,
	shardCoordinator vmcommon.Coordinator,
	gasConfig vmcommon.BaseOperationCost,
	rolesHandler vmcommon.ESDTRoleHandler,
	esdtStorageHandler vmcommon.ESDTNFTStorageHandler,
	enableEpochsHandler vmcommon.EnableEpochsHandler,
) (*esdtNFTTransfer, error) {
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(globalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
	}
	if check.IfNil(accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if check.IfNil(shardCoordinator) {
		return nil, ErrNilShardCoordinator
	}
	if check.IfNil(rolesHandler) {
		return nil, ErrNilRolesHandler
	}
	if check.IfNil(enableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}
	if check.IfNil(esdtStorageHandler) {
		return nil, ErrNilESDTNFTStorageHandler
	}

	e := &esdtNFTTransfer{
		keyPrefix:      []byte(baseESDTKeyPrefix),
		marshaller:     marshaller,
		funcGasCost:    funcGasCost,
		accounts:       accounts,
		gasConfig:      gasConfig,
		mutExecution:   sync.RWMutex{},
		payableHandler: &disabledPayableHandler{},
		rolesHandler:   rolesHandler,
		baseComponentsHolder: &baseComponentsHolder{
			esdtStorageHandler:    esdtStorageHandler,
			globalSettingsHandler: globalSettingsHandler,
			shardCoordinator:      shardCoordinator,
			enableEpochsHandler:   enableEpochsHandler,
		},
	}

	return e, nil
}

// SetPayableChecker will set the payableCheck handler to the function
func (e *esdtNFTTransfer) SetPayableChecker(payableHandler vmcommon.PayableChecker) error {
	if check.IfNil(payableHandler) {
		return ErrNilPayableHandler
	}

	e.payableHandler = payableHandler
	return nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtNFTTransfer) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.ESDTNFTTransfer
	e.gasConfig = gasCost.BaseOperationCost
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves ESDT NFT transfer roles function call
// Requires 4 arguments:
// arg0 - token identifier
// arg1 - nonce
// arg2 - quantity to transfer
// arg3 - destination address
// if cross-shard, the rest of arguments will be filled inside the SCR
func (e *esdtNFTTransfer) ProcessBuiltinFunction(
	acntSnd, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	e.mutExecution.RLock()
	defer e.mutExecution.RUnlock()

	err := checkBasicESDTArguments(vmInput)
	if err != nil {
		return nil, err
	}
	if len(vmInput.Arguments) < 4 {
		return nil, ErrInvalidArguments
	}

	if bytes.Equal(vmInput.CallerAddr, vmInput.RecipientAddr) {
		return e.processNFTTransferOnSenderShard(acntSnd, vmInput)
	}

	// in cross shard NFT transfer the sender account must be nil
	// or sender should be ESDTSCAddress in case of a sovereign scr
	isSenderESDTSCAddr := bytes.Equal(vmInput.CallerAddr, core.ESDTSCAddress)
	if !check.IfNil(acntSnd) && !isSenderESDTSCAddr {
		return nil, ErrInvalidRcvAddr
	}
	if check.IfNil(acntDst) {
		return nil, ErrInvalidRcvAddr
	}

	tickerID := vmInput.Arguments[0]
	esdtTokenKey := append(e.keyPrefix, tickerID...)
	nonce := big.NewInt(0).SetBytes(vmInput.Arguments[1]).Uint64()
	value := big.NewInt(0).SetBytes(vmInput.Arguments[2])

	esdtTransferData := &esdt.ESDigitalToken{}
	if !bytes.Equal(vmInput.Arguments[3], zeroByteArray) {
		marshaledNFTTransfer := vmInput.Arguments[3]
		err = e.marshaller.Unmarshal(esdtTransferData, marshaledNFTTransfer)
		if err != nil {
			return nil, err
		}
	} else {
		esdtTransferData.Value = big.NewInt(0).Set(value)
		esdtTransferData.Type = uint32(core.NonFungible)
	}

	err = e.payableHandler.CheckPayable(vmInput, vmInput.RecipientAddr, core.MinLenArgumentsESDTNFTTransfer)
	if err != nil {
		return nil, err
	}
	err = e.addNFTToDestination(
		vmInput.CallerAddr,
		vmInput.RecipientAddr,
		acntDst,
		esdtTransferData,
		esdtTokenKey,
		nonce,
		vmInput.ReturnCallAfterError,
		isSenderESDTSCAddr)
	if err != nil {
		return nil, err
	}

	// no need to consume gas on destination - sender already paid for it
	vmOutput := &vmcommon.VMOutput{GasRemaining: vmInput.GasProvided}
	if len(vmInput.Arguments) > core.MinLenArgumentsESDTNFTTransfer && vmcommon.IsSmartContractAddress(vmInput.RecipientAddr) {
		var callArgs [][]byte
		if len(vmInput.Arguments) > core.MinLenArgumentsESDTNFTTransfer+1 {
			callArgs = vmInput.Arguments[core.MinLenArgumentsESDTNFTTransfer+1:]
		}

		addOutputTransferToVMOutput(
			1,
			vmInput.CallerAddr,
			string(vmInput.Arguments[core.MinLenArgumentsESDTNFTTransfer]),
			callArgs,
			vmInput.RecipientAddr,
			vmInput.GasLocked,
			vmInput.CallType,
			vmOutput)
	}

	addESDTEntryForTransferInVMOutput(
		vmInput, vmOutput,
		[]byte(core.BuiltInFunctionESDTNFTTransfer),
		acntDst.AddressBytes(),
		[]*TopicTokenData{{
			vmInput.Arguments[0],
			nonce,
			value,
		}},
	)

	return vmOutput, nil
}

func (e *esdtNFTTransfer) processNFTTransferOnSenderShard(
	acntSnd vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	dstAddress := vmInput.Arguments[3]
	if len(dstAddress) != len(vmInput.CallerAddr) {
		return nil, fmt.Errorf("%w, not a valid destination address", ErrInvalidArguments)
	}
	if bytes.Equal(dstAddress, vmInput.CallerAddr) {
		return nil, fmt.Errorf("%w, can not transfer to self", ErrInvalidArguments)
	}
	isTransferToMeta := e.shardCoordinator.ComputeId(dstAddress) == core.MetachainShardId
	if isTransferToMeta {
		return nil, ErrInvalidRcvAddr
	}
	if vmInput.GasProvided < e.funcGasCost {
		return nil, ErrNotEnoughGas
	}

	tickerID := vmInput.Arguments[0]
	esdtTokenKey := append(e.keyPrefix, tickerID...)
	nonce := big.NewInt(0).SetBytes(vmInput.Arguments[1]).Uint64()
	esdtData, err := e.esdtStorageHandler.GetESDTNFTTokenOnSender(acntSnd, esdtTokenKey, nonce)
	if err != nil {
		return nil, err
	}
	if nonce == 0 {
		return nil, ErrNFTDoesNotHaveMetadata
	}

	if len(vmInput.Arguments[2]) > core.MaxLenForESDTIssueMint && e.enableEpochsHandler.IsFlagEnabled(ConsistentTokensValuesLengthCheckFlag) {
		return nil, fmt.Errorf("%w: max length for a transfer value is %d", ErrInvalidArguments, core.MaxLenForESDTIssueMint)
	}
	quantityToTransfer := big.NewInt(0).SetBytes(vmInput.Arguments[2])
	if esdtData.Value.Cmp(quantityToTransfer) < 0 {
		return nil, ErrInvalidNFTQuantity
	}

	isCheckTransferFlagEnabled := e.enableEpochsHandler.IsFlagEnabled(CheckTransferFlag)
	if isCheckTransferFlagEnabled && quantityToTransfer.Cmp(zero) <= 0 {
		return nil, ErrInvalidNFTQuantity
	}
	esdtData.Value.Sub(esdtData.Value, quantityToTransfer)

	_, err = e.esdtStorageHandler.SaveESDTNFTToken(
		acntSnd.AddressBytes(),
		acntSnd,
		esdtTokenKey,
		nonce,
		esdtData,
		false,
		vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	esdtData.Value.Set(quantityToTransfer)

	var userAccount vmcommon.UserAccountHandler
	if e.shardCoordinator.SelfId() == e.shardCoordinator.ComputeId(dstAddress) {
		accountHandler, errLoad := e.accounts.LoadAccount(dstAddress)
		if errLoad != nil {
			return nil, errLoad
		}

		var ok bool
		userAccount, ok = accountHandler.(vmcommon.UserAccountHandler)
		if !ok {
			return nil, ErrWrongTypeAssertion
		}

		err = e.payableHandler.CheckPayable(vmInput, dstAddress, core.MinLenArgumentsESDTNFTTransfer)
		if err != nil {
			return nil, err
		}
		err = e.addNFTToDestination(
			vmInput.CallerAddr,
			dstAddress,
			userAccount,
			esdtData,
			esdtTokenKey,
			nonce,
			vmInput.ReturnCallAfterError,
			false)
		if err != nil {
			return nil, err
		}

		err = e.accounts.SaveAccount(userAccount)
		if err != nil {
			return nil, err
		}
	} else {
		keepMetadataOnZeroLiquidity, err := hasDynamicRole(acntSnd, tickerID, e.marshaller)
		if err != nil {
			return nil, err
		}

		err = e.esdtStorageHandler.AddToLiquiditySystemAcc(esdtTokenKey, esdtData.Type, nonce, big.NewInt(0).Neg(quantityToTransfer), keepMetadataOnZeroLiquidity)
		if err != nil {
			return nil, err
		}
	}

	tokenID := esdtTokenKey
	if e.enableEpochsHandler.IsFlagEnabled(CheckCorrectTokenIDForTransferRoleFlag) {
		tokenID = tickerID
	}

	err = checkIfTransferCanHappenWithLimitedTransfer(tokenID, esdtTokenKey, acntSnd.AddressBytes(), dstAddress, e.globalSettingsHandler, e.rolesHandler, acntSnd, userAccount, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: vmInput.GasProvided - e.funcGasCost,
	}
	err = e.createNFTOutputTransfers(vmInput, vmOutput, esdtData, dstAddress, tickerID, nonce)
	if err != nil {
		return nil, err
	}

	addESDTEntryForTransferInVMOutput(
		vmInput, vmOutput,
		[]byte(core.BuiltInFunctionESDTNFTTransfer),
		dstAddress,
		[]*TopicTokenData{{
			vmInput.Arguments[0],
			nonce,
			quantityToTransfer,
		}},
	)

	return vmOutput, nil
}

func hasDynamicRole(account vmcommon.UserAccountHandler, tokenID []byte, marshaller vmcommon.Marshalizer) (bool, error) {
	roleKey := append(roleKeyPrefix, tokenID...)
	roles, _, err := getESDTRolesForAcnt(marshaller, account, roleKey)
	if err != nil {
		return false, err
	}

	dynamicRoles := [][]byte{
		[]byte(core.ESDTMetaDataRecreate),
		[]byte(core.ESDTRoleNFTUpdate),
		[]byte(core.ESDTRoleModifyCreator),
		[]byte(core.ESDTRoleModifyRoyalties),
		[]byte(core.ESDTRoleSetNewURI),
	}

	for _, role := range dynamicRoles {
		_, exists := doesRoleExist(roles, role)
		if exists {
			return true, nil
		}
	}

	return false, nil
}

func (e *esdtNFTTransfer) createNFTOutputTransfers(
	vmInput *vmcommon.ContractCallInput,
	vmOutput *vmcommon.VMOutput,
	esdtTransferData *esdt.ESDigitalToken,
	dstAddress []byte,
	tickerID []byte,
	nonce uint64,
) error {
	nftTransferCallArgs := make([][]byte, 0)
	nftTransferCallArgs = append(nftTransferCallArgs, vmInput.Arguments[:3]...)

	wasAlreadySent, err := e.esdtStorageHandler.WasAlreadySentToDestinationShardAndUpdateState(tickerID, nonce, dstAddress)
	if err != nil {
		return err
	}

	if !wasAlreadySent || esdtTransferData.Value.Cmp(oneValue) == 0 {
		marshaledNFTTransfer, err := e.marshaller.Marshal(esdtTransferData)
		if err != nil {
			return err
		}

		gasForTransfer := uint64(len(marshaledNFTTransfer)) * e.gasConfig.DataCopyPerByte
		if gasForTransfer > vmOutput.GasRemaining {
			return ErrNotEnoughGas
		}
		vmOutput.GasRemaining -= gasForTransfer
		nftTransferCallArgs = append(nftTransferCallArgs, marshaledNFTTransfer)
	} else {
		nftTransferCallArgs = append(nftTransferCallArgs, zeroByteArray)
	}

	if len(vmInput.Arguments) > core.MinLenArgumentsESDTNFTTransfer {
		nftTransferCallArgs = append(nftTransferCallArgs, vmInput.Arguments[4:]...)
	}

	isSCCallAfter := e.payableHandler.DetermineIsSCCallAfter(vmInput, dstAddress, core.MinLenArgumentsESDTNFTTransfer)

	if e.shardCoordinator.SelfId() != e.shardCoordinator.ComputeId(dstAddress) {
		gasToTransfer := uint64(0)
		if isSCCallAfter {
			gasToTransfer = vmOutput.GasRemaining
			vmOutput.GasRemaining = 0
		}
		addNFTTransferToVMOutput(
			1,
			vmInput.CallerAddr,
			dstAddress,
			core.BuiltInFunctionESDTNFTTransfer,
			nftTransferCallArgs,
			vmInput.GasLocked,
			gasToTransfer,
			vmInput.CallType,
			vmOutput,
		)

		return nil
	}

	if isSCCallAfter {
		var callArgs [][]byte
		if len(vmInput.Arguments) > core.MinLenArgumentsESDTNFTTransfer+1 {
			callArgs = vmInput.Arguments[core.MinLenArgumentsESDTNFTTransfer+1:]
		}

		addOutputTransferToVMOutput(
			1,
			vmInput.CallerAddr,
			string(vmInput.Arguments[core.MinLenArgumentsESDTNFTTransfer]),
			callArgs,
			dstAddress,
			vmInput.GasLocked,
			vmInput.CallType,
			vmOutput)
	}

	return nil
}

func (e *esdtNFTTransfer) addNFTToDestination(
	sndAddress []byte,
	dstAddress []byte,
	userAccount vmcommon.UserAccountHandler,
	esdtDataToTransfer *esdt.ESDigitalToken,
	esdtTokenKey []byte,
	nonce uint64,
	isReturnWithError bool,
	isSenderESDTSCAddr bool,
) error {
	currentESDTData, _, err := e.esdtStorageHandler.GetESDTNFTTokenOnDestination(userAccount, esdtTokenKey, nonce)
	if err != nil && !errors.Is(err, ErrNFTTokenDoesNotExist) {
		return err
	}
	err = checkFrozeAndPause(dstAddress, esdtTokenKey, currentESDTData, e.globalSettingsHandler, isReturnWithError)
	if err != nil {
		return err
	}

	transferValue := big.NewInt(0).Set(esdtDataToTransfer.Value)
	esdtDataToTransfer.Value.Add(esdtDataToTransfer.Value, currentESDTData.Value)
	_, err = e.esdtStorageHandler.SaveESDTNFTToken(sndAddress, userAccount, esdtTokenKey, nonce, esdtDataToTransfer, false, isReturnWithError)
	if err != nil {
		return err
	}

	isSameShard := e.shardCoordinator.SameShard(sndAddress, dstAddress)
	if !isSameShard || isSenderESDTSCAddr {
		err = e.esdtStorageHandler.AddToLiquiditySystemAcc(esdtTokenKey, currentESDTData.Type, nonce, transferValue, false)
		if err != nil {
			return err
		}
	}

	return nil
}

func addNFTTransferToVMOutput(
	index uint32,
	senderAddress []byte,
	recipient []byte,
	funcToCall string,
	arguments [][]byte,
	gasLocked uint64,
	gasLimit uint64,
	callType vm.CallType,
	vmOutput *vmcommon.VMOutput,
) {
	nftTransferTxData := funcToCall
	for _, arg := range arguments {
		nftTransferTxData += "@" + hex.EncodeToString(arg)
	}
	outTransfer := vmcommon.OutputTransfer{
		Index:         index,
		Value:         big.NewInt(0),
		GasLimit:      gasLimit,
		GasLocked:     gasLocked,
		Data:          []byte(nftTransferTxData),
		CallType:      callType,
		SenderAddress: senderAddress,
	}
	vmOutput.OutputAccounts = make(map[string]*vmcommon.OutputAccount)
	vmOutput.OutputAccounts[string(recipient)] = &vmcommon.OutputAccount{
		Address:         recipient,
		OutputTransfers: []vmcommon.OutputTransfer{outTransfer},
	}
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtNFTTransfer) IsInterfaceNil() bool {
	return e == nil
}
