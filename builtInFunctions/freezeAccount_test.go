package builtInFunctions

import (
	"testing"

	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/stretchr/testify/require"
)

func TestFreezeAccount_ProcessBuiltinFunction_Unfreeze_ExpectAccountUnfrozen(t *testing.T) {
	args := createFreezeAccountArgs(false)
	freezeAccountFunc, _ := NewFreezeAccountFunc(args)

	enabledGuardian := &Guardian{
		Address:         generateRandomByteArray(pubKeyLen),
		ActivationEpoch: args.BlockChainHook.CurrentEpoch() - 1,
	}
	guardians := &Guardians{Data: []*Guardian{enabledGuardian}}

	account := createUserAccountWithGuardians(t, guardians)

	code := vmcommon.CodeMetadata{Frozen: true}
	account.SetCodeMetadata(code.ToBytes())
	codeMetaData := account.GetCodeMetadata()
	codeMeta := vmcommon.CodeMetadataFromBytes(codeMetaData)

	vmInput := getDefaultVmInput(BuiltInFunctionFreezeAccount, [][]byte{})
	_, err := freezeAccountFunc.ProcessBuiltinFunction(account, account, vmInput)
	require.Nil(t, err)

	codeMetaData = account.GetCodeMetadata()
	codeMeta = vmcommon.CodeMetadataFromBytes(codeMetaData)
	require.False(t, codeMeta.Frozen)
}

func TestFreezeAccount_ProcessBuiltinFunction_Freeze_AccountDoesNotHaveEnabledGuardian_ExpectError(t *testing.T) {
	args := createFreezeAccountArgs(true)
	freezeAccountFunc, _ := NewFreezeAccountFunc(args)

	pendingGuardian := &Guardian{
		Address:         generateRandomByteArray(pubKeyLen),
		ActivationEpoch: args.BlockChainHook.CurrentEpoch() + 1,
	}
	guardians := &Guardians{Data: []*Guardian{pendingGuardian}}

	account := createUserAccountWithGuardians(t, guardians)
	vmInput := getDefaultVmInput(BuiltInFunctionFreezeAccount, [][]byte{})
	_, err := freezeAccountFunc.ProcessBuiltinFunction(account, account, vmInput)
	require.Equal(t, ErrNoGuardianEnabled, err)
}

func TestFreezeAccount_ProcessBuiltinFunction_AccountHasOneEnabledGuardian_ExpectAccountFrozen(t *testing.T) {
	args := createFreezeAccountArgs(true)
	freezeAccountFunc, _ := NewFreezeAccountFunc(args)

	enabledGuardian := &Guardian{
		Address:         generateRandomByteArray(pubKeyLen),
		ActivationEpoch: args.BlockChainHook.CurrentEpoch() - 1,
	}
	guardians := &Guardians{Data: []*Guardian{enabledGuardian}}

	account := createUserAccountWithGuardians(t, guardians)

	vmInput := getDefaultVmInput(BuiltInFunctionFreezeAccount, [][]byte{})
	_, err := freezeAccountFunc.ProcessBuiltinFunction(account, account, vmInput)
	require.Nil(t, err)

	codeMetaData := account.GetCodeMetadata()
	codeMeta := vmcommon.CodeMetadataFromBytes(codeMetaData)
	require.True(t, codeMeta.Frozen)

}

func createFreezeAccountArgs(freeze bool) FreezeAccountArgs {
	baseArgs := createBaseAccountFreezerArgs()

	return FreezeAccountArgs{
		BaseAccountFreezerArgs:   baseArgs,
		Freeze:                   freeze,
		FreezeAccountEnableEpoch: 1000,
	}
}
