package builtInFunctions

import (
	"encoding/hex"
	"math/big"
	"sync"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data/vm"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

type saveUserName struct {
	baseAlwaysActiveHandler
	gasCost             uint64
	enableEpochsHandler vmcommon.EnableEpochsHandler
	mapDnsAddresses     map[string]struct{}
	mapDnsV2Addresses   map[string]struct{}
	mutExecution        sync.RWMutex
}

// NewSaveUserNameFunc returns a username built in function implementation
func NewSaveUserNameFunc(
	gasCost uint64,
	mapDnsAddresses map[string]struct{},
	mapDnsV2Addresses map[string]struct{},
	enableEpochsHandler vmcommon.EnableEpochsHandler,
) (*saveUserName, error) {
	if mapDnsAddresses == nil || mapDnsV2Addresses == nil {
		return nil, ErrNilDnsAddresses
	}
	if check.IfNil(enableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}

	s := &saveUserName{
		gasCost:             gasCost,
		enableEpochsHandler: enableEpochsHandler,
	}
	s.mapDnsAddresses = make(map[string]struct{}, len(mapDnsAddresses))
	for key := range mapDnsAddresses {
		s.mapDnsAddresses[key] = struct{}{}
	}

	s.mapDnsV2Addresses = make(map[string]struct{}, len(mapDnsV2Addresses))
	for key := range mapDnsV2Addresses {
		s.mapDnsV2Addresses[key] = struct{}{}
	}

	return s, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (s *saveUserName) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	s.mutExecution.Lock()
	s.gasCost = gasCost.BuiltInCost.SaveUserName
	s.mutExecution.Unlock()
}

func inputCheckForUserNameCall(
	acntSnd vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
	mapDnsAddresses map[string]struct{},
	gasCost uint64,
	numArgs int,
) error {
	if vmInput == nil {
		return ErrNilVmInput
	}
	if vmInput.CallValue.Cmp(zero) != 0 {
		return ErrBuiltInFunctionCalledWithValue
	}
	if !check.IfNil(acntSnd) && vmInput.GasProvided < gasCost {
		return ErrNotEnoughGas
	}
	_, ok := mapDnsAddresses[string(vmInput.CallerAddr)]
	if !ok {
		return ErrCallerIsNotTheDNSAddress
	}
	if len(vmInput.Arguments) != numArgs {
		return ErrInvalidArguments
	}
	return nil
}

func createCrossShardUserNameCall(
	vmInput *vmcommon.ContractCallInput,
	builtInFuncName string,
	gasLimit uint64,
) (*vmcommon.VMOutput, error) {
	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}
	vmOutput.OutputAccounts = make(map[string]*vmcommon.OutputAccount)
	setUserNameTxData := builtInFuncName
	for _, arg := range vmInput.Arguments {
		setUserNameTxData += "@" + hex.EncodeToString(arg)
	}
	outTransfer := vmcommon.OutputTransfer{
		Index:         1,
		Value:         big.NewInt(0),
		GasLimit:      gasLimit,
		GasLocked:     vmInput.GasLocked,
		Data:          []byte(setUserNameTxData),
		CallType:      vm.AsynchronousCall,
		SenderAddress: vmInput.CallerAddr,
	}

	vmOutput.OutputAccounts[string(vmInput.RecipientAddr)] = &vmcommon.OutputAccount{
		Address:         vmInput.RecipientAddr,
		OutputTransfers: []vmcommon.OutputTransfer{outTransfer},
	}
	return vmOutput, nil
}

// ProcessBuiltinFunction sets the username to the account if it is allowed
func (s *saveUserName) ProcessBuiltinFunction(
	acntSnd, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	s.mutExecution.RLock()
	defer s.mutExecution.RUnlock()

	addressesToCheck := s.mapDnsV2Addresses
	if !s.enableEpochsHandler.IsFlagEnabled(ChangeUsernameFlag) {
		addressesToCheck = s.mapDnsAddresses
	}

	err := inputCheckForUserNameCall(acntSnd, vmInput, addressesToCheck, s.gasCost, 1)
	if err != nil {
		return nil, err
	}

	if check.IfNil(acntDst) {
		gasLimit := vmInput.GasProvided
		if s.enableEpochsHandler.IsFlagEnabled(ChangeUsernameFlag) {
			gasLimit = vmInput.GasProvided - s.gasCost
		}

		return createCrossShardUserNameCall(vmInput, core.BuiltInFunctionSetUserName, gasLimit)
	}

	currentUserName := acntDst.GetUserName()
	if !s.enableEpochsHandler.IsFlagEnabled(ChangeUsernameFlag) && len(currentUserName) > 0 {
		return nil, ErrUserNameChangeIsDisabled
	}

	acntDst.SetUserName(vmInput.Arguments[0])

	gasRemaining := vmInput.GasProvided - s.gasCost
	if s.enableEpochsHandler.IsFlagEnabled(ChangeUsernameFlag) && check.IfNil(acntSnd) {
		gasRemaining = vmInput.GasProvided
	}

	vmOutput := &vmcommon.VMOutput{
		GasRemaining: gasRemaining,
		ReturnCode:   vmcommon.Ok,
	}
	addLogEntryForUserNameChange(vmInput, vmOutput, currentUserName)

	return vmOutput, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (s *saveUserName) IsInterfaceNil() bool {
	return s == nil
}
