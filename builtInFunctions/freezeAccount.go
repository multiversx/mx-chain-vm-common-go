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

var logFreezeAccount = logger.GetOrCreate("systemSmartContracts/freezeAccount")

type FreezeAccountArgs struct {
	BaseAccountFreezerArgs

	FreezeAccountEnableEpoch uint32
	Freeze                   bool
}

type freezeAccount struct {
	*baseAccountFreezer
	*baseEnabled
	freeze bool
}

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

func (fa *freezeAccount) ProcessBuiltinFunction(
	senderAccount, receiverAccount vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	fa.mutExecution.Lock()
	defer fa.mutExecution.Unlock()

	err := fa.checkBaseArgs(senderAccount, receiverAccount, vmInput, 0)
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

	accountCodeMetaData := senderAccount.GetCodeMetadata()
	codeMetaData := vmcommon.CodeMetadataFromBytes(accountCodeMetaData)

	if fa.freeze {
		codeMetaData.Frozen = true
	} else {
		codeMetaData.Frozen = false // Todo: check if this tx came from first guardian
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

func (fa *freezeAccount) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	fa.mutExecution.Lock()
	fa.funcGasCost = gasCost.BuiltInCost.FreezeAccount
	fa.mutExecution.Unlock()
}
