package builtInFunctions

import (
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
)

func TestNewMigrateUserNameFunc(t *testing.T) {
	m, err := NewMigrateUserNameFunc(0, nil, nil)
	require.Equal(t, err, ErrNilDnsAddresses)

	m, err = NewMigrateUserNameFunc(0, make(map[string]struct{}), nil)
	require.Equal(t, err, ErrNilEnableEpochsHandler)

	dnsAddr := []byte("DNS")
	mapDnsAddresses := make(map[string]struct{})
	mapDnsAddresses[string(dnsAddr)] = struct{}{}
	m, err = NewMigrateUserNameFunc(0, mapDnsAddresses, &mock.EnableEpochsHandlerStub{})
	require.Nil(t, err)
	require.False(t, m.IsInterfaceNil())

	m.SetNewGasConfig(nil)
	m.SetNewGasConfig(&vmcommon.GasCost{BuiltInCost: vmcommon.BuiltInCost{SaveUserName: 10}})
	require.Equal(t, m.gasCost, uint64(10))
}

func TestNewDeleteUserNameFunc(t *testing.T) {
	m, err := NewDeleteUserNameFunc(0, nil, nil)
	require.Equal(t, err, ErrNilDnsAddresses)

	m, err = NewDeleteUserNameFunc(0, make(map[string]struct{}), nil)
	require.Equal(t, err, ErrNilEnableEpochsHandler)

	dnsAddr := []byte("DNS")
	mapDnsAddresses := make(map[string]struct{})
	mapDnsAddresses[string(dnsAddr)] = struct{}{}
	m, err = NewDeleteUserNameFunc(0, mapDnsAddresses, &mock.EnableEpochsHandlerStub{})
	require.Nil(t, err)
	require.False(t, m.IsInterfaceNil())

	m.SetNewGasConfig(nil)
	m.SetNewGasConfig(&vmcommon.GasCost{BuiltInCost: vmcommon.BuiltInCost{SaveUserName: 10}})
	require.Equal(t, m.gasCost, uint64(10))
}

func TestMigrateUserName_ProcessBuiltinFunction(t *testing.T) {
	t.Parallel()

	dnsAddr := []byte("DNS")
	mapDnsAddresses := make(map[string]struct{})
	mapDnsAddresses[string(dnsAddr)] = struct{}{}
	m := migrateUserName{
		gasCost:         100,
		mapDnsAddresses: mapDnsAddresses,
	}

	addr := []byte("addr")

	acc := mock.NewUserAccount(addr)
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  dnsAddr,
			GasProvided: 50,
			CallValue:   big.NewInt(0),
		},
	}

	_, err := m.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Equal(t, ErrNotEnoughGas, err)

	_, err = m.ProcessBuiltinFunction(nil, acc, nil)
	require.Equal(t, ErrNilVmInput, err)

	vmInput.GasProvided = 101
	vmInput.CallValue = big.NewInt(10)
	_, err = m.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Equal(t, ErrBuiltInFunctionCalledWithValue, err)

	vmInput.CallValue = big.NewInt(0)
	vmInput.CallerAddr = []byte("just")
	_, err = m.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Equal(t, ErrCallerIsNotTheDNSAddress, err)

	vmInput.CallerAddr = dnsAddr

	newUserName := []byte("afafafafafafafafafafafafafafafaf")
	vmInput.Arguments = [][]byte{newUserName}

	vmOutput, err := m.ProcessBuiltinFunction(nil, nil, vmInput)
	require.Nil(t, err)
	require.Equal(t, 1, len(vmOutput.OutputAccounts))

	_, err = m.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Equal(t, ErrCannotMigrateNilUserName, err)

	acc.Username = []byte("afafafaf")
	_, err = m.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Equal(t, ErrWrongUserNameSplit, err)

	acc.Username = []byte("aaaa.bbb")
	vmInput.Arguments = [][]byte{[]byte("xx.bbb")}
	_, err = m.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Equal(t, ErrUserNamePrefixNotEqual, err)

	vmInput.Arguments = [][]byte{[]byte("aaaa.ddd")}
	_, err = m.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Nil(t, err)
	require.Equal(t, acc.Username, vmInput.Arguments[0])

	m.delete = true
	_, err = m.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Nil(t, err)
	require.Equal(t, len(acc.Username), 0)
}
