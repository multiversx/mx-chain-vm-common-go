package builtInFunctions

import (
	"fmt"

	"github.com/ElrondNetwork/elrond-go-core/core/atomic"
	logger "github.com/ElrondNetwork/elrond-go-logger"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

const noOfArgsFreezeAccount = 0

var logFreezeAccount = logger.GetOrCreate("systemSmartContracts/baseFreezeAccount")

// FreezeAccountArgs is a struct placeholder for all necessary args
// to create either a NewFreezeAccountFunc or a NewUnfreezeAccountFunc
type FreezeAccountArgs struct {
	BaseAccountFreezerArgs
	FreezeAccountEnableEpoch uint32
}

type baseFreezeAccount struct {
	*baseEnabled
	*baseAccountFreezer
}

func newBaseFreezeAccount(args FreezeAccountArgs, builtInFunc string) (*baseFreezeAccount, error) {
	base, err := newBaseAccountFreezer(args.BaseAccountFreezerArgs)
	if err != nil {
		return nil, err
	}

	baseFreezeAcc := &baseFreezeAccount{}
	baseFreezeAcc.baseEnabled = &baseEnabled{
		function:        builtInFunc,
		activationEpoch: args.FreezeAccountEnableEpoch,
		flagActivated:   atomic.Flag{},
	}
	baseFreezeAcc.baseAccountFreezer = base

	logFreezeAccount.Debug(fmt.Sprintf("%s enable epoch:", builtInFunc), args.FreezeAccountEnableEpoch)
	args.EpochNotifier.RegisterNotifyHandler(baseFreezeAcc)

	return baseFreezeAcc, nil
}

func (bfa *baseFreezeAccount) checkFreezeAccountArgs(
	acntSnd, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) error {
	err := bfa.checkBaseAccountFreezerArgs(acntSnd, acntDst, vmInput, noOfArgsFreezeAccount)
	if err != nil {
		return err
	}

	// cannot freeze if account has no active guardian
	_, err = bfa.guardedAccountHandler.GetActiveGuardian(acntSnd)
	return err
}

func getCodeMetaData(account vmcommon.UserAccountHandler) vmcommon.CodeMetadata {
	codeMetaDataBytes := account.GetCodeMetadata()
	return vmcommon.CodeMetadataFromBytes(codeMetaDataBytes)
}

// SetNewGasConfig is called whenever gas cost is changed
func (bfa *baseFreezeAccount) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	bfa.mutExecution.Lock()
	bfa.funcGasCost = gasCost.BuiltInCost.FreezeAccount
	bfa.mutExecution.Unlock()
}
