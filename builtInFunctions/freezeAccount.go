package builtInFunctions

import (
	"fmt"

	"github.com/ElrondNetwork/elrond-go-core/core/atomic"
	logger "github.com/ElrondNetwork/elrond-go-logger"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

const (
	BuiltInFunctionFreezeAccount   = "freezeAccount"
	BuiltInFunctionUnfreezeAccount = "unfreezeAccount"
)

const noOfArgsFreezeAccount = 0

var logFreezeAccount = logger.GetOrCreate("systemSmartContracts/freezeAccount")

// FreezeAccountArgs is a struct placeholder for all
// necessary args to create a NewFreezeAccountFunc
type FreezeAccountArgs struct {
	BaseAccountFreezerArgs

	Freeze                   bool
	FreezeAccountEnableEpoch uint32
}

type freezeAccount struct {
	*baseEnabled
	*baseAccountFreezer

	freeze bool
}

// NewFreezeAccountFunc will instantiate a new freeze account built-in function
func NewFreezeAccountFunc(args FreezeAccountArgs) (*freezeAccount, error) {
	function := getFunc(args.Freeze)

	base, err := newBaseAccountFreezer(args.BaseAccountFreezerArgs)
	if err != nil {
		return nil, err
	}
	freezeAccountFunc := &freezeAccount{
		freeze: args.Freeze,
	}
	freezeAccountFunc.baseEnabled = &baseEnabled{
		function:        function,
		activationEpoch: args.FreezeAccountEnableEpoch,
		flagActivated:   atomic.Flag{},
	}
	freezeAccountFunc.baseAccountFreezer = base

	logFreezeAccount.Debug(fmt.Sprintf("%s enable epoch:", function), args.FreezeAccountEnableEpoch)
	args.EpochNotifier.RegisterNotifyHandler(freezeAccountFunc)

	return freezeAccountFunc, nil
}

func getFunc(freeze bool) string {
	function := BuiltInFunctionUnfreezeAccount
	if freeze {
		function = BuiltInFunctionFreezeAccount
	}

	return function
}

// ProcessBuiltinFunction will set/unset the frozen bit in
// user's code metadata, if it has at least one enabled guardian
func (fa *freezeAccount) ProcessBuiltinFunction(
	senderAccount, receiverAccount vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	fa.mutExecution.Lock()
	defer fa.mutExecution.Unlock()

	err := fa.checkArgs(senderAccount, receiverAccount, vmInput, noOfArgsFreezeAccount)
	if err != nil {
		return nil, err
	}

	guardians, err := fa.guardians(senderAccount)
	if err != nil {
		return nil, err
	}
	if !fa.atLeastOneGuardianEnabled(guardians) {
		return nil, ErrNoGuardianEnabled
	}

	codeMetaDataBytes := senderAccount.GetCodeMetadata()
	codeMetaData := vmcommon.CodeMetadataFromBytes(codeMetaDataBytes)

	if fa.freeze {
		codeMetaData.Frozen = true // Todo: check if freeze acc tx came from first set guardian
	} else {
		codeMetaData.Frozen = false
	}

	senderAccount.SetCodeMetadata(codeMetaData.ToBytes())
	return &vmcommon.VMOutput{ReturnCode: vmcommon.Ok, GasRemaining: vmInput.GasProvided - fa.funcGasCost}, nil
}

func (fa *freezeAccount) atLeastOneGuardianEnabled(
	guardians *Guardians,
) bool {
	for _, guardian := range guardians.Data {
		if fa.enabled(guardian) {
			return true
		}
	}
	return false
}

// SetNewGasConfig is called whenever gas cost is changed
func (fa *freezeAccount) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	fa.mutExecution.Lock()
	fa.funcGasCost = gasCost.BuiltInCost.FreezeAccount
	fa.mutExecution.Unlock()
}
