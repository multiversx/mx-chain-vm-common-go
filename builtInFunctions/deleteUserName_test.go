package builtInFunctions

import (
	"math/big"
	"testing"

	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/require"
)

func TestNewDeleteUserNameFunc(t *testing.T) {
	d, err := NewDeleteUserNameFunc(0, nil, nil)
	require.Equal(t, err, ErrNilDnsAddresses)
	require.Nil(t, d)

	d, err = NewDeleteUserNameFunc(0, make(map[string]struct{}), nil)
	require.Equal(t, err, ErrNilEnableEpochsHandler)
	require.Nil(t, d)

	dnsAddr := []byte("DNS")
	mapDnsAddresses := make(map[string]struct{})
	mapDnsAddresses[string(dnsAddr)] = struct{}{}
	d, err = NewDeleteUserNameFunc(0, mapDnsAddresses, &mock.EnableEpochsHandlerStub{})
	require.Nil(t, err)
	require.NotNil(t, d)
	require.False(t, d.IsInterfaceNil())

	d.SetNewGasConfig(nil)
	d.SetNewGasConfig(&vmcommon.GasCost{BuiltInCost: vmcommon.BuiltInCost{SaveUserName: 10}})
	require.Equal(t, d.gasCost, uint64(10))
}

func TestDeleteUserName_IsInterfaceNil(t *testing.T) {
	t.Parallel()

	var d *deleteUserName
	require.True(t, d.IsInterfaceNil())

	dnsAddr := []byte("DNS")
	mapDnsAddresses := make(map[string]struct{})
	mapDnsAddresses[string(dnsAddr)] = struct{}{}
	d, _ = NewDeleteUserNameFunc(0, mapDnsAddresses, &mock.EnableEpochsHandlerStub{})
	require.False(t, d.IsInterfaceNil())
}

func TestDeleteUserName_ProcessBuiltinFunction(t *testing.T) {
	t.Parallel()

	dnsAddr := []byte("DNS")
	mapDnsAddresses := make(map[string]struct{})
	mapDnsAddresses[string(dnsAddr)] = struct{}{}
	d := deleteUserName{
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

	_, err := d.ProcessBuiltinFunction(acc, acc, vmInput)
	require.Equal(t, ErrNotEnoughGas, err)

	_, err = d.ProcessBuiltinFunction(nil, acc, nil)
	require.Equal(t, ErrNilVmInput, err)

	vmInput.GasProvided = 101
	vmInput.CallValue = big.NewInt(10)
	_, err = d.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Equal(t, ErrBuiltInFunctionCalledWithValue, err)

	vmInput.CallValue = big.NewInt(0)
	vmInput.CallerAddr = []byte("just")
	_, err = d.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Equal(t, ErrCallerIsNotTheDNSAddress, err)

	vmInput.CallerAddr = dnsAddr

	newUserName := []byte("afafafafafafafafafafafafafafafaf")
	vmInput.Arguments = [][]byte{newUserName}

	_, err = d.ProcessBuiltinFunction(nil, nil, vmInput)
	require.Equal(t, err, ErrInvalidArguments)

	vmInput.Arguments = make([][]byte, 0)
	_, err = d.ProcessBuiltinFunction(acc, acc, vmInput)
	require.Nil(t, err)
	require.Equal(t, len(acc.Username), 0)

	vmOutput, err := d.ProcessBuiltinFunction(acc, nil, vmInput)
	require.Nil(t, err)
	require.Equal(t, len(vmOutput.OutputAccounts), 1)
	require.Equal(t, vmOutput.OutputAccounts[string(vmInput.RecipientAddr)].OutputTransfers[0].GasLimit, vmInput.GasProvided-d.gasCost)

	vmInput.GasProvided = 0
	_, err = d.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Nil(t, err)
	require.Equal(t, len(acc.Username), 0)
	require.Equal(t, vmOutput.GasRemaining, vmInput.GasProvided)
}
