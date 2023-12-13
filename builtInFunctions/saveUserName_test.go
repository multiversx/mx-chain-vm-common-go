package builtInFunctions

import (
	"math/big"
	"testing"

	"github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/require"
)

func TestNewSaveUserNameFunc(t *testing.T) {
	m, err := NewSaveUserNameFunc(0, nil, nil, nil)
	require.Equal(t, err, ErrNilDnsAddresses)
	require.Nil(t, m)

	m, err = NewSaveUserNameFunc(0, make(map[string]struct{}), nil, nil)
	require.Equal(t, err, ErrNilDnsAddresses)
	require.Nil(t, m)

	m, err = NewSaveUserNameFunc(0, make(map[string]struct{}), make(map[string]struct{}), nil)
	require.Equal(t, err, ErrNilEnableEpochsHandler)
	require.Nil(t, m)

	dnsAddr := []byte("DNS")
	mapDnsAddresses := make(map[string]struct{})
	mapDnsAddresses[string(dnsAddr)] = struct{}{}
	m, err = NewSaveUserNameFunc(0, mapDnsAddresses, make(map[string]struct{}), &mock.EnableEpochsHandlerStub{})
	require.Nil(t, err)
	require.NotNil(t, m)
	require.Equal(t, len(m.mapDnsAddresses), 1)
	require.Equal(t, len(m.mapDnsV2Addresses), 0)

	m.SetNewGasConfig(nil)
	m.SetNewGasConfig(&vmcommon.GasCost{BuiltInCost: vmcommon.BuiltInCost{SaveUserName: 10}})
	require.Equal(t, m.gasCost, uint64(10))
}

func TestSaveUserName_IsInterfaceNil(t *testing.T) {
	t.Parallel()

	var m *saveUserName
	require.True(t, m.IsInterfaceNil())

	dnsAddr := []byte("DNS")
	mapDnsAddresses := make(map[string]struct{})
	mapDnsAddresses[string(dnsAddr)] = struct{}{}
	m, _ = NewSaveUserNameFunc(0, mapDnsAddresses, make(map[string]struct{}), &mock.EnableEpochsHandlerStub{})
	require.False(t, m.IsInterfaceNil())
}

func TestSaveUserName_ProcessBuiltinFunction(t *testing.T) {
	t.Parallel()

	dnsAddr := []byte("DNS")
	mapDnsAddresses := make(map[string]struct{})
	mapDnsAddresses[string(dnsAddr)] = struct{}{}
	coa := saveUserName{
		gasCost:           1,
		mapDnsAddresses:   mapDnsAddresses,
		mapDnsV2Addresses: make(map[string]struct{}),
		isChangeEnabled: func() bool {
			return false
		},
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

	_, err := coa.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Equal(t, ErrInvalidArguments, err)

	vmInput.GasProvided = 0
	_, err = coa.ProcessBuiltinFunction(acc, acc, vmInput)
	require.Equal(t, ErrNotEnoughGas, err)

	vmInput.GasProvided = 50
	newUserName := []byte("afafafafafafafafafafafafafafafaf")
	vmInput.Arguments = [][]byte{newUserName}

	_, err = coa.ProcessBuiltinFunction(nil, acc, nil)
	require.Equal(t, ErrNilVmInput, err)

	vmOutput, err := coa.ProcessBuiltinFunction(nil, nil, vmInput)
	require.Nil(t, err)
	require.Equal(t, 1, len(vmOutput.OutputAccounts))

	_, err = coa.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Nil(t, err)
	require.Equal(t, acc.GetUserName(), vmInput.Arguments[0])

	_, err = coa.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Equal(t, ErrUserNameChangeIsDisabled, err)

	coa.isChangeEnabled = func() bool {
		return true
	}

	_, err = coa.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Equal(t, err, ErrCallerIsNotTheDNSAddress)

	dnsAddrV2 := []byte("dnsV2")
	coa.mapDnsV2Addresses[string(dnsAddrV2)] = struct{}{}
	vmInput.CallerAddr = dnsAddrV2
	vmInput.Arguments[0] = []byte("abcdabcd")
	vmOutput, err = coa.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Nil(t, err)
	require.Equal(t, acc.GetUserName(), vmInput.Arguments[0])
	require.Equal(t, vmOutput.GasRemaining, vmInput.GasProvided)

	vmOutput, err = coa.ProcessBuiltinFunction(acc, acc, vmInput)
	require.Nil(t, err)
	require.Equal(t, acc.GetUserName(), vmInput.Arguments[0])
	require.Equal(t, vmOutput.GasRemaining, vmInput.GasProvided-coa.gasCost)

}
