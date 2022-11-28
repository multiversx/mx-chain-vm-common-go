package builtInFunctions

import (
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

const noOfArgsFreezeAccount = 0

// FreezeAccountArgs is a struct placeholder for all necessary args
// to create either a NewFreezeAccountFunc or a NewUnfreezeAccountFunc
type FreezeAccountArgs struct {
	BaseAccountFreezerArgs
}

type baseFreezeAccount struct {
	*baseAccountFreezer
}

func newBaseFreezeAccount(args FreezeAccountArgs) (*baseFreezeAccount, error) {
	base, err := newBaseAccountFreezer(args.BaseAccountFreezerArgs)
	if err != nil {
		return nil, err
	}

	baseFreezeAcc := &baseFreezeAccount{
		base,
	}

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
