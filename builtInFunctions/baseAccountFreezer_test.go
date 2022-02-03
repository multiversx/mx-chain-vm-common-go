package builtInFunctions

import (
	"errors"
	"math/big"
	"strings"
	"testing"

	guardiansData "github.com/ElrondNetwork/elrond-go-core/data/guardians"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	mockvm "github.com/ElrondNetwork/elrond-vm-common/mock"
	"github.com/stretchr/testify/require"
)

var marshallerMock = &mockvm.MarshalizerMock{}

func TestNewBaseAccountFreezer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		args        func() BaseAccountFreezerArgs
		expectedErr error
	}{
		{
			args: func() BaseAccountFreezerArgs {
				args := createBaseAccountFreezerArgs()
				args.Marshaller = nil
				return args
			},
			expectedErr: ErrNilMarshaller,
		},
		{
			args: func() BaseAccountFreezerArgs {
				args := createBaseAccountFreezerArgs()
				args.EpochNotifier = nil
				return args
			},
			expectedErr: ErrNilEpochNotifier,
		},
		{
			args: func() BaseAccountFreezerArgs {
				return createBaseAccountFreezerArgs()
			},
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		instance, err := newBaseAccountFreezer(test.args())
		if test.expectedErr != nil {
			require.Nil(t, instance)
			require.Equal(t, test.expectedErr, err)
		} else {
			require.NotNil(t, instance)
			require.Nil(t, err)
		}
	}
}

func TestBaseAccountFreezer_CheckArgs(t *testing.T) {
	t.Parallel()

	address := generateRandomByteArray(pubKeyLen)
	account := mockvm.NewUserAccount(address)

	guardianAddress := generateRandomByteArray(pubKeyLen)
	vmInput := getDefaultVmInput([][]byte{guardianAddress})
	vmInput.CallerAddr = address

	tests := []struct {
		vmInput         func() *vmcommon.ContractCallInput
		senderAccount   vmcommon.UserAccountHandler
		receiverAccount vmcommon.UserAccountHandler
		expectedErr     error
		noOfArgs        uint32
	}{
		{
			vmInput: func() *vmcommon.ContractCallInput {
				return vmInput
			},
			senderAccount:   nil,
			receiverAccount: account,
			expectedErr:     ErrNilUserAccount,
			noOfArgs:        1,
		},
		{
			vmInput: func() *vmcommon.ContractCallInput {
				return vmInput
			},
			senderAccount:   account,
			receiverAccount: nil,
			expectedErr:     ErrNilUserAccount,
			noOfArgs:        1,
		},
		{
			vmInput: func() *vmcommon.ContractCallInput {
				return nil
			},
			senderAccount:   account,
			receiverAccount: account,
			expectedErr:     ErrNilVmInput,
			noOfArgs:        1,
		},
		{
			vmInput: func() *vmcommon.ContractCallInput {
				return vmInput
			},
			senderAccount:   mockvm.NewUserAccount([]byte("userAddress2")),
			receiverAccount: account,
			expectedErr:     ErrOperationNotPermitted,
			noOfArgs:        1,
		},
		{
			vmInput: func() *vmcommon.ContractCallInput {
				return vmInput
			},
			senderAccount:   account,
			receiverAccount: mockvm.NewUserAccount([]byte("userAddress2")),
			expectedErr:     ErrOperationNotPermitted,
			noOfArgs:        1,
		},
		{
			vmInput: func() *vmcommon.ContractCallInput {
				input := *vmInput
				input.CallerAddr = []byte("userAddress2")
				return &input
			},
			senderAccount:   account,
			receiverAccount: account,
			expectedErr:     ErrOperationNotPermitted,
			noOfArgs:        1,
		},
		{
			vmInput: func() *vmcommon.ContractCallInput {
				input := *vmInput
				input.CallValue = nil
				return &input
			},
			senderAccount:   account,
			receiverAccount: account,
			expectedErr:     ErrNilValue,
			noOfArgs:        1,
		},
		{
			vmInput: func() *vmcommon.ContractCallInput {
				input := *vmInput
				input.CallValue = big.NewInt(1)
				return &input
			},
			senderAccount:   account,
			receiverAccount: account,
			expectedErr:     ErrBuiltInFunctionCalledWithValue,
			noOfArgs:        1,
		},
		{
			vmInput: func() *vmcommon.ContractCallInput {
				input := *vmInput
				input.Arguments = [][]byte{guardianAddress, guardianAddress}
				return &input
			},
			senderAccount:   account,
			receiverAccount: account,
			expectedErr:     ErrInvalidNumberOfArguments,
			noOfArgs:        1,
		},
		{
			vmInput: func() *vmcommon.ContractCallInput {
				input := *vmInput
				input.GasProvided = 0
				return &input
			},
			senderAccount:   account,
			receiverAccount: account,
			expectedErr:     ErrNotEnoughGas,
			noOfArgs:        1,
		},
		{
			vmInput: func() *vmcommon.ContractCallInput {
				return vmInput
			},
			senderAccount:   account,
			receiverAccount: account,
			expectedErr:     nil,
			noOfArgs:        1,
		},
	}

	args := createBaseAccountFreezerArgs()
	baseAccFreezer, _ := newBaseAccountFreezer(args)

	for _, test := range tests {
		err := baseAccFreezer.checkBaseAccountFreezerArgs(test.senderAccount, test.receiverAccount, test.vmInput(), test.noOfArgs)
		if test.expectedErr != nil {
			require.Error(t, err)
			require.True(t, strings.Contains(err.Error(), test.expectedErr.Error()))
		} else {
			require.Nil(t, err)
		}
	}
}

func TestBaseAccountFreezer_enabledGuardian(t *testing.T) {
	t.Parallel()

	args := createBaseAccountFreezerArgs()
	baf, _ := newBaseAccountFreezer(args)
	baf.EpochConfirmed(currentEpoch, 0)

	t.Run("cannot get user account guardians, expect error", func(t *testing.T) {
		errRetrieveVal := errors.New("error retrieving value for key")
		accountHandler := &mockvm.DataTrieTrackerStub{
			RetrieveValueCalled: func(key []byte) ([]byte, error) {
				return nil, errRetrieveVal
			},
		}
		account := &mockvm.UserAccountStub{
			Address: userAddress,
			AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
				return accountHandler
			},
		}

		enabledGuardian, err := baf.enabledGuardian(account)
		require.Nil(t, enabledGuardian)
		require.Equal(t, errRetrieveVal, err)
	})
	t.Run("nil account handler, expect error", func(t *testing.T) {
		t.Parallel()

		account := &mockvm.UserAccountStub{
			AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
				return nil
			},
		}
		enabledGuardian, err := baf.enabledGuardian(account)
		require.Nil(t, enabledGuardian)
		require.Equal(t, ErrNilAccountHandler, err)
	})
	t.Run("two enabled guardians, expect the most recent one is returned", func(t *testing.T) {
		t.Parallel()

		enabledGuardian1 := &guardiansData.Guardian{
			Address:         generateRandomByteArray(pubKeyLen),
			ActivationEpoch: currentEpoch - 1,
		}
		enabledGuardian2 := &guardiansData.Guardian{
			Address:         generateRandomByteArray(pubKeyLen),
			ActivationEpoch: currentEpoch - 2,
		}
		guardians := &guardiansData.Guardians{Data: []*guardiansData.Guardian{enabledGuardian1, enabledGuardian2}}
		account := createUserAccountWithGuardians(t, guardians)

		enabledGuardian, err := baf.enabledGuardian(account)
		require.Nil(t, err)
		require.Equal(t, enabledGuardian1, enabledGuardian)
	})
	t.Run("two guardians, none enabled, expect no guardian returned", func(t *testing.T) {
		t.Parallel()

		enabledGuardian1 := &guardiansData.Guardian{
			Address:         generateRandomByteArray(pubKeyLen),
			ActivationEpoch: currentEpoch + 1,
		}
		enabledGuardian2 := &guardiansData.Guardian{
			Address:         generateRandomByteArray(pubKeyLen),
			ActivationEpoch: currentEpoch + 2,
		}
		guardians := &guardiansData.Guardians{Data: []*guardiansData.Guardian{enabledGuardian1, enabledGuardian2}}
		account := createUserAccountWithGuardians(t, guardians)

		enabledGuardian, err := baf.enabledGuardian(account)
		require.Equal(t, ErrNoGuardianEnabled, err)
		require.Nil(t, enabledGuardian)
	})

}

func createBaseAccountFreezerArgs() BaseAccountFreezerArgs {
	return BaseAccountFreezerArgs{
		Marshaller:    marshallerMock,
		EpochNotifier: &mockvm.EpochNotifierStub{},
		FuncGasCost:   100000,
	}
}
