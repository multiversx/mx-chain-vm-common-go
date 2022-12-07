package builtInFunctions

import (
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

type guardAccountFunc struct {
	*baseGuardAccount
}

// NewGuardAccountFunc will instantiate a new guard account built-in function
func NewGuardAccountFunc(args GuardAccountArgs) (*guardAccountFunc, error) {
	base, err := newBaseGuardAccount(args)
	if err != nil {
		return nil, err
	}
	return &guardAccountFunc{baseGuardAccount: base}, nil
}

// ProcessBuiltinFunction will set the frozen bit in
// user's code metadata, if it has at least one enabled guardian
func (fa *guardAccountFunc) ProcessBuiltinFunction(
	acntSnd, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	fa.mutExecution.Lock()
	defer fa.mutExecution.Unlock()

	err := fa.checkGuardAccountArgs(acntSnd, acntDst, vmInput)
	if err != nil {
		return nil, err
	}

	err = guardAccount(acntSnd)
	if err != nil {
		return nil, err
	}

	fa.guardedAccountHandler.CleanOtherThanActive(acntSnd)

	return &vmcommon.VMOutput{ReturnCode: vmcommon.Ok, GasRemaining: vmInput.GasProvided - fa.funcGasCost}, nil
}

func guardAccount(account vmcommon.UserAccountHandler) error {
	codeMetaData := getCodeMetaData(account)
	if codeMetaData.Guarded {
		return ErrSetGuardAccountFlag
	}

	codeMetaData.Guarded = true
	account.SetCodeMetadata(codeMetaData.ToBytes())
	return nil
}
