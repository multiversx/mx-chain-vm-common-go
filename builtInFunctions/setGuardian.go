package builtInFunctions

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/atomic"
	guardiansData "github.com/ElrondNetwork/elrond-go-core/data/guardians"
	logger "github.com/ElrondNetwork/elrond-go-logger"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

var logAccountFreezer = logger.GetOrCreate("systemSmartContracts/setGuardian")

const noOfArgsSetGuardian = 1

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
		function:        core.BuiltInFunctionSetGuardian,
		activationEpoch: args.SetGuardianEnableEpoch,
		flagActivated:   atomic.Flag{},
	}
	setGuardianFunc.baseAccountFreezer = base

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

	err := sg.checkBaseAccountFreezerArgs(acntSnd, acntDst, vmInput, noOfArgsSetGuardian)
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

func (sg *setGuardian) checkSetGuardianArgs(
	sender vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) error {
	senderAddr := sender.AddressBytes()
	guardianAddr := vmInput.Arguments[0]

	isGuardianAddrLenOk := len(vmInput.Arguments[0]) == len(senderAddr)
	isGuardianAddrSC := core.IsSmartContractAddress(guardianAddr)
	if !isGuardianAddrLenOk || isGuardianAddrSC {
		return fmt.Errorf("%w for guardian", ErrInvalidAddress)
	}

	if bytes.Equal(senderAddr, guardianAddr) {
		return ErrCannotSetOwnAddressAsGuardian
	}

	return nil
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

	return account.AccountDataHandler().SaveKeyValue(guardianKey, marshalledData)
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
