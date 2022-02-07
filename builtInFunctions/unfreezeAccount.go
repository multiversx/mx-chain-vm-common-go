package builtInFunctions

import (
	"github.com/ElrondNetwork/elrond-go-core/core"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

type unfreezeAccountFunc struct {
	*baseFreezeAccount
}

// NewUnfreezeAccountFunc will instantiate a new unfreeze account built-in function
func NewUnfreezeAccountFunc(args FreezeAccountArgs) (*unfreezeAccountFunc, error) {
	base, err := newBaseFreezeAccount(args, core.BuiltInFunctionUnfreezeAccount)
	if err != nil {
		return nil, err
	}
	return &unfreezeAccountFunc{baseFreezeAccount: base}, nil
}

// ProcessBuiltinFunction will unset the frozen bit in
// user's code metadata, if it has at least one enabled guardian
func (ua *unfreezeAccountFunc) ProcessBuiltinFunction(
	acntSnd, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	ua.mutExecution.Lock()
	defer ua.mutExecution.Unlock()

	err := ua.checkFreezeAccountArgs(acntSnd, acntDst, vmInput)
	if err != nil {
		return nil, err
	}

	err = unfreezeAccount(acntSnd)
	if err != nil {
		return nil, err
	}

	return &vmcommon.VMOutput{ReturnCode: vmcommon.Ok, GasRemaining: vmInput.GasProvided - ua.funcGasCost}, nil
}

func unfreezeAccount(account vmcommon.UserAccountHandler) error {
	codeMetaData := getCodeMetaData(account)
	if !codeMetaData.Frozen {
		return ErrSetUnfreezeAccount
	}

	codeMetaData.Frozen = false
	account.SetCodeMetadata(codeMetaData.ToBytes())
	return nil
}
