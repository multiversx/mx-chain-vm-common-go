package builtInFunctions

import (
	"bytes"
	"fmt"

	"github.com/multiversx/mx-chain-core-go/core"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

const noOfArgsSetGuardian = 2
const serviceUIDMaxLen = 32

// SetGuardianArgs is a struct placeholder for all necessary args
// to create a NewSetGuardianFunc
type SetGuardianArgs struct {
	BaseAccountGuarderArgs
}

type setGuardian struct {
	baseActiveHandler
	*baseAccountGuarder
}

// NewSetGuardianFunc will instantiate a new set guardian built-in function
func NewSetGuardianFunc(args SetGuardianArgs) (*setGuardian, error) {
	base, err := newBaseAccountGuarder(args.BaseAccountGuarderArgs)
	if err != nil {
		return nil, err
	}
	setGuardianFunc := &setGuardian{
		baseAccountGuarder: base,
	}
	setGuardianFunc.activeHandler = args.EnableEpochsHandler.IsSetGuardianEnabled

	return setGuardianFunc, nil
}

// ProcessBuiltinFunction will process the set guardian built-in function call
func (sg *setGuardian) ProcessBuiltinFunction(
	acntSnd, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	sg.mutExecution.RLock()
	defer sg.mutExecution.RUnlock()

	err := sg.checkBaseAccountGuarderArgs(acntSnd, vmInput, noOfArgsSetGuardian)
	if err != nil {
		return nil, err
	}
	err = sg.checkSetGuardianArgs(acntSnd, vmInput)
	if err != nil {
		return nil, err
	}

	newGuardian := vmInput.Arguments[0]
	guardianServiceUID := vmInput.Arguments[1]
	err = sg.guardedAccountHandler.SetGuardian(acntSnd, newGuardian, vmInput.TxGuardian, guardianServiceUID)
	if err != nil {
		return nil, err
	}

	return &vmcommon.VMOutput{ReturnCode: vmcommon.Ok, GasRemaining: vmInput.GasProvided - sg.funcGasCost}, nil
}

func (sg *setGuardian) checkSetGuardianArgs(
	sender vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) error {
	senderAddr := sender.AddressBytes()
	guardianAddr := vmInput.Arguments[0]
	guardianServiceUID := vmInput.Arguments[1]

	isGuardianAddrLenOk := len(vmInput.Arguments[0]) == len(senderAddr)
	isGuardianAddrSC := core.IsSmartContractAddress(guardianAddr)
	if !isGuardianAddrLenOk || isGuardianAddrSC {
		return fmt.Errorf("%w for guardian", ErrInvalidAddress)
	}

	if len(guardianServiceUID) > serviceUIDMaxLen {
		return fmt.Errorf("%w for guardian service", ErrInvalidServiceUID)
	}

	if bytes.Equal(senderAddr, guardianAddr) {
		return ErrCannotSetOwnAddressAsGuardian
	}

	return nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (sg *setGuardian) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	sg.mutExecution.Lock()
	sg.funcGasCost = gasCost.BuiltInCost.SetGuardian
	sg.mutExecution.Unlock()
}
