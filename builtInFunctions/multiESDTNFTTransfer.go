package builtInFunctions

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/atomic"
	"github.com/ElrondNetwork/elrond-vm-common/check"
	"github.com/ElrondNetwork/elrond-vm-common/data/esdt"
)

type esdtNFTMultiTransfer struct {
	*baseEnabled
	keyPrefix        []byte
	marshalizer      vmcommon.Marshalizer
	pauseHandler     vmcommon.ESDTPauseHandler
	payableHandler   vmcommon.PayableHandler
	funcGasCost      uint64
	accounts         vmcommon.AccountsAdapter
	shardCoordinator vmcommon.Coordinator
	gasConfig        vmcommon.BaseOperationCost
	mutExecution     sync.RWMutex
}

const argumentsPerTransfer = uint64(3)

// NewESDTNFTMultiTransferFunc returns the esdt NFT multi transfer built-in function component
func NewESDTNFTMultiTransferFunc(
	funcGasCost uint64,
	marshalizer vmcommon.Marshalizer,
	pauseHandler vmcommon.ESDTPauseHandler,
	accounts vmcommon.AccountsAdapter,
	shardCoordinator vmcommon.Coordinator,
	gasConfig vmcommon.BaseOperationCost,
	activationEpoch uint32,
	epochNotifier vmcommon.EpochNotifier,
) (*esdtNFTMultiTransfer, error) {
	if check.IfNil(marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(pauseHandler) {
		return nil, ErrNilPauseHandler
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

	e := &esdtNFTMultiTransfer{
		keyPrefix:        []byte(vmcommon.ElrondProtectedKeyPrefix + vmcommon.ESDTKeyIdentifier),
		marshalizer:      marshalizer,
		pauseHandler:     pauseHandler,
		funcGasCost:      funcGasCost,
		accounts:         accounts,
		shardCoordinator: shardCoordinator,
		gasConfig:        gasConfig,
		mutExecution:     sync.RWMutex{},
		payableHandler:   &disabledPayableHandler{},
	}

	e.baseEnabled = &baseEnabled{
		function:        vmcommon.BuiltInFunctionESDTNFTAddURI,
		activationEpoch: activationEpoch,
		flagActivated:   atomic.Flag{},
	}

	epochNotifier.RegisterNotifyHandler(e)

	return e, nil
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
	e.funcGasCost = gasCost.BuiltInCost.ESDTNFTTransfer
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
	if len(vmInput.Arguments) < 5 {
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

	esdtTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	marshalledNFTTransfer := vmInput.Arguments[3]
	esdtTransferData := &esdt.ESDigitalToken{}
	err = e.marshalizer.Unmarshal(esdtTransferData, marshalledNFTTransfer)
	if err != nil {
		return nil, err
	}

	err = e.addNFTToDestination(vmInput.RecipientAddr, acntDst, esdtTransferData, esdtTokenKey, mustVerifyPayable(vmInput, vmcommon.MinLenArgumentsESDTNFTTransfer))
	if err != nil {
		return nil, err
	}

	// no need to consume gas on destination - sender already paid for it
	vmOutput := &vmcommon.VMOutput{GasRemaining: vmInput.GasProvided}
	if len(vmInput.Arguments) > vmcommon.MinLenArgumentsESDTNFTTransfer && vmcommon.IsSmartContractAddress(vmInput.RecipientAddr) {
		var callArgs [][]byte
		if len(vmInput.Arguments) > vmcommon.MinLenArgumentsESDTNFTTransfer+1 {
			callArgs = vmInput.Arguments[vmcommon.MinLenArgumentsESDTNFTTransfer+1:]
		}

		addOutputTransferToVMOutput(
			vmInput.CallerAddr,
			string(vmInput.Arguments[vmcommon.MinLenArgumentsESDTNFTTransfer]),
			callArgs,
			vmInput.RecipientAddr,
			vmInput.GasLocked,
			vmInput.CallType,
			vmOutput)
	}

	tokenNonce := esdtTransferData.TokenMetaData.Nonce
	logEntry := newEntryForNFT(vmcommon.BuiltInFunctionESDTNFTTransfer, vmInput.CallerAddr, vmInput.Arguments[0], tokenNonce)
	logEntry.Topics = append(logEntry.Topics, acntDst.AddressBytes())
	vmOutput.Logs = []*vmcommon.LogEntry{logEntry}

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
	if e.shardCoordinator.ComputeId(dstAddress) == vmcommon.MetachainShardId {
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

	startIndex := uint64(2)
	listEsdtData := make([]*esdt.ESDigitalToken, numOfTransfers)
	listTokenID := make([][]byte, numOfTransfers)
	for i := uint64(0); i < numOfTransfers; i++ {
		tokenStartIndex := startIndex + i*argumentsPerTransfer
		listTokenID[i] = vmInput.Arguments[tokenStartIndex]
		nonce := big.NewInt(0).SetBytes(vmInput.Arguments[tokenStartIndex+1]).Uint64()
		quantityToTransfer := big.NewInt(0).SetBytes(vmInput.Arguments[tokenStartIndex+2])
		listEsdtData[i], err = e.transferOneTokenOnSenderShard(acntSnd, acntDst, dstAddress, listTokenID[i], nonce, quantityToTransfer, verifyPayable)
		if err != nil {
			return nil, err
		}
	}

	vmOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: vmInput.GasProvided - multiTransferCost,
	}

	err = e.createESDTNFTOutputTransfers(vmInput, vmOutput, listEsdtData, listTokenID, dstAddress)
	if err != nil {
		return nil, err
	}

	return vmOutput, nil
}

func (e *esdtNFTMultiTransfer) transferOneTokenOnSenderShard(
	acntSnd vmcommon.UserAccountHandler,
	acntDst vmcommon.UserAccountHandler,
	dstAddress []byte,
	tokenID []byte,
	nonce uint64,
	quantityToTransfer *big.Int,
	verifyPayable bool,
) (*esdt.ESDigitalToken, error) {
	if quantityToTransfer.Cmp(zero) <= 0 {
		return nil, ErrInvalidNFTQuantity
	}

	esdtTokenKey := append(e.keyPrefix, tokenID...)
	esdtData, err := getESDTNFTTokenOnSender(acntSnd, esdtTokenKey, nonce, e.marshalizer)
	if err != nil {
		return nil, err
	}

	if esdtData.Value.Cmp(quantityToTransfer) < 0 {
		return nil, ErrInvalidNFTQuantity
	}
	esdtData.Value.Sub(esdtData.Value, quantityToTransfer)

	err = saveESDTNFTToken(acntSnd, esdtTokenKey, esdtData, e.marshalizer, e.pauseHandler)
	if err != nil {
		return nil, err
	}

	esdtData.Value.Set(quantityToTransfer)

	if !check.IfNil(acntDst) {
		err = e.addNFTToDestination(dstAddress, acntDst, esdtData, esdtTokenKey, verifyPayable)
		if err != nil {
			return nil, err
		}
	}

	return esdtData, nil
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
	listESDTTransferData []*esdt.ESDigitalToken,
	listTokenIDs [][]byte,
	dstAddress []byte,
) error {

	multiTransferCallArgs := make([][]byte, 0, argumentsPerTransfer*uint64(len(listESDTTransferData))+1)
	numTokenTransfer := big.NewInt(int64(len(listESDTTransferData))).Bytes()
	multiTransferCallArgs = append(multiTransferCallArgs, numTokenTransfer)

	for i, esdtTransferData := range listESDTTransferData {

		multiTransferCallArgs = append(multiTransferCallArgs, listTokenIDs[i])
		if esdtTransferData.TokenMetaData != nil {
			marshalledNFTTransfer, err := e.marshalizer.Marshal(esdtTransferData)
			if err != nil {
				return err
			}

			gasForTransfer := uint64(len(marshalledNFTTransfer)) * e.gasConfig.DataCopyPerByte
			if gasForTransfer > vmOutput.GasRemaining {
				return ErrNotEnoughGas
			}
			vmOutput.GasRemaining -= gasForTransfer

			multiTransferCallArgs = append(multiTransferCallArgs, big.NewInt(0).SetUint64(esdtTransferData.TokenMetaData.Nonce).Bytes())
			multiTransferCallArgs = append(multiTransferCallArgs, marshalledNFTTransfer)
		} else {
			multiTransferCallArgs = append(multiTransferCallArgs, []byte{0})
			multiTransferCallArgs = append(multiTransferCallArgs, esdtTransferData.Value.Bytes())
		}
	}

	minNumOfArguments := uint64(len(listESDTTransferData))*argumentsPerTransfer + 2
	if uint64(len(vmInput.Arguments)) > minNumOfArguments {
		multiTransferCallArgs = append(multiTransferCallArgs, vmInput.Arguments[minNumOfArguments:]...)
	}

	isSCCallAfter := uint64(len(vmInput.Arguments)) > minNumOfArguments && vmcommon.IsSmartContractAddress(dstAddress)

	if e.shardCoordinator.SelfId() != e.shardCoordinator.ComputeId(dstAddress) {
		gasToTransfer := uint64(0)
		if isSCCallAfter {
			gasToTransfer = vmOutput.GasRemaining
			vmOutput.GasRemaining = 0
		}
		addNFTTransferToVMOutput(
			vmInput.CallerAddr,
			dstAddress,
			vmcommon.BuiltInFunctionMultiESDTNFTTransfer,
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

func (e *esdtNFTMultiTransfer) addNFTToDestination(
	dstAddress []byte,
	userAccount vmcommon.UserAccountHandler,
	esdtDataToTransfer *esdt.ESDigitalToken,
	esdtTokenKey []byte,
	mustVerifyPayable bool,
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

	currentESDTData, _, err := getESDTNFTTokenOnDestination(userAccount, esdtTokenKey, esdtDataToTransfer.TokenMetaData.Nonce, e.marshalizer)
	if err != nil && !errors.Is(err, ErrNFTTokenDoesNotExist) {
		return err
	}
	err = checkFrozeAndPause(dstAddress, esdtTokenKey, currentESDTData, e.pauseHandler)
	if err != nil {
		return err
	}

	if currentESDTData.TokenMetaData != nil {
		if !bytes.Equal(currentESDTData.TokenMetaData.Hash, esdtDataToTransfer.TokenMetaData.Hash) {
			return ErrWrongNFTOnDestination
		}
		esdtDataToTransfer.Value.Add(esdtDataToTransfer.Value, currentESDTData.Value)
	}

	err = saveESDTNFTToken(userAccount, esdtTokenKey, esdtDataToTransfer, e.marshalizer, e.pauseHandler)
	if err != nil {
		return err
	}

	return nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtNFTMultiTransfer) IsInterfaceNil() bool {
	return e == nil
}
