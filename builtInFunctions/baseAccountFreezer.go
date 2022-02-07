package builtInFunctions

import (
	"bytes"
	"fmt"
	"math/big"
	"sync"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	guardiansData "github.com/ElrondNetwork/elrond-go-core/data/guardians"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

var guardianKey = []byte(core.ElrondProtectedKeyPrefix + core.GuardiansKeyIdentifier)

// BaseAccountFreezerArgs is a struct placeholder for
// all necessary args to create a newBaseAccountFreezer
type BaseAccountFreezerArgs struct {
	Marshaller    marshal.Marshalizer
	EpochNotifier vmcommon.EpochNotifier
	FuncGasCost   uint64
}

type baseAccountFreezer struct {
	marshaller marshal.Marshalizer

	mutExecution sync.RWMutex
	funcGasCost  uint64
	currentEpoch uint32
}

func newBaseAccountFreezer(args BaseAccountFreezerArgs) (*baseAccountFreezer, error) {
	if check.IfNil(args.Marshaller) {
		return nil, ErrNilMarshaller
	}
	if check.IfNil(args.EpochNotifier) {
		return nil, ErrNilEpochNotifier
	}

	return &baseAccountFreezer{
		funcGasCost:  args.FuncGasCost,
		marshaller:   args.Marshaller,
		mutExecution: sync.RWMutex{},
	}, nil
}

func (baf *baseAccountFreezer) checkBaseAccountFreezerArgs(
	senderAccount,
	receiverAccount vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
	expectedNoOfArgs uint32,
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

	senderIsNotCaller := !bytes.Equal(senderAccount.AddressBytes(), vmInput.CallerAddr)
	senderIsNotReceiver := !bytes.Equal(senderAccount.AddressBytes(), receiverAccount.AddressBytes())
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

func (baf *baseAccountFreezer) enabledGuardian(account vmcommon.UserAccountHandler) (*guardiansData.Guardian, error) {
	guardians, err := baf.guardians(account)
	if err != nil {
		return nil, err
	}

	enabledGuardian := &guardiansData.Guardian{}
	latestActivationEpoch := uint32(0)
	for _, guardian := range guardians.Data {
		if baf.enabled(guardian) && guardian.ActivationEpoch > latestActivationEpoch {
			enabledGuardian = guardian
			latestActivationEpoch = guardian.ActivationEpoch
		}
	}

	if latestActivationEpoch == 0 {
		return nil, ErrNoGuardianEnabled
	}
	return enabledGuardian, nil // TODO: Check this guardian against relayer address
}

func (baf *baseAccountFreezer) guardians(account vmcommon.UserAccountHandler) (*guardiansData.Guardians, error) {
	accountHandler := account.AccountDataHandler()
	if check.IfNil(accountHandler) {
		return nil, ErrNilAccountHandler
	}

	marshalledData, err := accountHandler.RetrieveValue(guardianKey)
	if err != nil {
		return nil, err
	}

	// Account has no guardian set
	if len(marshalledData) == 0 {
		return &guardiansData.Guardians{Data: make([]*guardiansData.Guardian, 0)}, nil
	}

	guardians := &guardiansData.Guardians{}
	err = baf.marshaller.Unmarshal(guardians, marshalledData)
	if err != nil {
		return nil, err
	}

	return guardians, err
}

func (baf *baseAccountFreezer) pending(guardian *guardiansData.Guardian) bool {
	return guardian.ActivationEpoch > baf.currentEpoch
}

func (baf *baseAccountFreezer) enabled(guardian *guardiansData.Guardian) bool {
	return !baf.pending(guardian)
}

func (baf *baseAccountFreezer) EpochConfirmed(epoch uint32, _ uint64) {
	baf.mutExecution.Lock()
	baf.currentEpoch = epoch
	baf.mutExecution.Unlock()
}
