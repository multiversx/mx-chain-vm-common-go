package builtInFunctions

import (
	"math/big"
	"strings"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	mockvm "github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/require"
)

var marshallerMock = &mockvm.MarshalizerMock{}

func TestNewBaseAccountGuarder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		args        func() BaseAccountGuarderArgs
		expectedErr error
	}{
		{
			args: func() BaseAccountGuarderArgs {
				args := createBaseAccountGuarderArgs()
				args.Marshaller = nil
				return args
			},
			expectedErr: ErrNilMarshalizer,
		},
		{
			args: func() BaseAccountGuarderArgs {
				args := createBaseAccountGuarderArgs()
				args.EnableEpochsHandler = nil
				return args
			},
			expectedErr: ErrNilEnableEpochsHandler,
		},
		{
			args: func() BaseAccountGuarderArgs {
				args := createBaseAccountGuarderArgs()
				args.GuardedAccountHandler = nil
				return args
			},
			expectedErr: ErrNilGuardedAccountHandler,
		},
		{
			args: func() BaseAccountGuarderArgs {
				return createBaseAccountGuarderArgs()
			},
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		instance, err := newBaseAccountGuarder(test.args())
		if test.expectedErr != nil {
			require.Nil(t, instance)
			require.Equal(t, test.expectedErr, err)
		} else {
			require.NotNil(t, instance)
			require.Nil(t, err)
		}
	}
}

func TestBaseAccountGuarder_CheckArgs(t *testing.T) {
	t.Parallel()

	address := generateRandomByteArray(pubKeyLen)
	account := mockvm.NewUserAccount(address)

	guardianAddress := generateRandomByteArray(pubKeyLen)
	vmInput := getDefaultVmInput([][]byte{guardianAddress})
	vmInput.CallerAddr = address
	vmInput.RecipientAddr = account.Address

	tests := []struct {
		testname      string
		vmInput       func() *vmcommon.ContractCallInput
		senderAccount vmcommon.UserAccountHandler
		expectedErr   error
		noOfArgs      uint32
	}{
		{
			testname: "operation not permitted for different sender and caller",
			vmInput: func() *vmcommon.ContractCallInput {
				return vmInput
			},
			senderAccount: mockvm.NewUserAccount([]byte("userAddress2")),
			expectedErr:   ErrOperationNotPermitted,
			noOfArgs:      1,
		},
		{
			testname: "nil transfer value should return error",
			vmInput: func() *vmcommon.ContractCallInput {
				input := *vmInput
				input.CallValue = nil
				return &input
			},
			senderAccount: account,
			expectedErr:   ErrNilValue,
			noOfArgs:      1,
		},
		{
			testname: "non zero transfer value should return error",
			vmInput: func() *vmcommon.ContractCallInput {
				input := *vmInput
				input.CallValue = big.NewInt(1)
				return &input
			},
			senderAccount: account,
			expectedErr:   ErrBuiltInFunctionCalledWithValue,
			noOfArgs:      1,
		},
		{
			testname: "invalid number of arguments should return error",
			vmInput: func() *vmcommon.ContractCallInput {
				input := *vmInput
				input.Arguments = [][]byte{guardianAddress, guardianAddress}
				return &input
			},
			senderAccount: account,
			expectedErr:   ErrInvalidNumberOfArguments,
			noOfArgs:      1,
		},
		{
			testname: "not enough gas should return error",
			vmInput: func() *vmcommon.ContractCallInput {
				input := *vmInput
				input.GasProvided = 0
				return &input
			},
			senderAccount: account,
			expectedErr:   ErrNotEnoughGas,
			noOfArgs:      1,
		},
		{
			testname: "valid arguments should return no error",
			vmInput: func() *vmcommon.ContractCallInput {
				return vmInput
			},
			senderAccount: account,
			expectedErr:   nil,
			noOfArgs:      1,
		},
	}

	args := createBaseAccountGuarderArgs()
	baseAccGuarder, _ := newBaseAccountGuarder(args)

	for _, test := range tests {
		err := baseAccGuarder.checkBaseAccountGuarderArgs(
			test.senderAccount.AddressBytes(),
			test.vmInput().RecipientAddr,
			test.vmInput().CallValue,
			test.vmInput().GasProvided,
			test.vmInput().Arguments,
			test.noOfArgs)
		if test.expectedErr != nil {
			require.Error(t, err)
			require.True(t, strings.Contains(err.Error(), test.expectedErr.Error()))
		} else {
			require.Nil(t, err)
		}
	}
}

func createBaseAccountGuarderArgs() BaseAccountGuarderArgs {
	return BaseAccountGuarderArgs{
		Marshaller: marshallerMock,
		EnableEpochsHandler: &mockvm.EnableEpochsHandlerStub{
			IsFlagEnabledInCurrentEpochCalled: func(flag core.EnableEpochFlag) bool {
				return false
			},
		},
		FuncGasCost:           100000,
		GuardedAccountHandler: &mockvm.GuardedAccountHandlerStub{},
	}
}
