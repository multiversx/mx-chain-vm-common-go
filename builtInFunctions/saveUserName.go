package builtInFunctions

import (
	"encoding/hex"
	"math/big"
	"sync"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data/vm"
	"github.com/multiversx/mx-chain-vm-common-go"
)

type saveUserName struct {
	baseAlwaysActiveHandler
	gasCost           uint64
	isChangeEnabled   func() bool
	mapDnsAddresses   map[string]struct{}
	mapDnsV2Addresses map[string]struct{}
	mutExecution      sync.RWMutex
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
		gasCost:         gasCost,
		isChangeEnabled: enableEpochsHandler.IsChangeUsernameEnabled,
	}
	s.mapDnsAddresses = make(map[string]struct{}, len(mapDnsAddresses))
	for key := range mapDnsAddresses {
		s.mapDnsAddresses[key] = struct{}{}
	}

	s.mapDnsV2Addresses = make(map[string]struct{}, len(mapDnsV2Addresses))
	for key := range mapDnsAddresses {
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
	if vmInput.GasProvided < gasCost {
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
) (*vmcommon.VMOutput, error) {
	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}
	vmOutput.OutputAccounts = make(map[string]*vmcommon.OutputAccount)
	setUserNameTxData := builtInFuncName
	for _, arg := range vmInput.Arguments {
		setUserNameTxData += "@" + hex.EncodeToString(arg)
	}
	outTransfer := vmcommon.OutputTransfer{
		Value:         big.NewInt(0),
		GasLimit:      vmInput.GasProvided,
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
	_, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	s.mutExecution.RLock()
	defer s.mutExecution.RUnlock()

	addressesToCheck := s.mapDnsV2Addresses
	if !s.isChangeEnabled() {
		addressesToCheck = s.mapDnsAddresses
	}

	err := inputCheckForUserNameCall(vmInput, addressesToCheck, s.gasCost, 1)
	if err != nil {
		return nil, err
	}

	if check.IfNil(acntDst) {
		return createCrossShardUserNameCall(vmInput, core.BuiltInFunctionSetUserName)
	}

	currentUserName := acntDst.GetUserName()
	if !s.isChangeEnabled() && len(currentUserName) > 0 {
		return nil, ErrUserNameChangeIsDisabled
	}

	acntDst.SetUserName(vmInput.Arguments[0])

	return &vmcommon.VMOutput{GasRemaining: vmInput.GasProvided - s.gasCost, ReturnCode: vmcommon.Ok}, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (s *saveUserName) IsInterfaceNil() bool {
	return s == nil
}
