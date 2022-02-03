package builtInFunctions

import (
	"strings"
	"testing"

	guardiansData "github.com/ElrondNetwork/elrond-go-core/data/guardians"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/stretchr/testify/require"
)

func TestUnfreezeAccountFunc_ProcessBuiltinFunction(t *testing.T) {
	t.Parallel()

	args := createFreezeAccountArgs()
	vmInput := getDefaultVmInput([][]byte{})
	unfreezeAccountFunc, _ := NewUnfreezeAccountFunc(args)
	unfreezeAccountFunc.EpochConfirmed(currentEpoch, 0)

	t.Run("invalid args, expect error", func(t *testing.T) {
		output, err := unfreezeAccountFunc.ProcessBuiltinFunction(nil, nil, vmInput)
		require.Nil(t, output)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), ErrNilUserAccount.Error()))
	})

	t.Run("account has no enabled guardian, expect error", func(t *testing.T) {
		pendingGuardian := &guardiansData.Guardian{
			Address:         generateRandomByteArray(pubKeyLen),
			ActivationEpoch: currentEpoch + 1,
		}
		guardians := &guardiansData.Guardians{Data: []*guardiansData.Guardian{pendingGuardian}}

		account := createUserAccountWithGuardians(t, guardians)
		code := vmcommon.CodeMetadata{Frozen: true}
		account.SetCodeMetadata(code.ToBytes())

		output, err := unfreezeAccountFunc.ProcessBuiltinFunction(account, account, vmInput)
		require.Nil(t, output)
		require.Equal(t, ErrNoGuardianEnabled, err)
		requireAccountFrozen(t, account, true)
	})

	t.Run("unfreeze account should work", func(t *testing.T) {
		enabledGuardian := &guardiansData.Guardian{
			Address:         generateRandomByteArray(pubKeyLen),
			ActivationEpoch: currentEpoch - 1,
		}
		guardians := &guardiansData.Guardians{Data: []*guardiansData.Guardian{enabledGuardian}}

		account := createUserAccountWithGuardians(t, guardians)
		code := vmcommon.CodeMetadata{Frozen: true}
		account.SetCodeMetadata(code.ToBytes())
		requireAccountFrozen(t, account, true)

		output, err := unfreezeAccountFunc.ProcessBuiltinFunction(account, account, vmInput)
		require.Nil(t, err)
		requireVMOutputOk(t, output, vmInput.GasProvided, args.FuncGasCost)
		requireAccountFrozen(t, account, false)
	})
}
