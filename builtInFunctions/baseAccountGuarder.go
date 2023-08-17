package builtInFunctions

import (
	"bytes"
	"fmt"
	"math/big"
	"sync"

	"github.com/multiversx/mx-chain-core-go/core"
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

	accGuarder.activeHandler = args.EnableEpochsHandler.IsFlagEnabledInCurrentEpoch
	accGuarder.flag = core.SetGuardianFlag

	return accGuarder, nil
}

func (baf *baseAccountGuarder) checkBaseAccountGuarderArgs(
	senderAddr []byte,
	receiverAddr []byte,
	value *big.Int,
	funcCallGasProvided uint64,
	arguments [][]byte,
	expectedNoOfArgs uint32,
) error {
	senderIsNotReceiver := !bytes.Equal(senderAddr, receiverAddr)
	if senderIsNotReceiver {
		return ErrOperationNotPermitted
	}
	if value == nil {
		return ErrNilValue
	}
	if !isZero(value) {
		return ErrBuiltInFunctionCalledWithValue
	}
	if len(arguments) != int(expectedNoOfArgs) {
		return fmt.Errorf("%w, expected %d, got %d ", ErrInvalidNumberOfArguments, expectedNoOfArgs, len(arguments))
	}
	if funcCallGasProvided < baf.funcGasCost {
		return ErrNotEnoughGas
	}

	return nil
}

func isZero(n *big.Int) bool {
	return len(n.Bits()) == 0
}
