package builtInFunctions

import (
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

type freezeAccountFunc struct {
	*baseFreezeAccount
}

// NewFreezeAccountFunc will instantiate a new freeze account built-in function
func NewFreezeAccountFunc(args FreezeAccountArgs) (*freezeAccountFunc, error) {
	base, err := newBaseFreezeAccount(args)
	if err != nil {
		return nil, err
	}
	return &freezeAccountFunc{baseFreezeAccount: base}, nil
}

// ProcessBuiltinFunction will set the frozen bit in
// user's code metadata, if it has at least one enabled guardian
func (fa *freezeAccountFunc) ProcessBuiltinFunction(
	acntSnd, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	fa.mutExecution.Lock()
	defer fa.mutExecution.Unlock()

	err := fa.checkFreezeAccountArgs(acntSnd, acntDst, vmInput)
	if err != nil {
		return nil, err
	}

	err = freezeAccount(acntSnd)
	if err != nil {
		return nil, err
	}

	fa.guardedAccountHandler.CleanOtherThanActive(acntSnd)

	return &vmcommon.VMOutput{ReturnCode: vmcommon.Ok, GasRemaining: vmInput.GasProvided - fa.funcGasCost}, nil
}

func freezeAccount(account vmcommon.UserAccountHandler) error {
	codeMetaData := getCodeMetaData(account)
	if codeMetaData.Frozen {
		return ErrSetFreezeAccount
	}

	codeMetaData.Frozen = true
	account.SetCodeMetadata(codeMetaData.ToBytes())
	return nil
}
