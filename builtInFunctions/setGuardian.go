package builtInFunctions

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/atomic"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	logger "github.com/ElrondNetwork/elrond-go-logger"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

var logAccountFreezer = logger.GetOrCreate("systemSmartContracts/setGuardian")

// TODO: Use these values from elrond-go-core once a release tag is ready

const (
	GuardiansKeyIdentifier     = "guardians"
	BuiltInFunctionSetGuardian = "SetGuardian"
)

type Guardian struct {
	Address         []byte
	ActivationEpoch uint32
}

type Guardians struct {
	Data []*Guardian
}

// SetGuardianArgs is a struct placeholder for all necessary args
// to create a NewSetGuardianFunc
type SetGuardianArgs struct {
	Marshaller    marshal.Marshalizer
	EpochNotifier vmcommon.EpochNotifier

	FuncGasCost              uint64
	GuardianActivationEpochs uint32
	SetGuardianEnableEpoch   uint32
}

type setGuardian struct {
	*baseEnabled
	marshaller marshal.Marshalizer

	mutExecution             sync.RWMutex
	currentEpoch             uint32
	guardianActivationEpochs uint32
	funcGasCost              uint64
	keyPrefix                []byte
}

// NewSetGuardianFunc will instantiate a new set guardian built-in function
func NewSetGuardianFunc(args SetGuardianArgs) (*setGuardian, error) {
	if check.IfNil(args.Marshaller) {
		return nil, ErrNilMarshaller
	}
	if check.IfNil(args.EpochNotifier) {
		return nil, ErrNilEpochNotifier
	}

	setGuardianFunc := &setGuardian{
		funcGasCost:              args.FuncGasCost,
		marshaller:               args.Marshaller,
		guardianActivationEpochs: args.GuardianActivationEpochs,
		mutExecution:             sync.RWMutex{},
		keyPrefix:                []byte(core.ElrondProtectedKeyPrefix + GuardiansKeyIdentifier),
	}
	setGuardianFunc.baseEnabled = &baseEnabled{
		function:        BuiltInFunctionSetGuardian,
		activationEpoch: args.SetGuardianEnableEpoch,
		flagActivated:   atomic.Flag{},
	}

	logAccountFreezer.Debug("set guardian enable epoch", args.SetGuardianEnableEpoch)
	args.EpochNotifier.RegisterNotifyHandler(setGuardianFunc)

	return setGuardianFunc, nil
}

// ProcessBuiltinFunction will process the set guardian built-in function call
// Currently, the following cases are treated:
// Case 1. User does NOT have any guardian => set guardian
// Case 2. User has ONE guardian pending => do not set new guardian, wait until first one is set
// Case 3. User has ONE guardian enabled => set guardian
// Case 4. User has TWO guardians. FIRST is enabled, SECOND is pending => does not set, wait until second one is set
// Case 5. User has TWO guardians. FIRST is enabled, SECOND is enabled => replace oldest one + set new one as pending
func (sg *setGuardian) ProcessBuiltinFunction(
	acntSnd, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	sg.mutExecution.RLock()
	defer sg.mutExecution.RUnlock()

	err := sg.checkArguments(acntSnd, acntDst, vmInput)
	if err != nil {
		return nil, err
	}

	guardians, err := sg.guardians(acntSnd)
	if err != nil {
		return nil, err
	}
	if sg.contains(guardians, vmInput.Arguments[0]) {
		return nil, ErrGuardianAlreadyExists
	}

	switch len(guardians.Data) {
	case 0:
		return sg.tryAddGuardian(acntSnd, vmInput.Arguments[0], guardians, vmInput.GasProvided) // Case 1
	case 1:
		if sg.pending(guardians.Data[0]) { // Case 2
			return nil, fmt.Errorf("%w: %s", ErrOwnerAlreadyHasOneGuardianPending, hex.EncodeToString(guardians.Data[0].Address))
		}
		return sg.tryAddGuardian(acntSnd, vmInput.Arguments[0], guardians, vmInput.GasProvided) // Case 3
	case 2:
		if sg.pending(guardians.Data[1]) { // Case 4
			return nil, fmt.Errorf("%w: %s", ErrOwnerAlreadyHasOneGuardianPending, hex.EncodeToString(guardians.Data[1].Address))
		}
		guardians.Data = guardians.Data[1:]                                                     // remove oldest guardian
		return sg.tryAddGuardian(acntSnd, vmInput.Arguments[0], guardians, vmInput.GasProvided) // Case 5
	default:
		return &vmcommon.VMOutput{ReturnCode: vmcommon.ExecutionFailed}, nil
	}
}

func (sg *setGuardian) checkArguments(
	senderAccount,
	receiverAccount vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
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

	senderIsCaller := bytes.Equal(senderAccount.AddressBytes(), vmInput.CallerAddr)
	senderIsReceiver := bytes.Equal(senderAccount.AddressBytes(), receiverAccount.AddressBytes())
	if !(senderIsReceiver && senderIsCaller) {
		return ErrOperationNotPermitted
	}
	if vmInput.CallValue == nil {
		return ErrNilValue
	}
	if !isZero(vmInput.CallValue) {
		return ErrBuiltInFunctionCalledWithValue
	}
	if len(vmInput.Arguments) != 1 {
		return fmt.Errorf("%w, expected 1, got %d ", ErrInvalidNumberOfArguments, len(vmInput.Arguments))
	}
	if len(vmInput.Arguments[0]) != len(senderAccount.AddressBytes()) {
		return fmt.Errorf("%w for guardian", ErrInvalidAddress)
	}
	if bytes.Equal(vmInput.CallerAddr, vmInput.Arguments[0]) {
		return ErrCannotSetOwnAddressAsGuardian
	}
	if vmInput.GasProvided < sg.funcGasCost {
		return ErrNotEnoughGas
	}

	return nil
}

func isZero(n *big.Int) bool {
	return len(n.Bits()) == 0
}

func (sg *setGuardian) guardians(account vmcommon.UserAccountHandler) (*Guardians, error) {
	marshalledData, err := account.AccountDataHandler().RetrieveValue(sg.keyPrefix)
	if err != nil {
		return nil, err
	}

	// Account has no guardian set
	if len(marshalledData) == 0 {
		return &Guardians{Data: make([]*Guardian, 0)}, nil
	}

	guardians := &Guardians{}
	err = sg.marshaller.Unmarshal(guardians, marshalledData)
	if err != nil {
		return nil, err
	}

	return guardians, err
}

func (sg *setGuardian) contains(guardians *Guardians, guardianAddress []byte) bool {
	for _, guardian := range guardians.Data {
		if bytes.Equal(guardian.Address, guardianAddress) {
			return true
		}
	}
	return false
}

func (sg *setGuardian) tryAddGuardian(
	account vmcommon.UserAccountHandler,
	guardianAddress []byte,
	guardians *Guardians,
	gasProvided uint64,
) (*vmcommon.VMOutput, error) {
	err := sg.addGuardian(account, guardianAddress, guardians)
	if err != nil {
		return nil, err
	}
	return &vmcommon.VMOutput{ReturnCode: vmcommon.Ok, GasRemaining: gasProvided - sg.funcGasCost}, nil
}

func (sg *setGuardian) addGuardian(account vmcommon.UserAccountHandler, guardianAddress []byte, guardians *Guardians) error {
	guardian := &Guardian{
		Address:         guardianAddress,
		ActivationEpoch: sg.currentEpoch + sg.guardianActivationEpochs,
	}

	guardians.Data = append(guardians.Data, guardian)
	marshalledData, err := sg.marshaller.Marshal(guardians)
	if err != nil {
		return err
	}

	return account.AccountDataHandler().SaveKeyValue(sg.keyPrefix, marshalledData)
}

func (sg *setGuardian) pending(guardian *Guardian) bool {
	return guardian.ActivationEpoch > sg.currentEpoch
}

// SetNewGasConfig is called whenever gas cost is changed
func (sg *setGuardian) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	sg.mutExecution.Lock()
	sg.funcGasCost = gasCost.BuiltInCost.SetGuardian
	sg.mutExecution.Unlock()
}

func (sg *setGuardian) EpochConfirmed(epoch uint32, _ uint64) {
	sg.mutExecution.Lock()
	defer sg.mutExecution.Unlock()

	sg.currentEpoch = epoch
	sg.baseEnabled.EpochConfirmed(epoch, 0)
}
