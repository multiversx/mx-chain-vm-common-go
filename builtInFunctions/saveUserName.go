package builtInFunctions

import (
	"encoding/hex"
	"math/big"
	"sync"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data/vm"
	"github.com/ElrondNetwork/elrond-vm-common"
)

type saveUserName struct {
	baseAlwaysActive
	gasCost         uint64
	mapDnsAddresses map[string]struct{}
	enableChange    bool
	mutExecution    sync.RWMutex
}

// NewSaveUserNameFunc returns a username built in function implementation
func NewSaveUserNameFunc(
	gasCost uint64,
	mapDnsAddresses map[string]struct{},
	enableChange bool,
) (*saveUserName, error) {
	if mapDnsAddresses == nil {
		return nil, ErrNilDnsAddresses
	}

	s := &saveUserName{
		gasCost:      gasCost,
		enableChange: enableChange,
	}
	s.mapDnsAddresses = make(map[string]struct{}, len(mapDnsAddresses))
	for key := range mapDnsAddresses {
		s.mapDnsAddresses[key] = struct{}{}
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

// ProcessBuiltinFunction sets the username to the account if it is allowed
func (s *saveUserName) ProcessBuiltinFunction(
	_, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	s.mutExecution.RLock()
	defer s.mutExecution.RUnlock()

	if vmInput == nil {
		return nil, ErrNilVmInput
	}
	if vmInput.CallValue.Cmp(zero) != 0 {
		return nil, ErrBuiltInFunctionCalledWithValue
	}
	if vmInput.GasProvided < s.gasCost {
		return nil, ErrNotEnoughGas
	}
	_, ok := s.mapDnsAddresses[string(vmInput.CallerAddr)]
	if !ok {
		return nil, ErrCallerIsNotTheDNSAddress
	}
	if len(vmInput.Arguments) != 1 {
		return nil, ErrInvalidArguments
	}

	if check.IfNil(acntDst) {
		// cross-shard call, in sender shard only the gas is taken out
		vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}
		vmOutput.OutputAccounts = make(map[string]*vmcommon.OutputAccount)
		setUserNameTxData := vmcommon.BuiltInFunctionSetUserName + "@" + hex.EncodeToString(vmInput.Arguments[0])
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

	currentUserName := acntDst.GetUserName()
	if !s.enableChange && len(currentUserName) > 0 {
		return nil, ErrUserNameChangeIsDisabled
	}

	acntDst.SetUserName(vmInput.Arguments[0])

	return &vmcommon.VMOutput{GasRemaining: vmInput.GasProvided - s.gasCost, ReturnCode: vmcommon.Ok}, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (s *saveUserName) IsInterfaceNil() bool {
	return s == nil
}
