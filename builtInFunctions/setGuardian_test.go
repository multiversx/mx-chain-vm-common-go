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

func TestFreezeAccount_ExecuteSetGuardianCase1(t *testing.T) {
	userAccount := mockvm.NewUserAccount([]byte("user address"))
	blockChainHook := &mockvm.BlockChainHookStub{
		GetUserAccountCalled: func(address []byte) (vmcommon.UserAccountHandler, error) {
			return userAccount, nil
		},
	}

	args := createAccountFreezerMockArgs()
	args.BlockChainHook = blockChainHook
	accountFreezer, _ := NewFreezeAccountSmartContract(args)

	guardianAddress := generateRandomByteArray(32)
	vmInput := getDefaultVmInputForFunc(setGuardian, [][]byte{guardianAddress})

	_, err := accountFreezer.ProcessBuiltinFunction(nil, nil, vmInput)
	require.Nil(t, err)

	key := append([]byte(core.ElrondProtectedKeyPrefix), []byte(GuardiansKey)...)
	marshalledData, _ := userAccount.AccountDataHandler().RetrieveValue(key)
	storedGuardian := &Guardians{}
	_ = args.Marshaller.Unmarshal(storedGuardian, marshalledData)

	expectedStoredGuardian := &Guardian{
		Address:         guardianAddress,
		ActivationEpoch: blockChainHook.CurrentEpoch() + args.GuardianActivationEpochs,
	}
	require.Len(t, storedGuardian.Data, 1)
	require.Equal(t, storedGuardian.Data[0], expectedStoredGuardian)
}

func TestFreezeAccount_ExecuteSetGuardianCase2(t *testing.T) {
	userAccount := mockvm.NewUserAccount([]byte("user address"))
	key := append([]byte(core.ElrondProtectedKeyPrefix), []byte(GuardiansKey)...)
	guardianAddress := generateRandomByteArray(32)

	args := createAccountFreezerMockArgs()

	pendingGuardian := Guardians{
		Data: []*Guardian{
			{
				Address:         guardianAddress,
				ActivationEpoch: args.BlockChainHook.CurrentEpoch() + args.GuardianActivationEpochs - 1,
			},
		},
	}
	marshalledPendingGuardian, _ := args.Marshaller.Marshal(pendingGuardian)
	_ = userAccount.SaveKeyValue(key, marshalledPendingGuardian)

	blockChainHook := &mockvm.BlockChainHookStub{
		GetUserAccountCalled: func(address []byte) (vmcommon.UserAccountHandler, error) {
			return userAccount, nil
		},
	}

	args.BlockChainHook = blockChainHook
	accountFreezer, _ := NewFreezeAccountSmartContract(args)

	vmInput := getDefaultVmInputForFunc(setGuardian, [][]byte{guardianAddress})

	output, err := accountFreezer.ProcessBuiltinFunction(nil, nil, vmInput)
	require.Nil(t, output)
	require.True(t, strings.Contains(err.Error(), "owner already has one guardian"))

	marshalledData, _ := userAccount.AccountDataHandler().RetrieveValue(key)
	require.Equal(t, marshalledPendingGuardian, marshalledData)
}

func TestFreezeAccount_ExecuteSetGuardianCase3(t *testing.T) {
	userAccount := mockvm.NewUserAccount([]byte("user address"))
	key := append([]byte(core.ElrondProtectedKeyPrefix), []byte(GuardiansKey)...)
	guardianAddress := generateRandomByteArray(32)
	enabledGuardianAddress := generateRandomByteArray(32)

	args := createAccountFreezerMockArgs()

	enabledGuardian := Guardians{
		Data: []*Guardian{
			{
				Address:         enabledGuardianAddress,
				ActivationEpoch: args.BlockChainHook.CurrentEpoch() + args.GuardianActivationEpochs + 1,
			},
		},
	}
	marshalledEnabledGuardian, _ := args.Marshaller.Marshal(enabledGuardian)
	_ = userAccount.SaveKeyValue(key, marshalledEnabledGuardian)

	blockChainHook := &mockvm.BlockChainHookStub{
		GetUserAccountCalled: func(address []byte) (vmcommon.UserAccountHandler, error) {
			return userAccount, nil
		},
	}

	args.BlockChainHook = blockChainHook
	accountFreezer, _ := NewFreezeAccountSmartContract(args)

	vmInput := getDefaultVmInputForFunc(setGuardian, [][]byte{guardianAddress})

	_, err := accountFreezer.ProcessBuiltinFunction(nil, nil, vmInput)
	require.Nil(t, err)

	marshalledData, _ := userAccount.AccountDataHandler().RetrieveValue(key)
	guardians := Guardians{}
	_ = args.Marshaller.Unmarshal(&guardians, marshalledData)
	expectedStoredGuardians := Guardians{
		Data: []*Guardian{
			{
				Address:         enabledGuardianAddress,
				ActivationEpoch: args.BlockChainHook.CurrentEpoch() + args.GuardianActivationEpochs + 1,
			},
			{
				Address:         guardianAddress,
				ActivationEpoch: args.BlockChainHook.CurrentEpoch() + args.GuardianActivationEpochs + args.SetGuardianEnableEpoch,
			},
		},
	}

	require.Equal(t, expectedStoredGuardians, guardians)
}

func TestFreezeAccount_ExecuteSetGuardianCase4(t *testing.T) {
	userAccount := mockvm.NewUserAccount([]byte("user address"))
	key := append([]byte(core.ElrondProtectedKeyPrefix), []byte(GuardiansKey)...)
	guardianAddress := generateRandomByteArray(32)
	enabledGuardianAddress1 := generateRandomByteArray(32)
	enabledGuardianAddress2 := generateRandomByteArray(32)

	args := createAccountFreezerMockArgs()

	storedGuardians := Guardians{
		Data: []*Guardian{
			{
				Address:         enabledGuardianAddress1,
				ActivationEpoch: args.BlockChainHook.CurrentEpoch() + args.GuardianActivationEpochs + 1,
			},
			{
				Address:         enabledGuardianAddress2,
				ActivationEpoch: args.BlockChainHook.CurrentEpoch() + args.GuardianActivationEpochs - 1,
			},
		},
	}
	marshalledStoredGuardians, _ := args.Marshaller.Marshal(storedGuardians)
	_ = userAccount.SaveKeyValue(key, marshalledStoredGuardians)

	blockChainHook := &mockvm.BlockChainHookStub{
		GetUserAccountCalled: func(address []byte) (vmcommon.UserAccountHandler, error) {
			return userAccount, nil
		},
	}

	args.BlockChainHook = blockChainHook
	accountFreezer, _ := NewFreezeAccountSmartContract(args)

	vmInput := getDefaultVmInputForFunc(setGuardian, [][]byte{guardianAddress})

	output, err := accountFreezer.ProcessBuiltinFunction(nil, nil, vmInput)
	require.True(t, strings.Contains(err.Error(), "owner already has one guardian"))
	require.Nil(t, output)

	marshalledData, _ := userAccount.AccountDataHandler().RetrieveValue(key)
	require.Equal(t, marshalledStoredGuardians, marshalledData)
}

func TestFreezeAccount_ExecuteSetGuardianCase5(t *testing.T) {
	userAccount := mockvm.NewUserAccount([]byte("user address"))
	key := append([]byte(core.ElrondProtectedKeyPrefix), []byte(GuardiansKey)...)
	guardianAddress := generateRandomByteArray(32)
	enabledGuardianAddress1 := generateRandomByteArray(32)
	enabledGuardianAddress2 := generateRandomByteArray(32)

	args := createAccountFreezerMockArgs()

	storedGuardians := Guardians{
		Data: []*Guardian{
			{
				Address:         enabledGuardianAddress1,
				ActivationEpoch: args.BlockChainHook.CurrentEpoch() + args.GuardianActivationEpochs + 1,
			},
			{
				Address:         enabledGuardianAddress2,
				ActivationEpoch: args.BlockChainHook.CurrentEpoch() + args.GuardianActivationEpochs + 1,
			},
		},
	}
	marshalledStoredGuardians, _ := args.Marshaller.Marshal(storedGuardians)
	_ = userAccount.SaveKeyValue(key, marshalledStoredGuardians)

	blockChainHook := &mockvm.BlockChainHookStub{
		GetUserAccountCalled: func(address []byte) (vmcommon.UserAccountHandler, error) {
			return userAccount, nil
		},
	}

	args.BlockChainHook = blockChainHook
	accountFreezer, _ := NewFreezeAccountSmartContract(args)

	vmInput := getDefaultVmInputForFunc(setGuardian, [][]byte{guardianAddress})

	output, err := accountFreezer.ProcessBuiltinFunction(nil, nil, vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, output.ReturnCode)

	marshalledData, _ := userAccount.AccountDataHandler().RetrieveValue(key)
	guardians := Guardians{}
	_ = args.Marshaller.Unmarshal(&guardians, marshalledData)
	expectedStoredGuardians := Guardians{
		Data: []*Guardian{
			{
				Address:         enabledGuardianAddress2,
				ActivationEpoch: args.BlockChainHook.CurrentEpoch() + args.GuardianActivationEpochs + 1,
			},
			{
				Address:         guardianAddress,
				ActivationEpoch: args.BlockChainHook.CurrentEpoch() + args.GuardianActivationEpochs + args.SetGuardianEnableEpoch,
			},
		},
	}

	require.Equal(t, expectedStoredGuardians, guardians)
}

// TODO: Remove this from all duplicate places
func generateRandomByteArray(size int) []byte {
	r := make([]byte, size)
	_, _ = rand.Read(r)
	return r
}

func createAccountFreezerMockArgs() ArgsFreezeAccountSC {
	return ArgsFreezeAccountSC{
		GasCost:        vmcommon.GasCost{},
		Marshaller:     &mockvm.MarshalizerMock{},
		BlockChainHook: &mockvm.BlockChainHookStub{},
		PubKeyConverter: &mock.PubkeyConverterStub{
			LenCalled: func() int {
				return 32
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

func getDefaultVmInputForFunc(funcName string, args [][]byte) *vmcommon.ContractCallInput {
	return &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:     []byte("owner"),
			Arguments:      args,
			CallValue:      big.NewInt(0),
			CallType:       0,
			GasPrice:       0,
			GasProvided:    0,
			OriginalTxHash: nil,
			CurrentTxHash:  nil,
		},
		RecipientAddr: []byte("addr"),
		Function:      funcName,
	}
}
