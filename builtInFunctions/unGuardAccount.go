package builtInFunctions

import vmcommon "github.com/multiversx/mx-chain-vm-common-go"

type unGuardAccountFunc struct {
	*baseGuardAccount
}

// NewUnGuardAccountFunc will instantiate a new un-guard account built-in function
func NewUnGuardAccountFunc(args GuardAccountArgs) (*unGuardAccountFunc, error) {
	base, err := newBaseGuardAccount(args)
	if err != nil {
		return nil, err
	}
	return &unGuardAccountFunc{baseGuardAccount: base}, nil
}

// ProcessBuiltinFunction will unset the frozen bit in
// user's code metadata, if it has at least one enabled guardian
func (ua *unGuardAccountFunc) ProcessBuiltinFunction(
	acntSnd, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	ua.mutExecution.Lock()
	defer ua.mutExecution.Unlock()

	err := ua.checkGuardAccountArgs(acntSnd, vmInput)
	if err != nil {
		return nil, err
	}

	err = unGuardAccount(acntSnd)
	if err != nil {
		return nil, err
	}

	return &vmcommon.VMOutput{ReturnCode: vmcommon.Ok, GasRemaining: vmInput.GasProvided - ua.funcGasCost}, nil
}

func unGuardAccount(account vmcommon.UserAccountHandler) error {
	codeMetaData := getCodeMetaData(account)
	if !codeMetaData.Guarded {
		return ErrSetUnGuardAccount
	}

	codeMetaData.Guarded = false
	account.SetCodeMetadata(codeMetaData.ToBytes())
	return nil
}
