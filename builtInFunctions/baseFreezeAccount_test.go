package builtInFunctions

import (
	"testing"

	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/stretchr/testify/require"
)

func createFreezeAccountArgs() FreezeAccountArgs {
	return FreezeAccountArgs{
		BaseAccountFreezerArgs: createBaseAccountFreezerArgs(),
	}
}

func TestBaseFreezeAccount_SetNewGasConfig(t *testing.T) {
	t.Parallel()

	args := createFreezeAccountArgs()
	baseFreezeAccount, _ := newBaseFreezeAccount(args)
	require.Equal(t, args.FuncGasCost, baseFreezeAccount.funcGasCost)

	newFreezeAccountCost := args.FuncGasCost + 1
	newGasCost := &vmcommon.GasCost{BuiltInCost: vmcommon.BuiltInCost{FreezeAccount: newFreezeAccountCost}}

	baseFreezeAccount.SetNewGasConfig(newGasCost)
	require.Equal(t, newFreezeAccountCost, baseFreezeAccount.funcGasCost)
}
