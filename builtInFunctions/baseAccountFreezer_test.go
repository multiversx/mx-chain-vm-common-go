package builtInFunctions

import (
	"math/big"
	"strings"
	"testing"

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
			expectedErr: ErrNilMarshalizer,
		},
		{
			args: func() BaseAccountFreezerArgs {
				args := createBaseAccountFreezerArgs()
				args.EpochNotifier = nil
				return args
			},
			expectedErr: ErrNilEpochHandler,
		},
		{
			args: func() BaseAccountFreezerArgs {
				args := createBaseAccountFreezerArgs()
				args.GuardedAccountHandler = nil
				return args
			},
			expectedErr: ErrNilGuardedAccountHandler,
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

func createBaseAccountFreezerArgs() BaseAccountFreezerArgs {
	return BaseAccountFreezerArgs{
		Marshaller:            marshallerMock,
		EpochNotifier:         &mockvm.EpochNotifierStub{},
		FuncGasCost:           100000,
		GuardedAccountHandler: &mockvm.GuardedAccountHandlerStub{},
	}
}
