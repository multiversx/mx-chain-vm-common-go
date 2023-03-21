package builtInFunctions

import (
	"bytes"
	"fmt"
	"math/big"
	"sync"

	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/marshal"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

// BaseAccountGuarderArgs is a struct placeholder for
// all necessary args to create a newBaseAccountGuarder
type BaseAccountGuarderArgs struct {
	GuardedAccountHandler vmcommon.GuardedAccountHandler
	Marshaller            marshal.Marshalizer
	EnableEpochsHandler   vmcommon.EnableEpochsHandler
	FuncGasCost           uint64
}

type baseAccountGuarder struct {
	baseActiveHandler
	marshaller            marshal.Marshalizer
	guardedAccountHandler vmcommon.GuardedAccountHandler

	mutExecution sync.RWMutex
	funcGasCost  uint64
}

func newBaseAccountGuarder(args BaseAccountGuarderArgs) (*baseAccountGuarder, error) {
	if check.IfNil(args.Marshaller) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(args.EnableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}
	if check.IfNil(args.GuardedAccountHandler) {
		return nil, ErrNilGuardedAccountHandler
	}

	accGuarder := &baseAccountGuarder{
		funcGasCost:           args.FuncGasCost,
		marshaller:            args.Marshaller,
		mutExecution:          sync.RWMutex{},
		guardedAccountHandler: args.GuardedAccountHandler,
	}

	accGuarder.activeHandler = args.EnableEpochsHandler.IsGuardAccountEnabled

	return accGuarder, nil
}

func (baf *baseAccountGuarder) checkBaseAccountGuarderArgs(
	senderAccount vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
	expectedNoOfArgs uint32,
) error {
	if check.IfNil(senderAccount) {
		return fmt.Errorf("%w for sender", ErrNilUserAccount)
	}
	if vmInput == nil {
		return ErrNilVmInput
	}

	senderIsNotCaller := !bytes.Equal(senderAccount.AddressBytes(), vmInput.CallerAddr)
	senderIsNotReceiver := !bytes.Equal(vmInput.CallerAddr, vmInput.RecipientAddr)
	if senderIsNotCaller || senderIsNotReceiver {
		return ErrOperationNotPermitted
	}
	if vmInput.CallValue == nil {
		return ErrNilValue
	}
	if !isZero(vmInput.CallValue) {
		return ErrBuiltInFunctionCalledWithValue
	}
	if len(vmInput.Arguments) != int(expectedNoOfArgs) {
		return fmt.Errorf("%w, expected %d, got %d ", ErrInvalidNumberOfArguments, expectedNoOfArgs, len(vmInput.Arguments))
	}
	if vmInput.GasProvided < baf.funcGasCost {
		return ErrNotEnoughGas
	}

	return nil
}

func isZero(n *big.Int) bool {
	return len(n.Bits()) == 0
}
