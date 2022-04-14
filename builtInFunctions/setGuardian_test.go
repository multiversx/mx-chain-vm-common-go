package builtInFunctions

import (
	"encoding/hex"
	"errors"
	"math/big"
	"math/rand"
	"strings"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core/atomic"
	guardiansData "github.com/ElrondNetwork/elrond-go-core/data/guardians"
	"github.com/ElrondNetwork/elrond-go-core/data/mock"
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

func createUserAccountWithGuardians(t *testing.T, guardians *guardiansData.Guardians) vmcommon.UserAccountHandler {
	marshalledGuardians, err := marshallerMock.Marshal(guardians)
	require.Nil(t, err)

	account := mockvm.NewUserAccount(userAddress)
	err = account.SaveKeyValue(guardianKey, marshalledGuardians)
	require.Nil(t, err)

	return account
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

func TestSetGuardian_ProcessBuiltinFunctionAccountHasThreeGuardiansExpectError(t *testing.T) {
	t.Parallel()

	args := createSetGuardianFuncMockArgs()
	guardian1 := &guardiansData.Guardian{Address: generateRandomByteArray(pubKeyLen)}
	guardian2 := &guardiansData.Guardian{Address: generateRandomByteArray(pubKeyLen)}
	guardian3 := &guardiansData.Guardian{Address: generateRandomByteArray(pubKeyLen)}
	guardians := &guardiansData.Guardians{Data: []*guardiansData.Guardian{guardian1, guardian2, guardian3}}

	account := createUserAccountWithGuardians(t, guardians)
	newGuardianAddress := generateRandomByteArray(pubKeyLen)
	vmInput := getDefaultVmInput([][]byte{newGuardianAddress})

	setGuardianFunc, _ := NewSetGuardianFunc(args)
	output, err := setGuardianFunc.ProcessBuiltinFunction(account, account, vmInput)
	require.Nil(t, err)
	require.Equal(t, &vmcommon.VMOutput{ReturnCode: vmcommon.ExecutionFailed}, output)
	requireAccountHasGuardians(t, account, guardians)
}

func TestSetGuardian_ProcessBuiltinFunctionCannotUnMarshalGuardiansExpectError(t *testing.T) {
	t.Parallel()

	guardiansUnmarshalledBytes := []byte("guardiansUnmarshalledBytes")
	wasAccountAltered := atomic.Flag{}
	accountHandler := &mockvm.DataTrieTrackerStub{
		RetrieveValueCalled: func(key []byte) ([]byte, error) {
			return guardiansUnmarshalledBytes, nil
		},
		SaveKeyValueCalled: func(key []byte, value []byte) error {
			wasAccountAltered.SetValue(true)
			return nil
		},
	}
	account := &mockvm.UserAccountStub{
		Address: userAddress,
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return accountHandler
		},
	}

	errMarshaller := errors.New("error marshaller")
	marshaller := &mock.MarshalizerStub{
		UnmarshalCalled: func(obj interface{}, buff []byte) error {
			require.Equal(t, guardiansUnmarshalledBytes, buff)
			return errMarshaller
		},
	}

	args := createSetGuardianFuncMockArgs()
	args.Marshaller = marshaller

	setGuardianFunc, _ := NewSetGuardianFunc(args)
	setGuardianFunc.EpochConfirmed(currentEpoch, 0)

	vmInput := getDefaultVmInput([][]byte{generateRandomByteArray(pubKeyLen)})
	output, err := setGuardianFunc.ProcessBuiltinFunction(account, account, vmInput)
	require.Nil(t, output)
	require.Equal(t, errMarshaller, err)
	require.False(t, wasAccountAltered.IsSet())
}

func TestSetGuardian_ProcessBuiltinFunctionCannotMarshalGuardianExpectError(t *testing.T) {
	t.Parallel()

	args := createSetGuardianFuncMockArgs()
	guardian := &guardiansData.Guardian{
		Address:         generateRandomByteArray(pubKeyLen),
		ActivationEpoch: currentEpoch + args.GuardianActivationEpochs,
	}
	guardians := &guardiansData.Guardians{Data: []*guardiansData.Guardian{guardian}}

	errMarshaller := errors.New("error marshaller")
	marshaller := &mock.MarshalizerStub{
		MarshalCalled: func(obj interface{}) ([]byte, error) {
			require.Equal(t, guardians, obj)
			return nil, errMarshaller
		},
	}
	args.Marshaller = marshaller

	account := mockvm.NewUserAccount(userAddress)
	vmInput := getDefaultVmInput([][]byte{guardian.Address})

	setGuardianFunc, _ := NewSetGuardianFunc(args)
	setGuardianFunc.EpochConfirmed(currentEpoch, 0)

	output, err := setGuardianFunc.ProcessBuiltinFunction(account, account, vmInput)
	require.Nil(t, output)
	require.Equal(t, errMarshaller, err)

	storedData, _ := account.AccountDataHandler().RetrieveValue(guardianKey)
	require.Nil(t, storedData)
}

func TestSetGuardian_ProcessBuiltinFunctionCannotRetrieveOwnerGuardiansExpectError(t *testing.T) {
	t.Parallel()

	errRetrieveVal := errors.New("error retrieving value for key")
	wasAccountAltered := atomic.Flag{}
	accountHandler := &mockvm.DataTrieTrackerStub{
		RetrieveValueCalled: func(key []byte) ([]byte, error) {
			return nil, errRetrieveVal
		},
		SaveKeyValueCalled: func(key []byte, value []byte) error {
			wasAccountAltered.SetValue(true)
			return nil
		},
	}
	account := &mockvm.UserAccountStub{
		Address: userAddress,
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return accountHandler
		},
	}

	newGuardianAddress := generateRandomByteArray(pubKeyLen)
	vmInput := getDefaultVmInput([][]byte{newGuardianAddress})

	args := createSetGuardianFuncMockArgs()
	setGuardianFunc, _ := NewSetGuardianFunc(args)
	output, err := setGuardianFunc.ProcessBuiltinFunction(account, account, vmInput)
	require.Nil(t, output)
	require.Equal(t, errRetrieveVal, err)
	require.False(t, wasAccountAltered.IsSet())
}

func TestSetGuardian_ProcessBuiltinFunctionSetSameGuardianAddressExpectError(t *testing.T) {
	t.Parallel()

	args := createSetGuardianFuncMockArgs()
	guardian := &guardiansData.Guardian{Address: generateRandomByteArray(pubKeyLen)}
	guardians := &guardiansData.Guardians{Data: []*guardiansData.Guardian{guardian}}

	account := createUserAccountWithGuardians(t, guardians)
	vmInput := getDefaultVmInput([][]byte{guardian.Address})

	setGuardianFunc, _ := NewSetGuardianFunc(args)
	output, err := setGuardianFunc.ProcessBuiltinFunction(account, account, vmInput)
	require.Nil(t, output)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), ErrGuardianAlreadyExists.Error()))
	requireAccountHasGuardians(t, account, guardians)
}

func TestSetGuardian_ProcessBuiltinFunctionCase1AccountHasNoGuardianSet(t *testing.T) {
	t.Parallel()

	newGuardianAddress := generateRandomByteArray(pubKeyLen)
	vmInput := getDefaultVmInput([][]byte{newGuardianAddress})
	account := mockvm.NewUserAccount(userAddress)

	args := createSetGuardianFuncMockArgs()
	setGuardianFunc, _ := NewSetGuardianFunc(args)
	setGuardianFunc.EpochConfirmed(currentEpoch, 0)

	output, err := setGuardianFunc.ProcessBuiltinFunction(account, account, vmInput)
	require.Nil(t, err)
	requireVMOutputOk(t, output, vmInput.GasProvided, args.FuncGasCost)

	newGuardian := &guardiansData.Guardian{
		Address:         newGuardianAddress,
		ActivationEpoch: currentEpoch + args.GuardianActivationEpochs,
	}
	expectedStoredGuardians := &guardiansData.Guardians{Data: []*guardiansData.Guardian{newGuardian}}
	requireAccountHasGuardians(t, account, expectedStoredGuardians)
}

func TestSetGuardian_ProcessBuiltinFunctionCase2AccountHasOnePendingGuardian(t *testing.T) {
	t.Parallel()

	args := createSetGuardianFuncMockArgs()
	pendingGuardian := &guardiansData.Guardian{
		Address:         generateRandomByteArray(pubKeyLen),
		ActivationEpoch: currentEpoch + args.GuardianActivationEpochs,
	}
	guardians := &guardiansData.Guardians{Data: []*guardiansData.Guardian{pendingGuardian}}

	account := createUserAccountWithGuardians(t, guardians)
	newGuardianAddress := generateRandomByteArray(pubKeyLen)
	vmInput := getDefaultVmInput([][]byte{newGuardianAddress})

	setGuardianFunc, _ := NewSetGuardianFunc(args)
	output, err := setGuardianFunc.ProcessBuiltinFunction(account, account, vmInput)
	require.Nil(t, output)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), ErrOwnerAlreadyHasOneGuardianPending.Error()))
	require.True(t, strings.Contains(err.Error(), hex.EncodeToString(pendingGuardian.Address)))
	requireAccountHasGuardians(t, account, guardians)
}

func TestSetGuardian_ProcessBuiltinFunctionCase3AccountHasOneEnabledGuardian(t *testing.T) {
	t.Parallel()

	args := createSetGuardianFuncMockArgs()
	enabledGuardian := &guardiansData.Guardian{
		Address:         generateRandomByteArray(pubKeyLen),
		ActivationEpoch: currentEpoch - args.GuardianActivationEpochs,
	}
	guardians := &guardiansData.Guardians{Data: []*guardiansData.Guardian{enabledGuardian}}

	account := createUserAccountWithGuardians(t, guardians)
	newGuardianAddress := generateRandomByteArray(pubKeyLen)
	vmInput := getDefaultVmInput([][]byte{newGuardianAddress})

	setGuardianFunc, _ := NewSetGuardianFunc(args)
	setGuardianFunc.EpochConfirmed(currentEpoch, 0)

	output, err := setGuardianFunc.ProcessBuiltinFunction(account, account, vmInput)
	require.Nil(t, err)
	requireVMOutputOk(t, output, vmInput.GasProvided, args.FuncGasCost)

	newGuardian := &guardiansData.Guardian{
		Address:         newGuardianAddress,
		ActivationEpoch: currentEpoch + args.GuardianActivationEpochs,
	}
	expectedStoredGuardians := &guardiansData.Guardians{Data: []*guardiansData.Guardian{enabledGuardian, newGuardian}}
	requireAccountHasGuardians(t, account, expectedStoredGuardians)
}

func TestSetGuardian_ProcessBuiltinFunctionCase4AccountHasOneEnabledGuardianAndOnePendingGuardian(t *testing.T) {
	t.Parallel()

	args := createSetGuardianFuncMockArgs()
	enabledGuardian := &guardiansData.Guardian{
		Address:         generateRandomByteArray(pubKeyLen),
		ActivationEpoch: currentEpoch - args.GuardianActivationEpochs,
	}
	pendingGuardian := &guardiansData.Guardian{
		Address:         generateRandomByteArray(pubKeyLen),
		ActivationEpoch: currentEpoch + args.GuardianActivationEpochs,
	}
	guardians := &guardiansData.Guardians{Data: []*guardiansData.Guardian{enabledGuardian, pendingGuardian}}

	account := createUserAccountWithGuardians(t, guardians)
	newGuardianAddress := generateRandomByteArray(pubKeyLen)
	vmInput := getDefaultVmInput([][]byte{newGuardianAddress})

	setGuardianFunc, _ := NewSetGuardianFunc(args)
	output, err := setGuardianFunc.ProcessBuiltinFunction(account, account, vmInput)
	require.Nil(t, output)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), ErrOwnerAlreadyHasOneGuardianPending.Error()))
	require.True(t, strings.Contains(err.Error(), hex.EncodeToString(pendingGuardian.Address)))
	requireAccountHasGuardians(t, account, guardians)
}

func TestSetGuardian_ProcessBuiltinFunctionCase5OwnerHasTwoEnabledGuardians(t *testing.T) {
	t.Parallel()

	args := createSetGuardianFuncMockArgs()

	enabledGuardian1 := &guardiansData.Guardian{
		Address:         generateRandomByteArray(pubKeyLen),
		ActivationEpoch: currentEpoch - args.GuardianActivationEpochs,
	}
	enabledGuardian2 := &guardiansData.Guardian{
		Address:         generateRandomByteArray(pubKeyLen),
		ActivationEpoch: currentEpoch - args.GuardianActivationEpochs - 1,
	}
	guardians := &guardiansData.Guardians{Data: []*guardiansData.Guardian{enabledGuardian1, enabledGuardian2}}

	account := createUserAccountWithGuardians(t, guardians)
	newGuardianAddress := generateRandomByteArray(pubKeyLen)
	vmInput := getDefaultVmInput([][]byte{newGuardianAddress})

	setGuardianFunc, _ := NewSetGuardianFunc(args)
	setGuardianFunc.EpochConfirmed(currentEpoch, 0)

	output, err := setGuardianFunc.ProcessBuiltinFunction(account, account, vmInput)
	require.Nil(t, err)
	requireVMOutputOk(t, output, vmInput.GasProvided, args.FuncGasCost)

	newGuardian := &guardiansData.Guardian{
		Address:         newGuardianAddress,
		ActivationEpoch: currentEpoch + args.GuardianActivationEpochs,
	}
	expectedStoredGuardians := &guardiansData.Guardians{Data: []*guardiansData.Guardian{enabledGuardian2, newGuardian}}
	requireAccountHasGuardians(t, account, expectedStoredGuardians)
}

func generateRandomByteArray(size uint32) []byte {
	ret := make([]byte, size)
	_, _ = rand.Read(ret)
	return ret
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