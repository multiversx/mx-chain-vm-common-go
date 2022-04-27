package builtInFunctions

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"strings"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core/atomic"
	guardiansData "github.com/ElrondNetwork/elrond-go-core/data/guardians"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	mockvm "github.com/ElrondNetwork/elrond-vm-common/mock"
	"github.com/stretchr/testify/require"
)

const pubKeyLen = 32
const currentEpoch = 44444

var userAddress = generateRandomByteArray(pubKeyLen)

func requireAccountHasGuardians(t *testing.T, account vmcommon.UserAccountHandler, guardians *guardiansData.Guardians) {
	marshalledData, err := account.AccountDataHandler().RetrieveValue(guardianKey)
	require.Nil(t, err)

	storedGuardian := &guardiansData.Guardians{}
	err = marshallerMock.Unmarshal(storedGuardian, marshalledData)
	require.Nil(t, err)
	require.Equal(t, guardians, storedGuardian)
}

func requireVMOutputOk(t *testing.T, output *vmcommon.VMOutput, gasProvided, gasCost uint64) {
	expectedOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: gasProvided - gasCost,
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
				args.EpochNotifier = nil
				return args
			},
			expectedErr: ErrNilEpochNotifier,
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
	vmInput := getDefaultVmInput([][]byte{guardianAddress})
	vmInput.CallerAddr = address

	tests := []struct {
		vmInput         func() *vmcommon.ContractCallInput
		senderAccount   vmcommon.UserAccountHandler
		receiverAccount vmcommon.UserAccountHandler
		expectedErr     error
	}{
		{
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
			vmInput: func() *vmcommon.ContractCallInput {
				input := *vmInput
				input.Arguments = [][]byte{nil}
				return &input
			},
			senderAccount:   account,
			receiverAccount: account,
			expectedErr:     ErrInvalidAddress,
		},
		{
			vmInput: func() *vmcommon.ContractCallInput {
				input := *vmInput
				input.Arguments = [][]byte{address}
				return &input
			},
			senderAccount:   account,
			receiverAccount: account,
			expectedErr:     ErrCannotSetOwnAddressAsGuardian,
		},
		{
			vmInput: func() *vmcommon.ContractCallInput {
				input := *vmInput
				input.Arguments = [][]byte{make([]byte, pubKeyLen)} // Empty SC Address
				return &input
			},
			senderAccount:   account,
			receiverAccount: account,
			expectedErr:     ErrInvalidAddress,
		},
		{
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
	vmInput := getDefaultVmInput([][]byte{newGuardianAddress})
	expectedErr := errors.New("expected error")

	address := generateRandomByteArray(pubKeyLen)
	account := mockvm.NewUserAccount(address)
	vmInput.CallerAddr = address

	args.GuardedAccountHandler = &mockvm.GuardedAccountHandlerStub{
		SetGuardianCalled: func(uah vmcommon.UserAccountHandler, guardianAddress []byte) error {
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
	args.GuardedAccountHandler = &mockvm.GuardedAccountHandlerStub{
		SetGuardianCalled: func(uah vmcommon.UserAccountHandler, guardianAddress []byte) error {
			setGuardianCalled.SetValue(true)
			return nil
		},
	}

	setGuardianFunc, _ := NewSetGuardianFunc(args)
	setGuardianFunc.EpochConfirmed(currentEpoch, 0)

	vmInput := getDefaultVmInput([][]byte{generateRandomByteArray(pubKeyLen)})
	fmt.Println(userAddress)
	fmt.Println(vmInput.CallerAddr)
	output, err := setGuardianFunc.ProcessBuiltinFunction(account, account, vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, output.ReturnCode)
	require.True(t, setGuardianCalled.IsSet())
}

func generateRandomByteArray(size uint32) []byte {
	ret := make([]byte, size)
	_, _ = rand.Read(ret)
	return ret
}

func Test_test(t *testing.T) {
	a := generateRandomByteArray(32)
	b := generateRandomByteArray(32)

	require.NotEqual(t, a, b)
	fmt.Println(a)
	fmt.Println(b)
}

func createSetGuardianFuncMockArgs() SetGuardianArgs {
	return SetGuardianArgs{
		BaseAccountFreezerArgs:   createBaseAccountFreezerArgs(),
		GuardianActivationEpochs: 100,
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
	}
}
