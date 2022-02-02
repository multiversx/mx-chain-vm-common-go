package builtInFunctions

import (
	"fmt"

	"github.com/ElrondNetwork/elrond-go-core/core/atomic"
	logger "github.com/ElrondNetwork/elrond-go-logger"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

const (
	BuiltInFunctionFreezeAccount   = "FreezeAccount"
	BuiltInFunctionUnfreezeAccount = "UnfreezeAccount"
)

const noOfArgsFreezeAccount = 0

var logFreezeAccount = logger.GetOrCreate("systemSmartContracts/freezeAccount")

// FreezeAccountArgs is a struct placeholder for all
// necessary args to create a NewFreezeAccountFunc
type FreezeAccountArgs struct {
	BaseAccountFreezerArgs
	FreezeAccountEnableEpoch uint32
}

type freezeAccount struct {
	*baseEnabled
	*baseAccountFreezer

	freeze bool
}

// NewFreezeAccountFunc will instantiate a new freeze account built-in function
func NewFreezeAccountFunc(args FreezeAccountArgs) (*freezeAccount, error) {
	return newFreezeAccount(args, true, BuiltInFunctionFreezeAccount)
}

// NewUnfreezeAccountFunc will instantiate a new unfreeze account built-in function
func NewUnfreezeAccountFunc(args FreezeAccountArgs) (*freezeAccount, error) {
	return newFreezeAccount(args, false, BuiltInFunctionUnfreezeAccount)
}

func newFreezeAccount(args FreezeAccountArgs, freeze bool, builtInFunc string) (*freezeAccount, error) {
	base, err := newBaseAccountFreezer(args.BaseAccountFreezerArgs)
	if err != nil {
		return nil, err
	}
	freezeAccountFunc := &freezeAccount{
		freeze: freeze,
	}
	freezeAccountFunc.baseEnabled = &baseEnabled{
		function:        builtInFunc,
		activationEpoch: args.FreezeAccountEnableEpoch,
		flagActivated:   atomic.Flag{},
	}
	freezeAccountFunc.baseAccountFreezer = base

	logFreezeAccount.Debug(fmt.Sprintf("%s enable epoch:", builtInFunc), args.FreezeAccountEnableEpoch)
	args.EpochNotifier.RegisterNotifyHandler(freezeAccountFunc)

	return freezeAccountFunc, nil
}

// ProcessBuiltinFunction will set/unset the frozen bit in
// user's code metadata, if it has at least one enabled guardian
func (fa *freezeAccount) ProcessBuiltinFunction(
	acntSnd, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	fa.mutExecution.Lock()
	defer fa.mutExecution.Unlock()

	err := fa.checkBaseAccountFreezerArgs(acntSnd, acntDst, vmInput, noOfArgsFreezeAccount)
	if err != nil {
		return nil, err
	}

	_, err = fa.enabledGuardian(acntSnd)
	if err != nil {
		return nil, err
	}

	if fa.freeze {
		fa.freezeAccount(acntSnd)
	} else {
		fa.unfreezeAccount(acntSnd)
	}

	return &vmcommon.VMOutput{ReturnCode: vmcommon.Ok, GasRemaining: vmInput.GasProvided - fa.funcGasCost}, nil
}

func (fa *freezeAccount) freezeAccount(account vmcommon.UserAccountHandler) {
	codeMetaData := getCodeMetaData(account)
	codeMetaData.Frozen = true
	account.SetCodeMetadata(codeMetaData.ToBytes())
}

func (fa *freezeAccount) unfreezeAccount(account vmcommon.UserAccountHandler) {
	codeMetaData := getCodeMetaData(account)
	codeMetaData.Frozen = false
	account.SetCodeMetadata(codeMetaData.ToBytes())
}

func getCodeMetaData(account vmcommon.UserAccountHandler) vmcommon.CodeMetadata {
	codeMetaDataBytes := account.GetCodeMetadata()
	return vmcommon.CodeMetadataFromBytes(codeMetaDataBytes)
}

// SetNewGasConfig is called whenever gas cost is changed
func (fa *freezeAccount) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	fa.mutExecution.Lock()
	fa.funcGasCost = gasCost.BuiltInCost.FreezeAccount
	fa.mutExecution.Unlock()
}

// EpochConfirmed is called whenever a new epoch is confirmed
func (fa *freezeAccount) EpochConfirmed(epoch uint32, _ uint64) {
	fa.baseEnabled.EpochConfirmed(epoch, 0)
	fa.baseAccountFreezer.EpochConfirmed(epoch, 0)
}
