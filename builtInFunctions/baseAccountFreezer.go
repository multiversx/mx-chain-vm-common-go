package builtInFunctions

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

var guardianKeyPrefix = []byte(core.ElrondProtectedKeyPrefix + GuardiansKeyIdentifier)

type BaseAccountFreezerArgs struct {
	BlockChainHook BlockChainEpochHook
	Marshaller     marshal.Marshalizer
	EpochNotifier  vmcommon.EpochNotifier
	FuncGasCost    uint64
}

type baseAccountFreezer struct {
	marshaller     marshal.Marshalizer
	blockchainHook BlockChainEpochHook

	mutExecution sync.RWMutex
	funcGasCost  uint64
}

func newBaseAccountFreezer(args BaseAccountFreezerArgs) (*baseAccountFreezer, error) {
	if check.IfNil(args.Marshaller) {
		return nil, ErrNilMarshaller
	}
	if check.IfNil(args.BlockChainHook) {
		return nil, ErrNilBlockChainHook
	}
	if check.IfNil(args.EpochNotifier) {
		return nil, ErrNilEpochNotifier
	}

	return &baseAccountFreezer{
		funcGasCost:    args.FuncGasCost,
		marshaller:     args.Marshaller,
		blockchainHook: args.BlockChainHook,
		mutExecution:   sync.RWMutex{},
	}, nil
}

func (baf *baseAccountFreezer) checkBaseArgs(
	senderAccount,
	receiverAccount vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
	expectedArgsNo uint32,
) error {
	if check.IfNil(senderAccount) {
		return fmt.Errorf("%w for sender", ErrNilUserAccount)
	}
	if check.IfNil(receiverAccount) {
		return fmt.Errorf("%w for receiver", ErrNilUserAccount)
	}
	if vmInput == nil {
		return ErrNilVmInput
	}
	if !(bytes.Equal(senderAccount.AddressBytes(), receiverAccount.AddressBytes()) &&
		bytes.Equal(senderAccount.AddressBytes(), vmInput.CallerAddr)) {
		return ErrOperationNotPermitted
	}
	if vmInput.CallValue == nil {
		return ErrNilValue
	}
	if !isZero(vmInput.CallValue) {
		return ErrBuiltInFunctionCalledWithValue
	}
	if len(vmInput.Arguments) != int(expectedArgsNo) {
		return fmt.Errorf("%w, expected %d, got %d ", ErrInvalidNumberOfArguments, expectedArgsNo, len(vmInput.Arguments))
	}
	if vmInput.GasProvided < baf.funcGasCost {
		return ErrNotEnoughGas
	}

	return nil
}

func (baf *baseAccountFreezer) guardians(account vmcommon.UserAccountHandler) (*Guardians, error) {
	marshalledData, err := account.AccountDataHandler().RetrieveValue(guardianKeyPrefix)
	if err != nil {
		return nil, err
	}

	// Fine, account has no guardian set
	if len(marshalledData) == 0 {
		return &Guardians{Data: make([]*Guardian, 0)}, nil
	}

	guardians := &Guardians{}
	err = baf.marshaller.Unmarshal(guardians, marshalledData)
	return guardians, err
}

func (baf *baseAccountFreezer) pending(guardian *Guardian) bool {
	return guardian.ActivationEpoch > baf.blockchainHook.CurrentEpoch()
}

func (baf *baseAccountFreezer) enabled(guardian *Guardian) bool {
	return !baf.pending(guardian)
}
