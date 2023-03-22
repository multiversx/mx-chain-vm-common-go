package builtInFunctions

import vmcommon "github.com/multiversx/mx-chain-vm-common-go"

const noOfArgsGuardAccount = 0

// GuardAccountArgs is a struct placeholder for all necessary args
// to create either a NewGuardAccountFunc or a NewUnGuardAccountFunc
type GuardAccountArgs struct {
	BaseAccountGuarderArgs
}

type baseGuardAccount struct {
	*baseAccountGuarder
}

func newBaseGuardAccount(args GuardAccountArgs) (*baseGuardAccount, error) {
	base, err := newBaseAccountGuarder(args.BaseAccountGuarderArgs)
	if err != nil {
		return nil, err
	}

	baseGuardAcc := &baseGuardAccount{
		base,
	}

	return baseGuardAcc, nil
}

func (bfa *baseGuardAccount) checkGuardAccountArgs(
	acntSnd vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) error {
	err := bfa.checkBaseAccountGuarderArgs(acntSnd, vmInput, noOfArgsGuardAccount)
	if err != nil {
		return err
	}

	// cannot guard if account has no active guardian
	_, err = bfa.guardedAccountHandler.GetActiveGuardian(acntSnd)
	return err
}

func getCodeMetaData(account vmcommon.UserAccountHandler) vmcommon.CodeMetadata {
	codeMetaDataBytes := account.GetCodeMetadata()
	return vmcommon.CodeMetadataFromBytes(codeMetaDataBytes)
}

// SetNewGasConfig is called whenever gas cost is changed
func (bfa *baseGuardAccount) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	bfa.mutExecution.Lock()
	bfa.funcGasCost = gasCost.BuiltInCost.GuardAccount
	bfa.mutExecution.Unlock()
}
