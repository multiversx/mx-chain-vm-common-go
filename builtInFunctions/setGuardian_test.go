package builtInFunctions

import (
	"math/big"
	"math/rand"
	"strings"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/data/mock"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	mockvm "github.com/ElrondNetwork/elrond-vm-common/mock"
	"github.com/stretchr/testify/require"
)

const pubKeyLen = 32

var userAddress = []byte("user address")
var marshallerMock = &mockvm.MarshalizerMock{}

func guardiansProtectedKey() []byte {
	return append([]byte(core.ElrondProtectedKeyPrefix), []byte(GuardianKeyIdentifier)...)
}

func requireAccountHasGuardians(t *testing.T, account vmcommon.UserAccountHandler, guardians *Guardians) {
	key := guardiansProtectedKey()

	marshalledData, err := account.AccountDataHandler().RetrieveValue(key)
	require.Nil(t, err)

	storedGuardian := &Guardians{}
	err = marshallerMock.Unmarshal(storedGuardian, marshalledData)
	require.Nil(t, err)
	require.Equal(t, guardians, storedGuardian)
}

func createUserAccountWithGuardians(t *testing.T, guardians *Guardians) vmcommon.UserAccountHandler {
	key := guardiansProtectedKey()

	marshalledGuardians, err := marshallerMock.Marshal(guardians)
	require.Nil(t, err)

	account := mockvm.NewUserAccount(userAddress)
	err = account.SaveKeyValue(key, marshalledGuardians)
	require.Nil(t, err)

	return account
}

func requireSetGuardianVMOutputOk(t *testing.T, output *vmcommon.VMOutput, gasProvided, gasCost uint64) {
	expectedOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: gasProvided - gasCost,
	}
	require.Equal(t, expectedOutput, output)
}

func TestSetGuardian_ProcessBuiltinFunctionCase1AccountHasNoGuardianSet(t *testing.T) {
	newGuardianAddress := generateRandomByteArray(pubKeyLen)
	vmInput := getDefaultVmInput(BuiltInFunctionSetGuardian, [][]byte{newGuardianAddress})
	account := mockvm.NewUserAccount([]byte("user address"))

	args := createSetGuardianFuncMockArgs()
	setGuardianFunc, _ := NewSetGuardianFunc(args)

	output, err := setGuardianFunc.ProcessBuiltinFunction(account, nil, vmInput)
	require.Nil(t, err)
	requireSetGuardianVMOutputOk(t, output, vmInput.GasProvided, args.FuncGasCost)

	newGuardian := &Guardian{
		Address:         newGuardianAddress,
		ActivationEpoch: args.BlockChainHook.CurrentEpoch() + args.GuardianActivationEpochs,
	}
	expectedStoredGuardians := &Guardians{Data: []*Guardian{newGuardian}}
	requireAccountHasGuardians(t, account, expectedStoredGuardians)
}

func TestSetGuardian_ProcessBuiltinFunctionCase2AccountHasOnePendingGuardian(t *testing.T) {
	args := createSetGuardianFuncMockArgs()
	pendingGuardian := &Guardian{
		Address:         generateRandomByteArray(pubKeyLen),
		ActivationEpoch: args.BlockChainHook.CurrentEpoch() + args.GuardianActivationEpochs,
	}
	guardians := &Guardians{Data: []*Guardian{pendingGuardian}}

	account := createUserAccountWithGuardians(t, guardians)
	newGuardianAddress := generateRandomByteArray(pubKeyLen)
	vmInput := getDefaultVmInput(BuiltInFunctionSetGuardian, [][]byte{newGuardianAddress})

	setGuardianFunc, _ := NewSetGuardianFunc(args)
	output, err := setGuardianFunc.ProcessBuiltinFunction(account, nil, vmInput)
	require.Nil(t, output)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), ErrOwnerAlreadyHasOneGuardianPending.Error()))
	require.True(t, strings.Contains(err.Error(), args.PubKeyConverter.Encode(pendingGuardian.Address)))
	requireAccountHasGuardians(t, account, guardians)
}

func TestSetGuardian_ProcessBuiltinFunctionCase3AccountHasOneEnabledGuardian(t *testing.T) {
	args := createSetGuardianFuncMockArgs()
	enabledGuardian := &Guardian{
		Address:         generateRandomByteArray(pubKeyLen),
		ActivationEpoch: args.BlockChainHook.CurrentEpoch() - args.GuardianActivationEpochs,
	}
	guardians := &Guardians{Data: []*Guardian{enabledGuardian}}

	account := createUserAccountWithGuardians(t, guardians)
	newGuardianAddress := generateRandomByteArray(pubKeyLen)
	vmInput := getDefaultVmInput(BuiltInFunctionSetGuardian, [][]byte{newGuardianAddress})

	setGuardianFunc, _ := NewSetGuardianFunc(args)
	output, err := setGuardianFunc.ProcessBuiltinFunction(account, nil, vmInput)
	require.Nil(t, err)
	requireSetGuardianVMOutputOk(t, output, vmInput.GasProvided, args.FuncGasCost)

	newGuardian := &Guardian{
		Address:         newGuardianAddress,
		ActivationEpoch: args.BlockChainHook.CurrentEpoch() + args.GuardianActivationEpochs,
	}
	expectedStoredGuardians := &Guardians{Data: []*Guardian{enabledGuardian, newGuardian}}
	requireAccountHasGuardians(t, account, expectedStoredGuardians)
}

func TestSetGuardian_ProcessBuiltinFunctionCase4AccountHasOneEnabledGuardianAndOnePendingGuardian(t *testing.T) {
	args := createSetGuardianFuncMockArgs()
	enabledGuardian := &Guardian{
		Address:         generateRandomByteArray(pubKeyLen),
		ActivationEpoch: args.BlockChainHook.CurrentEpoch() - args.GuardianActivationEpochs,
	}
	pendingGuardian := &Guardian{
		Address:         generateRandomByteArray(pubKeyLen),
		ActivationEpoch: args.BlockChainHook.CurrentEpoch() + args.GuardianActivationEpochs,
	}
	guardians := &Guardians{Data: []*Guardian{enabledGuardian, pendingGuardian}}

	account := createUserAccountWithGuardians(t, guardians)
	newGuardianAddress := generateRandomByteArray(pubKeyLen)
	vmInput := getDefaultVmInput(BuiltInFunctionSetGuardian, [][]byte{newGuardianAddress})

	setGuardianFunc, _ := NewSetGuardianFunc(args)
	output, err := setGuardianFunc.ProcessBuiltinFunction(account, nil, vmInput)
	require.Nil(t, output)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), ErrOwnerAlreadyHasOneGuardianPending.Error()))
	require.True(t, strings.Contains(err.Error(), args.PubKeyConverter.Encode(pendingGuardian.Address)))
	requireAccountHasGuardians(t, account, guardians)
}

func TestSetGuardian_ProcessBuiltinFunctionCase5OwnerHasTwoEnabledGuardians(t *testing.T) {
	args := createSetGuardianFuncMockArgs()

	enabledGuardian1 := &Guardian{
		Address:         generateRandomByteArray(pubKeyLen),
		ActivationEpoch: args.BlockChainHook.CurrentEpoch() - args.GuardianActivationEpochs,
	}
	enabledGuardian2 := &Guardian{
		Address:         generateRandomByteArray(pubKeyLen),
		ActivationEpoch: args.BlockChainHook.CurrentEpoch() - args.GuardianActivationEpochs - 1,
	}
	guardians := &Guardians{Data: []*Guardian{enabledGuardian1, enabledGuardian2}}

	account := createUserAccountWithGuardians(t, guardians)
	newGuardianAddress := generateRandomByteArray(pubKeyLen)
	vmInput := getDefaultVmInput(BuiltInFunctionSetGuardian, [][]byte{newGuardianAddress})

	setGuardianFunc, _ := NewSetGuardianFunc(args)
	output, err := setGuardianFunc.ProcessBuiltinFunction(account, nil, vmInput)
	require.Nil(t, err)
	requireSetGuardianVMOutputOk(t, output, vmInput.GasProvided, args.FuncGasCost)

	newGuardian := &Guardian{
		Address:         newGuardianAddress,
		ActivationEpoch: args.BlockChainHook.CurrentEpoch() + args.GuardianActivationEpochs,
	}
	expectedStoredGuardians := &Guardians{Data: []*Guardian{enabledGuardian2, newGuardian}}
	requireAccountHasGuardians(t, account, expectedStoredGuardians)
}

func generateRandomByteArray(size uint32) []byte {
	ret := make([]byte, size)
	_, _ = rand.Read(ret)
	return ret
}

func createSetGuardianFuncMockArgs() SetGuardianArgs {
	return SetGuardianArgs{
		FuncGasCost: 100000,
		Marshaller:  marshallerMock,
		BlockChainHook: &mockvm.BlockChainEpochHookStub{
			CurrentEpochCalled: func() uint32 {
				return 1000
			},
		},
		PubKeyConverter: &mock.PubkeyConverterStub{
			LenCalled: func() int {
				return pubKeyLen
			},
			EncodeCalled: func(pkBytes []byte) string {
				return string(append([]byte("erd1"), pkBytes...))
			},
		},
		SetGuardianEnableEpoch:   0,
		GuardianActivationEpochs: 100,
		EpochNotifier:            &mockvm.EpochNotifierStub{},
	}
}

func getDefaultVmInput(funcName string, args [][]byte) *vmcommon.ContractCallInput {
	return &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  userAddress,
			Arguments:   args,
			CallValue:   big.NewInt(0),
			GasProvided: 500000,
		},
		Function: funcName,
	}
}
