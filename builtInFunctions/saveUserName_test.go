package builtInFunctions

import (
	"math/big"
	"testing"

	"github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/require"
)

func TestNewSaveUserNameFunc(t *testing.T) {
	m, err := NewSaveUserNameFunc(0, nil, nil)
	require.Equal(t, err, ErrNilDnsAddresses)

	m, err = NewSaveUserNameFunc(0, make(map[string]struct{}), nil)
	require.Equal(t, err, ErrNilEnableEpochsHandler)

	dnsAddr := []byte("DNS")
	mapDnsAddresses := make(map[string]struct{})
	mapDnsAddresses[string(dnsAddr)] = struct{}{}
	m, err = NewSaveUserNameFunc(0, mapDnsAddresses, &mock.EnableEpochsHandlerStub{})
	require.Nil(t, err)
	require.False(t, m.IsInterfaceNil())

	m.SetNewGasConfig(nil)
	m.SetNewGasConfig(&vmcommon.GasCost{BuiltInCost: vmcommon.BuiltInCost{SaveUserName: 10}})
	require.Equal(t, m.gasCost, uint64(10))
}

func TestSaveUserName_ProcessBuiltinFunction(t *testing.T) {
	t.Parallel()

	dnsAddr := []byte("DNS")
	mapDnsAddresses := make(map[string]struct{})
	mapDnsAddresses[string(dnsAddr)] = struct{}{}
	coa := saveUserName{
		gasCost:         1,
		mapDnsAddresses: mapDnsAddresses,
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

	newUserName := []byte("afafafafafafafafafafafafafafafaf")
	vmInput.Arguments = [][]byte{newUserName}

	_, err = coa.ProcessBuiltinFunction(nil, acc, nil)
	require.Equal(t, ErrNilVmInput, err)

	vmOutput, err := coa.ProcessBuiltinFunction(nil, nil, vmInput)
	require.Nil(t, err)
	require.Equal(t, 1, len(vmOutput.OutputAccounts))

	_, err = coa.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Nil(t, err)

	_, err = coa.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Equal(t, ErrUserNameChangeIsDisabled, err)

	coa.isChangeEnabled = func() bool {
		return true
	}

	_, err = coa.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Nil(t, err)
}
