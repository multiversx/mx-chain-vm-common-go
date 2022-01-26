package builtInFunctions

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/ElrondNetwork/elrond-go-core/core/atomic"
	logger "github.com/ElrondNetwork/elrond-go-logger"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

var logAccountFreezer = logger.GetOrCreate("systemSmartContracts/setGuardian")

// TODO: Use these values from elrond-go-core once a release tag is ready

const (
	GuardiansKeyIdentifier     = "guardians"
	BuiltInFunctionSetGuardian = "SetGuardian"
)

const noOfArgsSetGuardian = 1

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
	BaseAccountFreezerArgs

	GuardianActivationEpochs uint32
	SetGuardianEnableEpoch   uint32
}

type setGuardian struct {
	*baseEnabled
	*baseAccountFreezer

	guardianActivationEpochs uint32
}

// NewSetGuardianFunc will instantiate a new set guardian built-in function
func NewSetGuardianFunc(args SetGuardianArgs) (*setGuardian, error) {
	base, err := newBaseAccountFreezer(args.BaseAccountFreezerArgs)
	if err != nil {
		return nil, err
	}
	setGuardianFunc := &setGuardian{
		guardianActivationEpochs: args.GuardianActivationEpochs,
	}
	setGuardianFunc.baseEnabled = &baseEnabled{
		function:        BuiltInFunctionSetGuardian,
		activationEpoch: args.SetGuardianEnableEpoch,
		flagActivated:   atomic.Flag{},
	}
	setGuardianFunc.baseAccountFreezer = base

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

	err := sg.checkArgs(acntSnd, acntDst, vmInput, noOfArgsSetGuardian)
	if err != nil {
		return nil, err
	}
	err = sg.checkSetGuardianArgs(acntSnd, vmInput)
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
		// Case 1
		return sg.tryAddGuardian(acntSnd, vmInput.Arguments[0], guardians, vmInput.GasProvided)
	case 1:
		// Case 2
		if sg.pending(guardians.Data[0]) {
			return nil, fmt.Errorf("%w: %s", ErrOwnerAlreadyHasOneGuardianPending, hex.EncodeToString(guardians.Data[0].Address))
		}
		// Case 3
		return sg.tryAddGuardian(acntSnd, vmInput.Arguments[0], guardians, vmInput.GasProvided)
	case 2:
		// Case 4
		if sg.pending(guardians.Data[1]) {
			return nil, fmt.Errorf("%w: %s", ErrOwnerAlreadyHasOneGuardianPending, hex.EncodeToString(guardians.Data[1].Address))
		}
		// Case 5
		guardians.Data = guardians.Data[1:] // remove oldest guardian
		return sg.tryAddGuardian(acntSnd, vmInput.Arguments[0], guardians, vmInput.GasProvided)
	default:
		return &vmcommon.VMOutput{ReturnCode: vmcommon.ExecutionFailed}, nil
	}
}

func (sg *setGuardian) checkSetGuardianArgs(
	sender vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) error {
	senderAddr := sender.AddressBytes()
	guardianAddr := vmInput.Arguments[0]

	if len(senderAddr) != len(guardianAddr) {
		return fmt.Errorf("%w for guardian", ErrInvalidAddress)
	}
	if bytes.Equal(senderAddr, guardianAddr) {
		return ErrCannotOwnAddressAsGuardian
	}

	return nil
}

func (sg *setGuardian) guardians(account vmcommon.UserAccountHandler) (*Guardians, error) {
	marshalledData, err := account.AccountDataHandler().RetrieveValue(guardianKeyPrefix)
	if err != nil {
		return nil, err
	}

	// Fine, account has no guardian set
	if len(marshalledData) == 0 {
		return &Guardians{Data: make([]*Guardian, 0)}, nil
	}

	guardians := &Guardians{}
	err = sg.marshaller.Unmarshal(guardians, marshalledData)
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

	return account.AccountDataHandler().SaveKeyValue(guardianKeyPrefix, marshalledData)
}

// SetNewGasConfig is called whenever gas cost is changed
func (sg *setGuardian) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	sg.mutExecution.Lock()
	sg.funcGasCost = gasCost.BuiltInCost.SetGuardian
	sg.mutExecution.Unlock()
}

func (sg *setGuardian) EpochConfirmed(epoch uint32, _ uint64) {
	sg.baseEnabled.EpochConfirmed(epoch, 0)
	sg.baseAccountFreezer.EpochConfirmed(epoch, 0)
}
