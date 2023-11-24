package builtInFunctions

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"strings"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/atomic"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	mockvm "github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/require"
)

const pubKeyLen = 32

var userAddress = generateRandomByteArray(pubKeyLen)

func requireVMOutputOk(t *testing.T, output *vmcommon.VMOutput, gasProvided, gasCost uint64, entry *vmcommon.LogEntry) {
	expectedOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: gasProvided - gasCost,
		Logs:         []*vmcommon.LogEntry{entry},
	}
	require.Equal(t, expectedOutput, output)
}

func TestNewSetGuardianFunc(t *testing.T) {
	t.Parallel()

	tests := []struct {
		args        func() SetGuardianArgs
		expectedErr error
	}{
		{
			args: func() SetGuardianArgs {
				return createSetGuardianFuncMockArgs()
			},
			expectedErr: nil,
		},
		{
			args: func() SetGuardianArgs {
				args := createSetGuardianFuncMockArgs()
				args.EnableEpochsHandler = nil
				return args
			},
			expectedErr: ErrNilEnableEpochsHandler,
		},
	}

	for _, test := range tests {
		instance, err := NewSetGuardianFunc(test.args())
		if test.expectedErr != nil {
			require.Nil(t, instance)
			require.Equal(t, test.expectedErr, err)
		} else {
			require.NotNil(t, instance)
			require.Nil(t, err)
		}
	}
}

func TestSetGuardian_ProcessBuiltinFunctionCheckArguments(t *testing.T) {
	t.Parallel()

	address := generateRandomByteArray(pubKeyLen)
	account := mockvm.NewUserAccount(address)

	guardianAddress := generateRandomByteArray(pubKeyLen)
	serviceUID := []byte{1, 1, 1}
	vmInput := getDefaultVmInput([][]byte{guardianAddress, serviceUID})
	vmInput.CallerAddr = address
	vmInput.RecipientAddr = address

	tests := []struct {
		testname        string
		vmInput         func() *vmcommon.ContractCallInput
		senderAccount   vmcommon.UserAccountHandler
		receiverAccount vmcommon.UserAccountHandler
		expectedErr     error
	}{
		{
			testname: "builtin function called with value should error",
			vmInput: func() *vmcommon.ContractCallInput {
				input := *vmInput
				input.CallValue = big.NewInt(1)
				return &input
			},
			senderAccount:   account,
			receiverAccount: account,
			expectedErr:     ErrBuiltInFunctionCalledWithValue,
		},
		{
			testname: "nil guardian address should error",
			vmInput: func() *vmcommon.ContractCallInput {
				input := *vmInput
				input.Arguments = [][]byte{nil, serviceUID}
				return &input
			},
			senderAccount:   account,
			receiverAccount: account,
			expectedErr:     ErrInvalidAddress,
		},
		{
			testname: "builtin sender address as guardian should error",
			vmInput: func() *vmcommon.ContractCallInput {
				input := *vmInput
				input.Arguments = [][]byte{address, serviceUID}
				return &input
			},
			senderAccount:   account,
			receiverAccount: account,
			expectedErr:     ErrCannotSetOwnAddressAsGuardian,
		},
		{
			testname: "empty sc address as guardian should error",
			vmInput: func() *vmcommon.ContractCallInput {
				input := *vmInput
				input.Arguments = [][]byte{make([]byte, pubKeyLen), serviceUID} // Empty SC Address
				return &input
			},
			senderAccount:   account,
			receiverAccount: account,
			expectedErr:     ErrInvalidAddress,
		},

		{
			testname: "nil account should error",
			vmInput: func() *vmcommon.ContractCallInput {
				return vmInput
			},
			senderAccount:   nil,
			receiverAccount: account,
			expectedErr:     ErrNilUserAccount,
		},
		{
			testname: "nil vm input should error",
			vmInput: func() *vmcommon.ContractCallInput {
				return nil
			},
			senderAccount:   account,
			receiverAccount: account,
			expectedErr:     ErrNilVmInput,
		},
		{
			testname: "nil vm input arguments should error",
			vmInput: func() *vmcommon.ContractCallInput {
				input := *vmInput
				input.Arguments = nil
				return &input
			},
			senderAccount:   account,
			receiverAccount: account,
			expectedErr:     ErrInvalidNumberOfArguments,
		},
		{
			testname: "one input arguments should error",
			vmInput: func() *vmcommon.ContractCallInput {
				input := *vmInput
				input.Arguments = [][]byte{guardianAddress}
				return &input
			},
			senderAccount:   account,
			receiverAccount: account,
			expectedErr:     ErrInvalidNumberOfArguments,
		},
		{
			testname: "sender different to caller should error",
			vmInput: func() *vmcommon.ContractCallInput {
				return vmInput
			},
			senderAccount:   mockvm.NewUserAccount([]byte("userAddress2")),
			receiverAccount: account,
			expectedErr:     ErrOperationNotPermitted,
		},
		{
			testname: "OK",
			vmInput: func() *vmcommon.ContractCallInput {
				return vmInput
			},
			senderAccount:   account,
			receiverAccount: account,
			expectedErr:     nil,
		},
	}

	args := createSetGuardianFuncMockArgs()
	setGuardianFunc, _ := NewSetGuardianFunc(args)

	for _, test := range tests {
		instance, err := setGuardianFunc.ProcessBuiltinFunction(test.senderAccount, test.receiverAccount, test.vmInput())
		if test.expectedErr != nil {
			require.Nil(t, instance)
			require.Error(t, err)
			require.True(t, strings.Contains(err.Error(), test.expectedErr.Error()))
		} else {
			require.NotNil(t, instance)
			require.Nil(t, err)
		}
	}
}

func TestSetGuardian_SetNewGasConfig(t *testing.T) {
	t.Parallel()

	args := createSetGuardianFuncMockArgs()
	setGuardianFunc, _ := NewSetGuardianFunc(args)
	require.Equal(t, args.FuncGasCost, setGuardianFunc.funcGasCost)

	newSetGuardianCost := args.FuncGasCost + 1
	newGasCost := &vmcommon.GasCost{BuiltInCost: vmcommon.BuiltInCost{SetGuardian: newSetGuardianCost}}
	setGuardianFunc.SetNewGasConfig(newGasCost)
	require.Equal(t, newSetGuardianCost, setGuardianFunc.funcGasCost)
}

func TestSetGuardian_ProcessBuiltinFunctionAccountAccountHandlerSetError(t *testing.T) {
	t.Parallel()

	args := createSetGuardianFuncMockArgs()
	newGuardianAddress := generateRandomByteArray(pubKeyLen)
	serviceUID := []byte{1, 1, 1}
	vmInput := getDefaultVmInput([][]byte{newGuardianAddress, serviceUID})
	expectedErr := errors.New("expected error")

	address := generateRandomByteArray(pubKeyLen)
	account := mockvm.NewUserAccount(address)
	vmInput.CallerAddr = address
	vmInput.RecipientAddr = address

	args.GuardedAccountHandler = &mockvm.GuardedAccountHandlerStub{
		SetGuardianCalled: func(_ vmcommon.UserAccountHandler, guardianAddress []byte, _ []byte, _ []byte) error {
			return expectedErr
		},
	}

	setGuardianFunc, _ := NewSetGuardianFunc(args)
	output, err := setGuardianFunc.ProcessBuiltinFunction(account, account, vmInput)
	require.Equal(t, expectedErr, err)
	require.Nil(t, output)
}

func TestSetGuardian_ProcessBuiltinFunctionSetGuardianOK(t *testing.T) {
	t.Parallel()

	setGuardianCalled := atomic.Flag{}
	account := &mockvm.UserAccountStub{
		Address: userAddress,
	}

	args := createSetGuardianFuncMockArgs()
	args.EnableEpochsHandler = &mockvm.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == SetGuardianFlag
		},
	}
	args.GuardedAccountHandler = &mockvm.GuardedAccountHandlerStub{
		SetGuardianCalled: func(_ vmcommon.UserAccountHandler, _ []byte, _ []byte, _ []byte) error {
			setGuardianCalled.SetValue(true)
			return nil
		},
	}

	setGuardianFunc, _ := NewSetGuardianFunc(args)

	serviceUID := []byte{1, 1, 1}
	vmInput := getDefaultVmInput([][]byte{generateRandomByteArray(pubKeyLen), serviceUID})
	fmt.Println(userAddress)
	fmt.Println(vmInput.CallerAddr)
	output, err := setGuardianFunc.ProcessBuiltinFunction(account, account, vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, output.ReturnCode)
	require.True(t, setGuardianCalled.IsSet())
	require.Equal(t, []*vmcommon.LogEntry{
		{
			Address:    userAddress,
			Identifier: []byte(core.BuiltInFunctionSetGuardian),
			Topics:     [][]byte{vmInput.Arguments[0], serviceUID},
		},
	}, output.Logs)
}

func generateRandomByteArray(size uint32) []byte {
	ret := make([]byte, size)
	_, _ = rand.Read(ret)
	return ret
}

func createSetGuardianFuncMockArgs() SetGuardianArgs {
	return SetGuardianArgs{
		BaseAccountGuarderArgs: createBaseAccountGuarderArgs(),
	}
}

func getDefaultVmInput(args [][]byte) *vmcommon.ContractCallInput {
	return &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  userAddress,
			Arguments:   args,
			CallValue:   big.NewInt(0),
			GasProvided: 500000,
		},
		RecipientAddr: userAddress,
	}
}
