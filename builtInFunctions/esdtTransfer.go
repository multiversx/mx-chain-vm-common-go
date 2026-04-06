package builtInFunctions

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data/esdt"
	"github.com/multiversx/mx-chain-core-go/data/vm"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

var zero = big.NewInt(0)

type esdtTransfer struct {
	baseAlwaysActiveHandler
	vmcommon.BlockchainDataProvider
	funcGasCost           uint64
	marshaller            vmcommon.Marshalizer
	keyPrefix             []byte
	globalSettingsHandler vmcommon.ExtendedESDTGlobalSettingsHandler
	payableHandler        vmcommon.PayableChecker
	shardCoordinator      vmcommon.Coordinator
	mutExecution          sync.RWMutex
	gasConfig             vmcommon.BaseOperationCost

	rolesHandler        vmcommon.ESDTRoleHandler
	enableEpochsHandler vmcommon.EnableEpochsHandler
	drwaReader          drwaStateReader
}

// NewESDTTransferFunc returns the esdt transfer built-in function component
func NewESDTTransferFunc(
	funcGasCost uint64,
	marshaller vmcommon.Marshalizer,
	globalSettingsHandler vmcommon.ExtendedESDTGlobalSettingsHandler,
	shardCoordinator vmcommon.Coordinator,
	rolesHandler vmcommon.ESDTRoleHandler,
	enableEpochsHandler vmcommon.EnableEpochsHandler,
) (*esdtTransfer, error) {
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(globalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
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

	e := &esdtTransfer{
		BlockchainDataProvider: NewBlockchainDataProvider(),
		funcGasCost:            funcGasCost,
		marshaller:             marshaller,
		keyPrefix:              []byte(baseESDTKeyPrefix),
		globalSettingsHandler:  globalSettingsHandler,
		payableHandler:         &disabledPayableHandler{},
		shardCoordinator:       shardCoordinator,
		rolesHandler:           rolesHandler,
		enableEpochsHandler:    enableEpochsHandler,
	}

	return e, nil
}

func (e *esdtTransfer) SetDRWAReader(reader drwaStateReader) {
	e.mutExecution.Lock()
	e.drwaReader = reader
	e.mutExecution.Unlock()
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtTransfer) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.ESDTTransfer
	e.gasConfig = gasCost.BaseOperationCost
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
	isTransferToMeta := e.shardCoordinator.ComputeId(vmInput.RecipientAddr) == core.MetachainShardId
	if isTransferToMeta {
		return nil, ErrInvalidRcvAddr
	}

	if e.enableEpochsHandler.IsFlagEnabled(ConsistentTokensValuesLengthCheckFlag) {
		if len(vmInput.Arguments[1]) > core.MaxLenForESDTIssueMint {
			return nil, fmt.Errorf("%w: max length for esdt transfer value is %d", ErrInvalidArguments, core.MaxLenForESDTIssueMint)
		}
	}
	value := big.NewInt(0).SetBytes(vmInput.Arguments[1])
	if value.Cmp(zero) <= 0 {
		return nil, ErrNegativeValue
	}

	skipGasUse := noGasUseIfReturnCallAfterErrorWithFlag(e.enableEpochsHandler, vmInput)
	gasToUse := e.funcGasCost
	esdtTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	tokenID := vmInput.Arguments[0]

	keyToCheck := esdtTokenKey
	if e.enableEpochsHandler.IsFlagEnabled(CheckCorrectTokenIDForTransferRoleFlag) {
		keyToCheck = tokenID
	}

	err = checkIfTransferCanHappenWithLimitedTransfer(keyToCheck, esdtTokenKey, vmInput.CallerAddr, vmInput.RecipientAddr, e.globalSettingsHandler, e.rolesHandler, acntSnd, acntDst, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}
	if isDRWAEnforcementEnabled(e.enableEpochsHandler) {
		// Go M-2 fix: Pre-charge maximum DRWA gas before performing state reads.
		// Prevents free trie reads as a DoS vector for regulated tokens.
		// Max cost: 4 reads (sender policy + sender holder + receiver policy + receiver holder).
		// DRWA gas model: conservatively pre-charge for 4 trie reads (max case: in-shard
		// regulated transfer checks both sender and receiver, 2 reads each). Cross-shard
		// transfers check only one side (2 reads); unused pre-charged gas is returned.
		drwaMaxGas := computeDRWAReadGasCost(e.gasConfig, e.funcGasCost, 4)
		if !skipGasUse && vmInput.GasProvided < gasToUse+drwaMaxGas {
			return nil, ErrNotEnoughGas
		}

		// C-2 fix: When account is nil (cross-shard), we still validate our side.
		// Source shard: acntSnd is present (validate sender), acntDst is nil (skip receiver — destination shard validates).
		// Destination shard: acntDst is present (validate receiver), acntSnd is nil (skip sender — source shard validated).
		// This is the correct cross-shard split per spec §11_DRWA_Cross_Shard_Enforcement.
		// The critical invariant: if BOTH are nil, deny the transfer (should never happen in valid protocol).
		if check.IfNil(acntSnd) && check.IfNil(acntDst) {
			// Both accounts nil means no shard can validate — reject for safety
			if e.drwaReader != nil {
				isRegulated, _, _ := isDRWARegulatedToken(e.drwaReader, tokenID)
				if isRegulated {
					return nil, fmt.Errorf("DRWA enforcement: both sender and receiver accounts nil for regulated token %s — cross-shard validation gap", string(tokenID))
				}
			}
		}
		if !check.IfNil(acntSnd) {
			regulated, drwaErr := evaluateDRWASenderTransfer(e.drwaReader, tokenID, vmInput.CallerAddr, acntSnd, e.CurrentRound())
			if regulated {
				gasToUse += computeDRWAReadGasCost(e.gasConfig, e.funcGasCost, 2)
			}
			err = drwaErr
			if err != nil {
				return nil, err
			}
		}
		if !check.IfNil(acntDst) {
			regulated, drwaErr := evaluateDRWAReceiverTransfer(e.drwaReader, tokenID, vmInput.RecipientAddr, acntDst, e.CurrentRound())
			if regulated {
				gasToUse += computeDRWAReadGasCost(e.gasConfig, e.funcGasCost, 2)
			}
			err = drwaErr
			if err != nil {
				return nil, err
			}
		}
	}
	gasRemaining := computeGasRemainingIfNeeded(acntSnd, vmInput.GasProvided, gasToUse, skipGasUse)

	if !check.IfNil(acntSnd) {
		// gas is paid only by sender
		if vmInput.GasProvided < gasToUse && !skipGasUse {
			return nil, ErrNotEnoughGas
		}

		err = addToESDTBalance(acntSnd, esdtTokenKey, big.NewInt(0).Neg(value), e.marshaller, e.globalSettingsHandler, vmInput.ReturnCallAfterError)
		if err != nil {
			return nil, err
		}
	}

	isSCCallAfter := e.payableHandler.DetermineIsSCCallAfter(vmInput, vmInput.RecipientAddr, core.MinLenArgumentsESDTTransfer)
	vmOutput := &vmcommon.VMOutput{GasRemaining: gasRemaining, ReturnCode: vmcommon.Ok}
	if !check.IfNil(acntDst) {
		err = e.payableHandler.CheckPayable(vmInput, vmInput.RecipientAddr, core.MinLenArgumentsESDTTransfer)
		if err != nil {
			return nil, err
		}

		err = addToESDTBalance(acntDst, esdtTokenKey, value, e.marshaller, e.globalSettingsHandler, vmInput.ReturnCallAfterError)
		if err != nil {
			return nil, err
		}

		if isSCCallAfter {
			vmOutput.GasRemaining, _ = vmcommon.SafeSubUint64(vmInput.GasProvided, gasToUse)
			var callArgs [][]byte
			if len(vmInput.Arguments) > core.MinLenArgumentsESDTTransfer+1 {
				callArgs = vmInput.Arguments[core.MinLenArgumentsESDTTransfer+1:]
			}

			addOutputTransferToVMOutput(
				1,
				vmInput.CallerAddr,
				string(vmInput.Arguments[core.MinLenArgumentsESDTTransfer]),
				callArgs,
				vmInput.RecipientAddr,
				vmInput.GasLocked,
				vmInput.CallType,
				vmOutput)

			addESDTEntryForTransferInVMOutput(
				vmInput, vmOutput,
				[]byte(core.BuiltInFunctionESDTTransfer),
				acntDst.AddressBytes(),
				[]*TopicTokenData{{
					tokenID,
					0,
					value,
				}},
			)
			return vmOutput, nil
		}

		if vmInput.CallType == vm.AsynchronousCallBack && check.IfNil(acntSnd) {
			// gas was already consumed on sender shard
			vmOutput.GasRemaining = vmInput.GasProvided
		}

		addESDTEntryForTransferInVMOutput(
			vmInput, vmOutput,
			[]byte(core.BuiltInFunctionESDTTransfer),
			acntDst.AddressBytes(),
			[]*TopicTokenData{{
				tokenID,
				0,
				value,
			}})
		return vmOutput, nil
	}

	// cross-shard ESDT transfer call through a smart contract
	if vmcommon.IsSmartContractAddress(vmInput.CallerAddr) {
		addOutputTransferToVMOutput(
			1,
			vmInput.CallerAddr,
			core.BuiltInFunctionESDTTransfer,
			vmInput.Arguments,
			vmInput.RecipientAddr,
			vmInput.GasLocked,
			vmInput.CallType,
			vmOutput)
	}

	addESDTEntryForTransferInVMOutput(
		vmInput, vmOutput,
		[]byte(core.BuiltInFunctionESDTTransfer),
		vmInput.RecipientAddr,
		[]*TopicTokenData{{
			tokenID,
			0,
			value,
		}})
	return vmOutput, nil
}

func addOutputTransferToVMOutput(
	index uint32,
	senderAddress []byte,
	function string,
	arguments [][]byte,
	recipient []byte,
	gasLocked uint64,
	callType vm.CallType,
	vmOutput *vmcommon.VMOutput,
) {
	encodedTxData := function
	for _, arg := range arguments {
		encodedTxData += "@" + hex.EncodeToString(arg)
	}
	outTransfer := vmcommon.OutputTransfer{
		Index:         index,
		Value:         big.NewInt(0),
		GasLimit:      vmOutput.GasRemaining,
		GasLocked:     gasLocked,
		Data:          []byte(encodedTxData),
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
	marshaller vmcommon.Marshalizer,
	globalSettingsHandler vmcommon.ESDTGlobalSettingsHandler,
	isReturnWithError bool,
) error {
	esdtData, err := getESDTDataFromKey(userAcnt, key, marshaller)
	if err != nil {
		return err
	}

	if esdtData.Type != uint32(core.Fungible) {
		return ErrOnlyFungibleTokensHaveBalanceTransfer
	}

	err = checkFrozeAndPause(userAcnt.AddressBytes(), key, esdtData, globalSettingsHandler, isReturnWithError)
	if err != nil {
		return err
	}

	esdtData.Value.Add(esdtData.Value, value)
	if esdtData.Value.Cmp(zero) < 0 {
		return ErrInsufficientFunds
	}

	return saveESDTData(userAcnt, esdtData, key, marshaller)
}

func checkFrozeAndPause(
	senderAddr []byte,
	key []byte,
	esdtData *esdt.ESDigitalToken,
	globalSettingsHandler vmcommon.ESDTGlobalSettingsHandler,
	isReturnWithError bool,
) error {
	if isReturnWithError {
		return nil
	}
	if bytes.Equal(senderAddr, core.ESDTSCAddress) {
		return nil
	}

	esdtUserMetaData := ESDTUserMetadataFromBytes(esdtData.Properties)
	if esdtUserMetaData.Frozen {
		return ErrESDTIsFrozenForAccount
	}

	if globalSettingsHandler.IsPaused(key) {
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
	marshaller vmcommon.Marshalizer,
) error {
	isValueZero := esdtData.Value.Cmp(zero) == 0
	if isValueZero && arePropertiesEmpty(esdtData.Properties) {
		return userAcnt.AccountDataHandler().SaveKeyValue(key, nil)
	}

	marshaledData, err := marshaller.Marshal(esdtData)
	if err != nil {
		return err
	}

	return userAcnt.AccountDataHandler().SaveKeyValue(key, marshaledData)
}

func getESDTDataFromKey(
	userAcnt vmcommon.UserAccountHandler,
	key []byte,
	marshaller vmcommon.Marshalizer,
) (*esdt.ESDigitalToken, error) {
	esdtData := &esdt.ESDigitalToken{Value: big.NewInt(0), Type: uint32(core.Fungible)}
	marshaledData, _, err := userAcnt.AccountDataHandler().RetrieveValue(key)
	if core.IsGetNodeFromDBError(err) {
		return nil, err
	}
	if err != nil || len(marshaledData) == 0 {
		return esdtData, nil
	}

	err = marshaller.Unmarshal(esdtData, marshaledData)
	if err != nil {
		return nil, err
	}

	return esdtData, nil
}

// will return nil if transfer is not limited
// if we are at sender shard, the sender or the destination must have the transfer role
// we cannot transfer a limited esdt to destination shard, as there we do not know if that token was transferred or not
// by an account with transfer account
func checkIfTransferCanHappenWithLimitedTransfer(
	tokenID []byte, esdtTokenKey []byte,
	senderAddress, destinationAddress []byte,
	globalSettingsHandler vmcommon.ExtendedESDTGlobalSettingsHandler,
	roleHandler vmcommon.ESDTRoleHandler,
	acntSnd, acntDst vmcommon.UserAccountHandler,
	isReturnWithError bool,
) error {
	if isReturnWithError {
		return nil
	}
	if check.IfNil(acntSnd) {
		return nil
	}
	if !globalSettingsHandler.IsLimitedTransfer(esdtTokenKey) {
		return nil
	}

	if globalSettingsHandler.IsSenderOrDestinationWithTransferRole(senderAddress, destinationAddress, tokenID) {
		return nil
	}

	errSender := roleHandler.CheckAllowedToExecute(acntSnd, tokenID, []byte(core.ESDTRoleTransfer))
	if errSender == nil {
		return nil
	}

	errDestination := roleHandler.CheckAllowedToExecute(acntDst, tokenID, []byte(core.ESDTRoleTransfer))
	return errDestination
}

// SetPayableChecker will set the payableCheck handler to the function
func (e *esdtTransfer) SetPayableChecker(payableHandler vmcommon.PayableChecker) error {
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

func noGasUseIfReturnCallAfterErrorWithFlag(enableEpochsHandler vmcommon.EnableEpochsHandler, vmInput *vmcommon.ContractCallInput) bool {
	if !enableEpochsHandler.IsFlagEnabled(EGLDInESDTMultiTransferFlag) {
		return false
	}
	return vmInput.ReturnCallAfterError
}
