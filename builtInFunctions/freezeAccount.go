package builtInFunctions

import (
	"fmt"
	"sync"

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
	FuncGasCost              uint64
	FreezeAccountEnableEpoch uint32
	Freeze                   bool
	EpochNotifier            vmcommon.EpochNotifier
}

type freezeAccount struct {
	*accountFreezerBase
	*baseEnabled
	freeze       bool
	funcGasCost  uint64
	mutExecution sync.RWMutex
}

func NewFreezeAccountFunc(args FreezeAccountArgs) (*freezeAccount, error) {
	function := BuiltInFunctionFreezeAccount
	if !args.Freeze {
		function = BuiltInFunctionUnfreezeAccount
	}

	freezeAccountFunc := &freezeAccount{
		freeze:       args.Freeze,
		mutExecution: sync.RWMutex{},
	}
	freezeAccountFunc.baseEnabled = &baseEnabled{
		function:        function,
		activationEpoch: args.FreezeAccountEnableEpoch,
		flagActivated:   atomic.Flag{},
	}

	logFreezeAccount.Debug(fmt.Sprintf("%s enable epoch:", function), args.FreezeAccountEnableEpoch)
	args.EpochNotifier.RegisterNotifyHandler(freezeAccountFunc)

	return freezeAccountFunc, nil
}

func (fa *freezeAccount) ProcessBuiltinFunction(
	senderAccount, receiverAccount vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	fa.mutExecution.Lock()
	defer fa.mutExecution.Unlock()

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

func (fa *freezeAccount) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	fa.mutExecution.Lock()
	fa.funcGasCost = gasCost.BuiltInCost.FreezeAccount
	fa.mutExecution.Unlock()
}
