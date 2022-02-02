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
	guardiansData "github.com/ElrondNetwork/elrond-go-core/data/guardians"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	logger "github.com/ElrondNetwork/elrond-go-logger"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

var logAccountFreezer = logger.GetOrCreate("systemSmartContracts/setGuardian")

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
		keyPrefix:                []byte(core.ElrondProtectedKeyPrefix + core.GuardiansKeyIdentifier),
	}
	setGuardianFunc.baseEnabled = &baseEnabled{
		function:        core.BuiltInFunctionSetGuardian,
		activationEpoch: args.SetGuardianEnableEpoch,
		flagActivated:   atomic.Flag{},
	}

	logAccountFreezer.Debug("set guardian enable epoch", args.SetGuardianEnableEpoch)
	args.EpochNotifier.RegisterNotifyHandler(setGuardianFunc)

	return setGuardianFunc, nil
}

// ProcessBuiltinFunction will process the set guardian built-in function call
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
		// User does NOT have any guardian => set guardian
		return sg.tryAddGuardian(acntSnd, vmInput.Arguments[0], guardians, vmInput.GasProvided)
	case 1:
		// User has ONE guardian pending => do not set new guardian, wait until FIRST one is set
		if sg.pending(guardians.Data[0]) {
			return nil, fmt.Errorf("%w: %s", ErrOwnerAlreadyHasOneGuardianPending, hex.EncodeToString(guardians.Data[0].Address))
		}
		// User has ONE guardian enabled => set guardian
		return sg.tryAddGuardian(acntSnd, vmInput.Arguments[0], guardians, vmInput.GasProvided)
	case 2:
		// User has TWO guardians. FIRST is enabled, SECOND is pending => do not set new guardian, wait until SECOND one is set
		if sg.pending(guardians.Data[1]) {
			return nil, fmt.Errorf("%w: %s", ErrOwnerAlreadyHasOneGuardianPending, hex.EncodeToString(guardians.Data[1].Address))
		}
		// User has TWO guardians. FIRST is enabled, SECOND is enabled => replace oldest one + set new one as pending
		guardians.Data = guardians.Data[1:] // remove oldest guardian
		return sg.tryAddGuardian(acntSnd, vmInput.Arguments[0], guardians, vmInput.GasProvided)
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
	if len(vmInput.Arguments) != 1 {
		return fmt.Errorf("%w, expected 1, got %d ", ErrInvalidNumberOfArguments, len(vmInput.Arguments))
	}

	isGuardianAddrLenOk := len(vmInput.Arguments[0]) == len(senderAccount.AddressBytes())
	isGuardianAddrSC := core.IsSmartContractAddress(senderAccount.AddressBytes())
	if !isGuardianAddrLenOk || isGuardianAddrSC {
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

func (sg *setGuardian) guardians(account vmcommon.UserAccountHandler) (*guardiansData.Guardians, error) {

	marshalledData, err := account.AccountDataHandler().RetrieveValue(sg.keyPrefix)
	if err != nil {
		return nil, err
	}

	// Account has no guardian set
	if len(marshalledData) == 0 {
		return &guardiansData.Guardians{Data: make([]*guardiansData.Guardian, 0)}, nil
	}

	guardians := &guardiansData.Guardians{}
	err = sg.marshaller.Unmarshal(guardians, marshalledData)
	if err != nil {
		return nil, err
	}

	return guardians, err
}

func (sg *setGuardian) contains(guardians *guardiansData.Guardians, guardianAddress []byte) bool {
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
	guardians *guardiansData.Guardians,
	gasProvided uint64,
) (*vmcommon.VMOutput, error) {
	err := sg.addGuardian(account, guardianAddress, guardians)
	if err != nil {
		return nil, err
	}
	return &vmcommon.VMOutput{ReturnCode: vmcommon.Ok, GasRemaining: gasProvided - sg.funcGasCost}, nil
}

func (sg *setGuardian) addGuardian(account vmcommon.UserAccountHandler, guardianAddress []byte, guardians *guardiansData.Guardians) error {
	guardian := &guardiansData.Guardian{
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

func (sg *setGuardian) pending(guardian *guardiansData.Guardian) bool {
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
