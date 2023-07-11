package builtInFunctions

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
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
	setGuardianFunc.activeHandler = args.EnableEpochsHandler.IsSetGuardianEnabledInEpoch
	setGuardianFunc.currentEpochHandler = args.EnableEpochsHandler.GetCurrentEpoch

	return setGuardianFunc, nil
}

// ProcessBuiltinFunction will process the set guardian built-in function call
func (sg *setGuardian) ProcessBuiltinFunction(
	acntSnd, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	if check.IfNil(acntSnd) {
		return nil, fmt.Errorf("%w for sender", ErrNilUserAccount)
	}
	if vmInput == nil {
		return nil, ErrNilVmInput
	}
	if len(vmInput.Arguments) != noOfArgsSetGuardian {
		return nil, fmt.Errorf("%w, expected %d, got %d ", ErrInvalidNumberOfArguments, noOfArgsSetGuardian, len(vmInput.Arguments))
	}

	senderAddr := acntSnd.AddressBytes()
	senderIsNotCaller := !bytes.Equal(senderAddr, vmInput.CallerAddr)
	if senderIsNotCaller {
		return nil, ErrOperationNotPermitted
	}
	sg.mutExecution.RLock()
	defer sg.mutExecution.RUnlock()

	newGuardian := vmInput.Arguments[0]
	guardianServiceUID := vmInput.Arguments[1]
	gasProvidedForCall := vmInput.GasProvided

	err := sg.CheckIsExecutable(
		senderAddr,
		vmInput.CallValue,
		vmInput.RecipientAddr,
		gasProvidedForCall,
		vmInput.Arguments,
	)
	if err != nil {
		return nil, err
	}

	err = sg.guardedAccountHandler.SetGuardian(acntSnd, newGuardian, vmInput.TxGuardian, guardianServiceUID)
	if err != nil {
		return nil, err
	}

	entry := &vmcommon.LogEntry{
		Address:    acntSnd.AddressBytes(),
		Identifier: []byte(core.BuiltInFunctionSetGuardian),
		Topics:     [][]byte{newGuardian, guardianServiceUID},
	}

	return &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: vmInput.GasProvided - sg.funcGasCost,
		Logs:         []*vmcommon.LogEntry{entry},
	}, nil
}

// CheckIsExecutable will check if the set guardian built-in function can be executed
func (sg *setGuardian) CheckIsExecutable(
	senderAddr []byte,
	value *big.Int,
	receiverAddr []byte,
	gasProvidedForCall uint64,
	arguments [][]byte,
) error {

	err := sg.checkBaseAccountGuarderArgs(
		senderAddr,
		receiverAddr,
		value,
		gasProvidedForCall,
		arguments,
		noOfArgsSetGuardian,
	)
	if err != nil {
		return err
	}

	return sg.checkSetGuardianArgs(senderAddr, arguments)
}

func (sg *setGuardian) checkSetGuardianArgs(
	senderAddr []byte,
	arguments [][]byte,
) error {

	guardianAddr := arguments[0]
	guardianServiceUID := arguments[1]

	isGuardianAddrLenOk := len(arguments[0]) == len(senderAddr)
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
