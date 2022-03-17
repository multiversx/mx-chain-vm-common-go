package builtInFunctions

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/atomic"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data/esdt"
	"github.com/ElrondNetwork/elrond-vm-common"
)

type esdtNFTMultiTransfer struct {
	*baseEnabled
	keyPrefix                 []byte
	marshalizer               vmcommon.Marshalizer
	globalSettingsHandler     vmcommon.ESDTGlobalSettingsHandler
	payableHandler            vmcommon.PayableHandler
	funcGasCost               uint64
	accounts                  vmcommon.AccountsAdapter
	shardCoordinator          vmcommon.Coordinator
	gasConfig                 vmcommon.BaseOperationCost
	mutExecution              sync.RWMutex
	esdtStorageHandler        vmcommon.ESDTNFTStorageHandler
	rolesHandler              vmcommon.ESDTRoleHandler
	transferToMetaEnableEpoch uint32
	flagTransferToMeta        atomic.Flag
}

const argumentsPerTransfer = uint64(3)

// NewESDTNFTMultiTransferFunc returns the esdt NFT multi transfer built-in function component
func NewESDTNFTMultiTransferFunc(
	funcGasCost uint64,
	marshalizer vmcommon.Marshalizer,
	globalSettingsHandler vmcommon.ESDTGlobalSettingsHandler,
	accounts vmcommon.AccountsAdapter,
	shardCoordinator vmcommon.Coordinator,
	gasConfig vmcommon.BaseOperationCost,
	activationEpoch uint32,
	epochNotifier vmcommon.EpochNotifier,
	roleHandler vmcommon.ESDTRoleHandler,
	transferToMetaEnableEpoch uint32,
	esdtStorageHandler vmcommon.ESDTNFTStorageHandler,
) (*esdtNFTMultiTransfer, error) {
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
	if check.IfNil(epochNotifier) {
		return nil, ErrNilEpochHandler
	}
	if check.IfNil(roleHandler) {
		return nil, ErrNilRolesHandler
	}
	if check.IfNil(esdtStorageHandler) {
		return nil, ErrNilESDTNFTStorageHandler
	}

	e := &esdtNFTMultiTransfer{
		keyPrefix:                 []byte(core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier),
		marshalizer:               marshalizer,
		globalSettingsHandler:     globalSettingsHandler,
		funcGasCost:               funcGasCost,
		accounts:                  accounts,
		shardCoordinator:          shardCoordinator,
		gasConfig:                 gasConfig,
		mutExecution:              sync.RWMutex{},
		payableHandler:            &disabledPayableHandler{},
		rolesHandler:              roleHandler,
		transferToMetaEnableEpoch: transferToMetaEnableEpoch,
		esdtStorageHandler:        esdtStorageHandler,
	}

	e.baseEnabled = &baseEnabled{
		function:        core.BuiltInFunctionMultiESDTNFTTransfer,
		activationEpoch: activationEpoch,
		flagActivated:   atomic.Flag{},
	}

	epochNotifier.RegisterNotifyHandler(e)

	return e, nil
}

// EpochConfirmed is called whenever a new epoch is confirmed
func (e *esdtNFTMultiTransfer) EpochConfirmed(epoch uint32, nonce uint64) {
	e.baseEnabled.EpochConfirmed(epoch, nonce)
	e.flagTransferToMeta.SetValue(epoch >= e.transferToMetaEnableEpoch)
	log.Debug("ESDT NFT transfer to metachain flag", "enabled", e.flagTransferToMeta.IsSet())
}

// SetPayableHandler will set the payable handler to the function
func (e *esdtNFTMultiTransfer) SetPayableHandler(payableHandler vmcommon.PayableHandler) error {
	if check.IfNil(payableHandler) {
		return ErrNilPayableHandler
	}

	e.payableHandler = payableHandler
	return nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtNFTMultiTransfer) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.ESDTNFTMultiTransfer
	e.gasConfig = gasCost.BaseOperationCost
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves ESDT NFT transfer roles function call
// Requires the following arguments:
// arg0 - destination address
// arg1 - number of tokens to transfer
// list of (tokenID - nonce - quantity) - in case of ESDT nonce == 0
// function and list of arguments for SC Call
// if cross-shard, the rest of arguments will be filled inside the SCR
// arg0 - number of tokens to transfer
// list of (tokenID - nonce - quantity/ESDT NFT data)
// function and list of arguments for SC Call
func (e *esdtNFTMultiTransfer) ProcessBuiltinFunction(
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
		return e.processESDTNFTMultiTransferOnSenderShard(acntSnd, vmInput)
	}

	// in cross shard NFT transfer the sender account must be nil
	if !check.IfNil(acntSnd) {
		return nil, ErrInvalidRcvAddr
	}
	if check.IfNil(acntDst) {
		return nil, ErrInvalidRcvAddr
	}

	numOfTransfers := big.NewInt(0).SetBytes(vmInput.Arguments[0]).Uint64()
	if numOfTransfers == 0 {
		return nil, fmt.Errorf("%w, 0 tokens to transfer", ErrInvalidArguments)
	}
	minNumOfArguments := numOfTransfers*argumentsPerTransfer + 1
	if uint64(len(vmInput.Arguments)) < minNumOfArguments {
		return nil, fmt.Errorf("%w, invalid number of arguments", ErrInvalidArguments)
	}

	verifyPayable := mustVerifyPayable(vmInput, int(minNumOfArguments))
	vmOutput := &vmcommon.VMOutput{GasRemaining: vmInput.GasProvided}
	vmOutput.Logs = make([]*vmcommon.LogEntry, 0, numOfTransfers)
	startIndex := uint64(1)

	err = e.checkIfPayable(verifyPayable, vmInput.CallerAddr, vmInput.RecipientAddr)
	if err != nil {
		return nil, err
	}

	for i := uint64(0); i < numOfTransfers; i++ {
		tokenStartIndex := startIndex + i*argumentsPerTransfer
		tokenID := vmInput.Arguments[tokenStartIndex]
		nonce := big.NewInt(0).SetBytes(vmInput.Arguments[tokenStartIndex+1]).Uint64()

		esdtTokenKey := append(e.keyPrefix, tokenID...)

		value := big.NewInt(0)
		if nonce > 0 {
			esdtTransferData := &esdt.ESDigitalToken{}
			if len(vmInput.Arguments[tokenStartIndex+2]) > vmcommon.MaxLengthForValueToOptTransfer {
				marshaledNFTTransfer := vmInput.Arguments[tokenStartIndex+2]
				err = e.marshalizer.Unmarshal(esdtTransferData, marshaledNFTTransfer)
				if err != nil {
					return nil, fmt.Errorf("%w for token %s", err, string(tokenID))
				}
			} else {
				esdtTransferData.Value = big.NewInt(0).SetBytes(vmInput.Arguments[tokenStartIndex+2])
				esdtTransferData.Type = uint32(core.NonFungible)
			}

			value.Set(esdtTransferData.Value)
			err = e.addNFTToDestination(
				vmInput.CallerAddr,
				vmInput.RecipientAddr,
				acntDst,
				esdtTransferData,
				esdtTokenKey,
				nonce,
				vmInput.ReturnCallAfterError)
			if err != nil {
				return nil, fmt.Errorf("%w for token %s", err, string(tokenID))
			}
		} else {
			transferredValue := big.NewInt(0).SetBytes(vmInput.Arguments[tokenStartIndex+2])
			err = addToESDTBalance(acntDst, esdtTokenKey, transferredValue, e.marshalizer, e.globalSettingsHandler, vmInput.ReturnCallAfterError)
			if err != nil {
				return nil, fmt.Errorf("%w for token %s", err, string(tokenID))
			}
			value = transferredValue
		}

		addESDTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionMultiESDTNFTTransfer), tokenID, nonce, value, vmInput.CallerAddr, acntDst.AddressBytes())
	}

	// no need to consume gas on destination - sender already paid for it
	if len(vmInput.Arguments) > int(minNumOfArguments) && vmcommon.IsSmartContractAddress(vmInput.RecipientAddr) {
		var callArgs [][]byte
		if len(vmInput.Arguments) > int(minNumOfArguments)+1 {
			callArgs = vmInput.Arguments[minNumOfArguments+1:]
		}

		addOutputTransferToVMOutput(
			vmInput.CallerAddr,
			string(vmInput.Arguments[minNumOfArguments]),
			callArgs,
			vmInput.RecipientAddr,
			vmInput.GasLocked,
			vmInput.CallType,
			vmOutput)
	}

	return vmOutput, nil
}

func (e *esdtNFTMultiTransfer) processESDTNFTMultiTransferOnSenderShard(
	acntSnd vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	dstAddress := vmInput.Arguments[0]
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
	numOfTransfers := big.NewInt(0).SetBytes(vmInput.Arguments[1]).Uint64()
	if numOfTransfers == 0 {
		return nil, fmt.Errorf("%w, 0 tokens to transfer", ErrInvalidArguments)
	}
	minNumOfArguments := numOfTransfers*argumentsPerTransfer + 2
	if uint64(len(vmInput.Arguments)) < minNumOfArguments {
		return nil, fmt.Errorf("%w, invalid number of arguments", ErrInvalidArguments)
	}

	multiTransferCost := numOfTransfers * e.funcGasCost
	if vmInput.GasProvided < multiTransferCost {
		return nil, ErrNotEnoughGas
	}

	verifyPayable := mustVerifyPayable(vmInput, int(minNumOfArguments))
	acntDst, err := e.loadAccountIfInShard(dstAddress)
	if err != nil {
		return nil, err
	}

	if !check.IfNil(acntDst) {
		err = e.checkIfPayable(verifyPayable, vmInput.CallerAddr, dstAddress)
		if err != nil {
			return nil, err
		}
	}

	vmOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: vmInput.GasProvided - multiTransferCost,
		Logs:         make([]*vmcommon.LogEntry, 0, numOfTransfers),
	}

	startIndex := uint64(2)
	listEsdtData := make([]*esdt.ESDigitalToken, numOfTransfers)
	listTransferData := make([]*vmcommon.ESDTTransfer, numOfTransfers)

	for i := uint64(0); i < numOfTransfers; i++ {
		tokenStartIndex := startIndex + i*argumentsPerTransfer
		listTransferData[i] = &vmcommon.ESDTTransfer{
			ESDTValue:      big.NewInt(0).SetBytes(vmInput.Arguments[tokenStartIndex+2]),
			ESDTTokenName:  vmInput.Arguments[tokenStartIndex],
			ESDTTokenType:  0,
			ESDTTokenNonce: big.NewInt(0).SetBytes(vmInput.Arguments[tokenStartIndex+1]).Uint64(),
		}
		if listTransferData[i].ESDTTokenNonce > 0 {
			listTransferData[i].ESDTTokenType = uint32(core.NonFungible)
		}

		listEsdtData[i], err = e.transferOneTokenOnSenderShard(
			acntSnd,
			acntDst,
			dstAddress,
			listTransferData[i],
			vmInput.ReturnCallAfterError)
		if err != nil {
			return nil, fmt.Errorf("%w for token %s", err, string(listTransferData[i].ESDTTokenName))
		}

		addESDTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionMultiESDTNFTTransfer), listTransferData[i].ESDTTokenName, listTransferData[i].ESDTTokenNonce, listTransferData[i].ESDTValue, vmInput.CallerAddr, dstAddress)
	}

	if !check.IfNil(acntDst) {
		err = e.accounts.SaveAccount(acntDst)
		if err != nil {
			return nil, err
		}
	}

	err = e.createESDTNFTOutputTransfers(vmInput, vmOutput, listEsdtData, listTransferData, dstAddress)
	if err != nil {
		return nil, err
	}

	return vmOutput, nil
}

func (e *esdtNFTMultiTransfer) transferOneTokenOnSenderShard(
	acntSnd vmcommon.UserAccountHandler,
	acntDst vmcommon.UserAccountHandler,
	dstAddress []byte,
	transferData *vmcommon.ESDTTransfer,
	isReturnCallWithError bool,
) (*esdt.ESDigitalToken, error) {
	if transferData.ESDTValue.Cmp(zero) <= 0 {
		return nil, ErrInvalidNFTQuantity
	}

	esdtTokenKey := append(e.keyPrefix, transferData.ESDTTokenName...)
	esdtData, err := e.esdtStorageHandler.GetESDTNFTTokenOnSender(acntSnd, esdtTokenKey, transferData.ESDTTokenNonce)
	if err != nil {
		return nil, err
	}

	if esdtData.Value.Cmp(transferData.ESDTValue) < 0 {
		return nil, computeInsufficientQuantityESDTError(transferData.ESDTTokenName, transferData.ESDTTokenNonce)
	}
	esdtData.Value.Sub(esdtData.Value, transferData.ESDTValue)

	_, err = e.esdtStorageHandler.SaveESDTNFTToken(acntSnd.AddressBytes(), acntSnd, esdtTokenKey, transferData.ESDTTokenNonce, esdtData, false, isReturnCallWithError)
	if err != nil {
		return nil, err
	}

	esdtData.Value.Set(transferData.ESDTValue)

	err = checkIfTransferCanHappenWithLimitedTransfer(esdtTokenKey, e.globalSettingsHandler, e.rolesHandler, acntSnd, acntDst, isReturnCallWithError)
	if err != nil {
		return nil, err
	}

	if !check.IfNil(acntDst) {
		err = e.addNFTToDestination(acntSnd.AddressBytes(), dstAddress, acntDst, esdtData, esdtTokenKey, transferData.ESDTTokenNonce, isReturnCallWithError)
		if err != nil {
			return nil, err
		}
	}

	return esdtData, nil
}

func computeInsufficientQuantityESDTError(tokenID []byte, nonce uint64) error {
	err := fmt.Errorf("%w for token: %s", ErrInsufficientQuantityESDT, string(tokenID))
	if nonce > 0 {
		err = fmt.Errorf("%w nonce %d", err, nonce)
	}

	return err
}

func (e *esdtNFTMultiTransfer) loadAccountIfInShard(dstAddress []byte) (vmcommon.UserAccountHandler, error) {
	if e.shardCoordinator.SelfId() != e.shardCoordinator.ComputeId(dstAddress) {
		return nil, nil
	}

	accountHandler, errLoad := e.accounts.LoadAccount(dstAddress)
	if errLoad != nil {
		return nil, errLoad
	}
	userAccount, ok := accountHandler.(vmcommon.UserAccountHandler)
	if !ok {
		return nil, ErrWrongTypeAssertion
	}

	return userAccount, nil
}

func (e *esdtNFTMultiTransfer) createESDTNFTOutputTransfers(
	vmInput *vmcommon.ContractCallInput,
	vmOutput *vmcommon.VMOutput,
	listESDTData []*esdt.ESDigitalToken,
	listESDTTransfers []*vmcommon.ESDTTransfer,
	dstAddress []byte,
) error {
	multiTransferCallArgs := make([][]byte, 0, argumentsPerTransfer*uint64(len(listESDTTransfers))+1)
	numTokenTransfer := big.NewInt(int64(len(listESDTTransfers))).Bytes()
	multiTransferCallArgs = append(multiTransferCallArgs, numTokenTransfer)

	for i, esdtTransfer := range listESDTTransfers {
		multiTransferCallArgs = append(multiTransferCallArgs, esdtTransfer.ESDTTokenName)
		nonceAsBytes := []byte{0}
		if esdtTransfer.ESDTTokenNonce > 0 {
			nonceAsBytes = big.NewInt(0).SetUint64(esdtTransfer.ESDTTokenNonce).Bytes()
		}
		multiTransferCallArgs = append(multiTransferCallArgs, nonceAsBytes)

		if esdtTransfer.ESDTTokenNonce > 0 {
			wasAlreadySent, err := e.esdtStorageHandler.WasAlreadySentToDestinationShardAndUpdateState(esdtTransfer.ESDTTokenName, esdtTransfer.ESDTTokenNonce, dstAddress)
			if err != nil {
				return err
			}

			sendCrossShardAsMarshalledData := !wasAlreadySent || esdtTransfer.ESDTValue.Cmp(oneValue) == 0 ||
				len(esdtTransfer.ESDTValue.Bytes()) > vmcommon.MaxLengthForValueToOptTransfer
			if sendCrossShardAsMarshalledData {
				marshaledNFTTransfer, err := e.marshalizer.Marshal(listESDTData[i])
				if err != nil {
					return err
				}

				gasForTransfer := uint64(len(marshaledNFTTransfer)) * e.gasConfig.DataCopyPerByte
				if gasForTransfer > vmOutput.GasRemaining {
					return ErrNotEnoughGas
				}
				vmOutput.GasRemaining -= gasForTransfer

				multiTransferCallArgs = append(multiTransferCallArgs, marshaledNFTTransfer)
			} else {
				multiTransferCallArgs = append(multiTransferCallArgs, esdtTransfer.ESDTValue.Bytes())
			}

		} else {
			multiTransferCallArgs = append(multiTransferCallArgs, esdtTransfer.ESDTValue.Bytes())
		}
	}

	minNumOfArguments := uint64(len(listESDTTransfers))*argumentsPerTransfer + 2
	if uint64(len(vmInput.Arguments)) > minNumOfArguments {
		multiTransferCallArgs = append(multiTransferCallArgs, vmInput.Arguments[minNumOfArguments:]...)
	}

	isSCCallAfter := determineIsSCCallAfter(vmInput, dstAddress, int(minNumOfArguments))

	if e.shardCoordinator.SelfId() != e.shardCoordinator.ComputeId(dstAddress) {
		gasToTransfer := uint64(0)
		if isSCCallAfter {
			gasToTransfer = vmOutput.GasRemaining
			vmOutput.GasRemaining = 0
		}
		addNFTTransferToVMOutput(
			vmInput.CallerAddr,
			dstAddress,
			core.BuiltInFunctionMultiESDTNFTTransfer,
			multiTransferCallArgs,
			vmInput.GasLocked,
			gasToTransfer,
			vmInput.CallType,
			vmOutput,
		)

		return nil
	}

	if isSCCallAfter {
		var callArgs [][]byte
		if uint64(len(vmInput.Arguments)) > minNumOfArguments+1 {
			callArgs = vmInput.Arguments[minNumOfArguments+1:]
		}

		addOutputTransferToVMOutput(
			vmInput.CallerAddr,
			string(vmInput.Arguments[minNumOfArguments]),
			callArgs,
			dstAddress,
			vmInput.GasLocked,
			vmInput.CallType,
			vmOutput)
	}

	return nil
}

func (e *esdtNFTMultiTransfer) checkIfPayable(
	mustVerifyPayable bool,
	sndAddress []byte,
	dstAddress []byte,
) error {
	if !mustVerifyPayable {
		return nil
	}

	isPayable, errIsPayable := e.payableHandler.IsPayable(sndAddress, dstAddress)
	if errIsPayable != nil {
		return errIsPayable
	}
	if !isPayable {
		return ErrAccountNotPayable
	}

	return nil
}

func (e *esdtNFTMultiTransfer) addNFTToDestination(
	sndAddress []byte,
	dstAddress []byte,
	userAccount vmcommon.UserAccountHandler,
	esdtDataToTransfer *esdt.ESDigitalToken,
	esdtTokenKey []byte,
	nonce uint64,
	isReturnCallWithError bool,
) error {
	currentESDTData, _, err := e.esdtStorageHandler.GetESDTNFTTokenOnDestination(userAccount, esdtTokenKey, nonce)
	if err != nil && !errors.Is(err, ErrNFTTokenDoesNotExist) {
		return err
	}
	err = checkFrozeAndPause(dstAddress, esdtTokenKey, currentESDTData, e.globalSettingsHandler, isReturnCallWithError)
	if err != nil {
		return err
	}

	esdtDataToTransfer.Value.Add(esdtDataToTransfer.Value, currentESDTData.Value)

	_, err = e.esdtStorageHandler.SaveESDTNFTToken(sndAddress, userAccount, esdtTokenKey, nonce, esdtDataToTransfer, false, isReturnCallWithError)
	if err != nil {
		return err
	}

	return nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtNFTMultiTransfer) IsInterfaceNil() bool {
	return e == nil
}
