package builtInFunctions

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"sync"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data/esdt"
	"github.com/ElrondNetwork/elrond-go-core/data/vm"
	"github.com/ElrondNetwork/elrond-vm-common"
)

var zero = big.NewInt(0)

type esdtTransfer struct {
	baseAlwaysActive
	funcGasCost      uint64
	marshalizer      vmcommon.Marshalizer
	keyPrefix        []byte
	pauseHandler     vmcommon.ESDTPauseHandler
	payableHandler   vmcommon.PayableHandler
	shardCoordinator vmcommon.Coordinator
	mutExecution     sync.RWMutex
}

// NewESDTTransferFunc returns the esdt transfer built-in function component
func NewESDTTransferFunc(
	funcGasCost uint64,
	marshalizer vmcommon.Marshalizer,
	pauseHandler vmcommon.ESDTPauseHandler,
	shardCoordinator vmcommon.Coordinator,
) (*esdtTransfer, error) {
	if check.IfNil(marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(pauseHandler) {
		return nil, ErrNilPauseHandler
	}
	if check.IfNil(shardCoordinator) {
		return nil, ErrNilShardCoordinator
	}

	e := &esdtTransfer{
		funcGasCost:      funcGasCost,
		marshalizer:      marshalizer,
		keyPrefix:        []byte(vmcommon.ElrondProtectedKeyPrefix + vmcommon.ESDTKeyIdentifier),
		pauseHandler:     pauseHandler,
		payableHandler:   &disabledPayableHandler{},
		shardCoordinator: shardCoordinator,
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtTransfer) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.ESDTTransfer
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves ESDT transfer function calls
func (e *esdtTransfer) ProcessBuiltinFunction(
	acntSnd, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	e.mutExecution.RLock()
	defer e.mutExecution.RUnlock()

	err := checkBasicESDTArguments(vmInput)
	if err != nil {
		return nil, err
	}
	if e.shardCoordinator.ComputeId(vmInput.RecipientAddr) == vmcommon.MetachainShardId {
		return nil, ErrInvalidRcvAddr
	}

	value := big.NewInt(0).SetBytes(vmInput.Arguments[1])
	if value.Cmp(zero) <= 0 {
		return nil, ErrNegativeValue
	}

	gasRemaining := computeGasRemaining(acntSnd, vmInput.GasProvided, e.funcGasCost)
	esdtTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	tokenID := vmInput.Arguments[0]

	if !check.IfNil(acntSnd) {
		// gas is paid only by sender
		if vmInput.GasProvided < e.funcGasCost {
			return nil, ErrNotEnoughGas
		}

		err = addToESDTBalance(acntSnd, esdtTokenKey, big.NewInt(0).Neg(value), e.marshalizer, e.pauseHandler, vmInput.ReturnCallAfterError)
		if err != nil {
			return nil, err
		}
	}

	isSCCallAfter := vmcommon.IsSmartContractAddress(vmInput.RecipientAddr) && len(vmInput.Arguments) > vmcommon.MinLenArgumentsESDTTransfer

	vmOutput := &vmcommon.VMOutput{GasRemaining: gasRemaining, ReturnCode: vmcommon.Ok}
	if !check.IfNil(acntDst) {
		if mustVerifyPayable(vmInput, vmcommon.MinLenArgumentsESDTTransfer) {
			isPayable, errPayable := e.payableHandler.IsPayable(vmInput.RecipientAddr)
			if errPayable != nil {
				return nil, errPayable
			}
			if !isPayable {
				return nil, ErrAccountNotPayable
			}
		}

		err = addToESDTBalance(acntDst, esdtTokenKey, value, e.marshalizer, e.pauseHandler, vmInput.ReturnCallAfterError)
		if err != nil {
			return nil, err
		}

		if isSCCallAfter {
			vmOutput.GasRemaining, err = vmcommon.SafeSubUint64(vmInput.GasProvided, e.funcGasCost)
			var callArgs [][]byte
			if len(vmInput.Arguments) > vmcommon.MinLenArgumentsESDTTransfer+1 {
				callArgs = vmInput.Arguments[vmcommon.MinLenArgumentsESDTTransfer+1:]
			}

			addOutputTransferToVMOutput(
				vmInput.CallerAddr,
				string(vmInput.Arguments[vmcommon.MinLenArgumentsESDTTransfer]),
				callArgs,
				vmInput.RecipientAddr,
				vmInput.GasLocked,
				vmInput.CallType,
				vmOutput)

			addESDTEntryInVMOutput(vmOutput, []byte(vmcommon.BuiltInFunctionESDTTransfer), tokenID, value, vmInput.CallerAddr, acntDst.AddressBytes())
			return vmOutput, nil
		}

		if vmInput.CallType == vm.AsynchronousCallBack && check.IfNil(acntSnd) {
			// gas was already consumed on sender shard
			vmOutput.GasRemaining = vmInput.GasProvided
		}

		addESDTEntryInVMOutput(vmOutput, []byte(vmcommon.BuiltInFunctionESDTTransfer), tokenID, value, vmInput.CallerAddr, acntDst.AddressBytes())
		return vmOutput, nil
	}

	// cross-shard ESDT transfer call through a smart contract
	if vmcommon.IsSmartContractAddress(vmInput.CallerAddr) {
		addOutputTransferToVMOutput(
			vmInput.CallerAddr,
			vmcommon.BuiltInFunctionESDTTransfer,
			vmInput.Arguments,
			vmInput.RecipientAddr,
			vmInput.GasLocked,
			vmInput.CallType,
			vmOutput)
	}

	addESDTEntryInVMOutput(vmOutput, []byte(vmcommon.BuiltInFunctionESDTTransfer), tokenID, value, vmInput.CallerAddr)
	return vmOutput, nil
}

func mustVerifyPayable(vmInput *vmcommon.ContractCallInput, minLenArguments int) bool {
	if vmInput.CallType == vm.AsynchronousCall || vmInput.CallType == vm.ESDTTransferAndExecute {
		return false
	}
	if bytes.Equal(vmInput.CallerAddr, vmcommon.ESDTSCAddress) {
		return false
	}

	if len(vmInput.Arguments) > minLenArguments {
		return false
	}

	return true
}

func addOutputTransferToVMOutput(
	senderAddress []byte,
	function string,
	arguments [][]byte,
	recipient []byte,
	gasLocked uint64,
	callType vm.CallType,
	vmOutput *vmcommon.VMOutput,
) {
	esdtTransferTxData := function
	for _, arg := range arguments {
		esdtTransferTxData += "@" + hex.EncodeToString(arg)
	}
	outTransfer := vmcommon.OutputTransfer{
		Value:         big.NewInt(0),
		GasLimit:      vmOutput.GasRemaining,
		GasLocked:     gasLocked,
		Data:          []byte(esdtTransferTxData),
		CallType:      callType,
		SenderAddress: senderAddress,
	}
	vmOutput.OutputAccounts = make(map[string]*vmcommon.OutputAccount)
	vmOutput.OutputAccounts[string(recipient)] = &vmcommon.OutputAccount{
		Address:         recipient,
		OutputTransfers: []vmcommon.OutputTransfer{outTransfer},
	}
	vmOutput.GasRemaining = 0
}

func addToESDTBalance(
	userAcnt vmcommon.UserAccountHandler,
	key []byte,
	value *big.Int,
	marshalizer vmcommon.Marshalizer,
	pauseHandler vmcommon.ESDTPauseHandler,
	isReturnWithError bool,
) error {
	esdtData, err := getESDTDataFromKey(userAcnt, key, marshalizer)
	if err != nil {
		return err
	}

	if esdtData.Type != uint32(vmcommon.Fungible) {
		return ErrOnlyFungibleTokensHaveBalanceTransfer
	}

	err = checkFrozeAndPause(userAcnt.AddressBytes(), key, esdtData, pauseHandler, isReturnWithError)
	if err != nil {
		return err
	}

	esdtData.Value.Add(esdtData.Value, value)
	if esdtData.Value.Cmp(zero) < 0 {
		return ErrInsufficientFunds
	}

	err = saveESDTData(userAcnt, esdtData, key, marshalizer)
	if err != nil {
		return err
	}

	return nil
}

func checkFrozeAndPause(
	senderAddr []byte,
	key []byte,
	esdtData *esdt.ESDigitalToken,
	pauseHandler vmcommon.ESDTPauseHandler,
	isReturnWithError bool,
) error {
	if isReturnWithError {
		return nil
	}
	if bytes.Equal(senderAddr, vmcommon.ESDTSCAddress) {
		return nil
	}

	esdtUserMetaData := ESDTUserMetadataFromBytes(esdtData.Properties)
	if esdtUserMetaData.Frozen {
		return ErrESDTIsFrozenForAccount
	}

	if pauseHandler.IsPaused(key) {
		return ErrESDTTokenIsPaused
	}

	return nil
}

func arePropertiesEmpty(properties []byte) bool {
	for _, property := range properties {
		if property != 0 {
			return false
		}
	}
	return true
}

func saveESDTData(
	userAcnt vmcommon.UserAccountHandler,
	esdtData *esdt.ESDigitalToken,
	key []byte,
	marshalizer vmcommon.Marshalizer,
) error {
	isValueZero := esdtData.Value.Cmp(zero) == 0
	if isValueZero && arePropertiesEmpty(esdtData.Properties) {
		return userAcnt.AccountDataHandler().SaveKeyValue(key, nil)
	}

	marshaledData, err := marshalizer.Marshal(esdtData)
	if err != nil {
		return err
	}

	return userAcnt.AccountDataHandler().SaveKeyValue(key, marshaledData)
}

func getESDTDataFromKey(
	userAcnt vmcommon.UserAccountHandler,
	key []byte,
	marshalizer vmcommon.Marshalizer,
) (*esdt.ESDigitalToken, error) {
	esdtData := &esdt.ESDigitalToken{Value: big.NewInt(0), Type: uint32(vmcommon.Fungible)}
	marshaledData, err := userAcnt.AccountDataHandler().RetrieveValue(key)
	if err != nil || len(marshaledData) == 0 {
		return esdtData, nil
	}

	err = marshalizer.Unmarshal(esdtData, marshaledData)
	if err != nil {
		return nil, err
	}

	return esdtData, nil
}

// SetPayableHandler will set the payable handler to the function
func (e *esdtTransfer) SetPayableHandler(payableHandler vmcommon.PayableHandler) error {
	if check.IfNil(payableHandler) {
		return ErrNilPayableHandler
	}

	e.payableHandler = payableHandler
	return nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtTransfer) IsInterfaceNil() bool {
	return e == nil
}
