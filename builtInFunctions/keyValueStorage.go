package builtInFunctions

import (
	"bytes"
	"fmt"
	"math/big"
	"sync"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-vm-common"
)

type saveKeyValueStorage struct {
	baseAlwaysActive
	gasConfig    vmcommon.BaseOperationCost
	funcGasCost  uint64
	mutExecution sync.RWMutex
}

// NewSaveKeyValueStorageFunc returns the save key-value storage built in function
func NewSaveKeyValueStorageFunc(
	gasConfig vmcommon.BaseOperationCost,
	funcGasCost uint64,
) (*saveKeyValueStorage, error) {
	s := &saveKeyValueStorage{
		gasConfig:   gasConfig,
		funcGasCost: funcGasCost,
	}

	return s, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (k *saveKeyValueStorage) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	k.mutExecution.Lock()
	k.funcGasCost = gasCost.BuiltInCost.SaveKeyValue
	k.gasConfig = gasCost.BaseOperationCost
	k.mutExecution.Unlock()
}

// ProcessBuiltinFunction will save the value for the selected key
func (k *saveKeyValueStorage) ProcessBuiltinFunction(
	acntSnd, _ vmcommon.UserAccountHandler,
	input *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	k.mutExecution.RLock()
	defer k.mutExecution.RUnlock()

	err := checkArgumentsForSaveKeyValue(acntSnd, input)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{
		GasRemaining: input.GasProvided,
		GasRefund:    big.NewInt(0),
	}

	useGas := k.funcGasCost
	for i := 0; i < len(input.Arguments); i += 2 {
		key := input.Arguments[i]
		value := input.Arguments[i+1]
		length := uint64(len(value) + len(key))
		useGas += length * k.gasConfig.PersistPerByte

		if !vmcommon.IsAllowedToSaveUnderKey(key) {
			return nil, fmt.Errorf("%w it is not allowed to save under key %s", ErrOperationNotPermitted, key)
		}

		oldValue, _ := acntSnd.AccountDataHandler().RetrieveValue(key)
		if bytes.Equal(oldValue, value) {
			continue
		}

		lengthChange := uint64(0)
		lengthOldValue := uint64(len(oldValue))
		lengthNewValue := uint64(len(value))
		if lengthOldValue < lengthNewValue {
			lengthChange = lengthNewValue - lengthOldValue
		}

		useGas += k.gasConfig.StorePerByte * lengthChange
		if input.GasProvided < useGas {
			return nil, ErrNotEnoughGas
		}

		err = acntSnd.AccountDataHandler().SaveKeyValue(key, value)
		if err != nil {
			return nil, err
		}
	}

	vmOutput.GasRemaining -= useGas

	return vmOutput, nil
}

func checkArgumentsForSaveKeyValue(acntDst vmcommon.UserAccountHandler, input *vmcommon.ContractCallInput) error {
	if input == nil {
		return ErrNilVmInput
	}
	if len(input.Arguments) < 2 {
		return ErrInvalidArguments
	}
	if len(input.Arguments)%2 != 0 {
		return ErrInvalidArguments
	}
	if input.CallValue.Cmp(zero) != 0 {
		return ErrBuiltInFunctionCalledWithValue
	}
	if check.IfNil(acntDst) {
		return ErrNilSCDestAccount
	}
	if !bytes.Equal(input.CallerAddr, input.RecipientAddr) {
		return fmt.Errorf("%w not the owner of the account", ErrOperationNotPermitted)
	}
	if vmcommon.IsSmartContractAddress(input.CallerAddr) {
		return fmt.Errorf("%w key-value builtin function not allowed for smart contracts", ErrOperationNotPermitted)
	}

	return nil
}

// IsInterfaceNil return true if underlying object in nil
func (k *saveKeyValueStorage) IsInterfaceNil() bool {
	return k == nil
}
