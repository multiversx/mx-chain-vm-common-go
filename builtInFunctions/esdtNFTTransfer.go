package builtInFunctions

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data/esdt"
	"github.com/ElrondNetwork/elrond-go-core/data/vm"
	"github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/atomic"
)

type esdtNFTTransfer struct {
	baseAlwaysActive
	keyPrefix                 []byte
	marshalizer               vmcommon.Marshalizer
	globalSettingsHandler     vmcommon.ESDTGlobalSettingsHandler
	payableHandler            vmcommon.PayableHandler
	funcGasCost               uint64
	accounts                  vmcommon.AccountsAdapter
	shardCoordinator          vmcommon.Coordinator
	gasConfig                 vmcommon.BaseOperationCost
	mutExecution              sync.RWMutex
	rolesHandler              vmcommon.ESDTRoleHandler
	transferToMetaEnableEpoch uint32
	flagTransferToMeta        atomic.Flag
}

// NewESDTNFTTransferFunc returns the esdt NFT transfer built-in function component
func NewESDTNFTTransferFunc(
	funcGasCost uint64,
	marshalizer vmcommon.Marshalizer,
	globalSettingsHandler vmcommon.ESDTGlobalSettingsHandler,
	accounts vmcommon.AccountsAdapter,
	shardCoordinator vmcommon.Coordinator,
	gasConfig vmcommon.BaseOperationCost,
	rolesHandler vmcommon.ESDTRoleHandler,
	transferToMetaEnableEpoch uint32,
	epochNotifier vmcommon.EpochNotifier,
) (*esdtNFTTransfer, error) {
	if check.IfNil(marshalizer) {
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
	if check.IfNil(epochNotifier) {
		return nil, ErrNilEpochHandler
	}

	e := &esdtNFTTransfer{
		keyPrefix:                 []byte(core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier),
		marshalizer:               marshalizer,
		globalSettingsHandler:     globalSettingsHandler,
		funcGasCost:               funcGasCost,
		accounts:                  accounts,
		shardCoordinator:          shardCoordinator,
		gasConfig:                 gasConfig,
		mutExecution:              sync.RWMutex{},
		payableHandler:            &disabledPayableHandler{},
		rolesHandler:              rolesHandler,
		transferToMetaEnableEpoch: transferToMetaEnableEpoch,
	}

	epochNotifier.RegisterNotifyHandler(e)

	return e, nil
}

// EpochConfirmed is called whenever a new epoch is confirmed
func (e *esdtNFTTransfer) EpochConfirmed(epoch uint32, _ uint64) {
	e.flagTransferToMeta.Toggle(epoch >= e.transferToMetaEnableEpoch)
	log.Debug("ESDT NFT transfer to metachain enabled", e.flagTransferToMeta.IsSet())
}

// SetPayableHandler will set the payable handler to the function
func (e *esdtNFTTransfer) SetPayableHandler(payableHandler vmcommon.PayableHandler) error {
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
	if !check.IfNil(acntSnd) {
		return nil, ErrInvalidRcvAddr
	}
	if check.IfNil(acntDst) {
		return nil, ErrInvalidRcvAddr
	}

	esdtTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	marshaledNFTTransfer := vmInput.Arguments[3]
	esdtTransferData := &esdt.ESDigitalToken{}
	err = e.marshalizer.Unmarshal(esdtTransferData, marshaledNFTTransfer)
	if err != nil {
		return nil, err
	}

	err = e.addNFTToDestination(vmInput.RecipientAddr, acntDst, esdtTransferData, esdtTokenKey, mustVerifyPayable(vmInput, core.MinLenArgumentsESDTNFTTransfer), vmInput.ReturnCallAfterError)
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
			vmInput.CallerAddr,
			string(vmInput.Arguments[core.MinLenArgumentsESDTNFTTransfer]),
			callArgs,
			vmInput.RecipientAddr,
			vmInput.GasLocked,
			vmInput.CallType,
			vmOutput)
	}

	tokenNonce := esdtTransferData.TokenMetaData.Nonce

	addESDTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionESDTNFTTransfer), vmInput.Arguments[0], tokenNonce, esdtTransferData.Value, vmInput.CallerAddr, acntDst.AddressBytes())

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
	isInvalidTransferToMeta := e.shardCoordinator.ComputeId(dstAddress) == core.MetachainShardId && !e.flagTransferToMeta.IsSet()
	if isInvalidTransferToMeta {
		return nil, ErrInvalidRcvAddr
	}
	if vmInput.GasProvided < e.funcGasCost {
		return nil, ErrNotEnoughGas
	}

	esdtTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	nonce := big.NewInt(0).SetBytes(vmInput.Arguments[1]).Uint64()
	if nonce == 0 {
		return nil, ErrNFTDoesNotHaveMetadata
	}
	esdtData, err := getESDTNFTTokenOnSender(acntSnd, esdtTokenKey, nonce, e.marshalizer)
	if err != nil {
		return nil, err
	}

	quantityToTransfer := big.NewInt(0).SetBytes(vmInput.Arguments[2])
	if esdtData.Value.Cmp(quantityToTransfer) < 0 {
		return nil, ErrInvalidNFTQuantity
	}
	esdtData.Value.Sub(esdtData.Value, quantityToTransfer)

	_, err = saveESDTNFTToken(acntSnd, esdtTokenKey, esdtData, e.marshalizer, e.globalSettingsHandler, vmInput.ReturnCallAfterError)
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

		err = e.addNFTToDestination(dstAddress, userAccount, esdtData, esdtTokenKey, mustVerifyPayable(vmInput, core.MinLenArgumentsESDTNFTTransfer), vmInput.ReturnCallAfterError)
		if err != nil {
			return nil, err
		}

		err = e.accounts.SaveAccount(userAccount)
		if err != nil {
			return nil, err
		}
	}

	err = checkIfTransferCanHappenWithLimitedTransfer(esdtTokenKey, e.globalSettingsHandler, e.rolesHandler, acntSnd, userAccount, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: vmInput.GasProvided - e.funcGasCost,
	}
	err = e.createNFTOutputTransfers(vmInput, vmOutput, esdtData, dstAddress)
	if err != nil {
		return nil, err
	}

	tokenNonce := esdtData.TokenMetaData.Nonce
	addESDTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionESDTNFTTransfer), vmInput.Arguments[0], tokenNonce, quantityToTransfer, vmInput.CallerAddr, dstAddress)

	return vmOutput, nil
}

func (e *esdtNFTTransfer) createNFTOutputTransfers(
	vmInput *vmcommon.ContractCallInput,
	vmOutput *vmcommon.VMOutput,
	esdtTransferData *esdt.ESDigitalToken,
	dstAddress []byte,
) error {
	marshaledNFTTransfer, err := e.marshalizer.Marshal(esdtTransferData)
	if err != nil {
		return err
	}

	gasForTransfer := uint64(len(marshaledNFTTransfer)) * e.gasConfig.DataCopyPerByte
	if gasForTransfer > vmOutput.GasRemaining {
		return ErrNotEnoughGas
	}
	vmOutput.GasRemaining -= gasForTransfer

	nftTransferCallArgs := make([][]byte, 0)
	nftTransferCallArgs = append(nftTransferCallArgs, vmInput.Arguments[:3]...)
	nftTransferCallArgs = append(nftTransferCallArgs, marshaledNFTTransfer)
	if len(vmInput.Arguments) > core.MinLenArgumentsESDTNFTTransfer {
		nftTransferCallArgs = append(nftTransferCallArgs, vmInput.Arguments[4:]...)
	}

	isSCCallAfter := len(vmInput.Arguments) > core.MinLenArgumentsESDTNFTTransfer && vmcommon.IsSmartContractAddress(dstAddress)

	if e.shardCoordinator.SelfId() != e.shardCoordinator.ComputeId(dstAddress) {
		gasToTransfer := uint64(0)
		if isSCCallAfter {
			gasToTransfer = vmOutput.GasRemaining
			vmOutput.GasRemaining = 0
		}
		addNFTTransferToVMOutput(
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
	dstAddress []byte,
	userAccount vmcommon.UserAccountHandler,
	esdtDataToTransfer *esdt.ESDigitalToken,
	esdtTokenKey []byte,
	mustVerifyPayable bool,
	isReturnWithError bool,
) error {
	if mustVerifyPayable {
		isPayable, errIsPayable := e.payableHandler.IsPayable(dstAddress)
		if errIsPayable != nil {
			return errIsPayable
		}
		if !isPayable {
			return ErrAccountNotPayable
		}
	}

	nonce := uint64(0)
	if esdtDataToTransfer.TokenMetaData != nil {
		nonce = esdtDataToTransfer.TokenMetaData.Nonce
	}

	currentESDTData, _, err := getESDTNFTTokenOnDestination(userAccount, esdtTokenKey, nonce, e.marshalizer)
	if err != nil && !errors.Is(err, ErrNFTTokenDoesNotExist) {
		return err
	}
	err = checkFrozeAndPause(dstAddress, esdtTokenKey, currentESDTData, e.globalSettingsHandler, isReturnWithError)
	if err != nil {
		return err
	}

	if currentESDTData.TokenMetaData != nil {
		if !bytes.Equal(currentESDTData.TokenMetaData.Hash, esdtDataToTransfer.TokenMetaData.Hash) {
			return ErrWrongNFTOnDestination
		}
	}
	esdtDataToTransfer.Value.Add(esdtDataToTransfer.Value, currentESDTData.Value)

	_, err = saveESDTNFTToken(userAccount, esdtTokenKey, esdtDataToTransfer, e.marshalizer, e.globalSettingsHandler, isReturnWithError)
	if err != nil {
		return err
	}

	return nil
}

func addNFTTransferToVMOutput(
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
