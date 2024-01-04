package builtInFunctions

import (
	"math/big"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data/vm"
	"github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/require"
)

func TestNewChangeOwnerNilEnableEpochsHandler(t *testing.T) {
	t.Parallel()

	gasCost := uint64(100)
	coa, err := NewChangeOwnerAddressFunc(gasCost, nil)
	require.Nil(t, coa)
	require.Equal(t, ErrNilEnableEpochsHandler, err)
}

func TestNewChangeOwnerAddressFunc(t *testing.T) {
	t.Parallel()

	gasCost := uint64(100)
	coa, err := NewChangeOwnerAddressFunc(gasCost, &mock.EnableEpochsHandlerStub{})
	require.Nil(t, err)
	require.False(t, check.IfNil(coa))
	require.Equal(t, gasCost, coa.gasCost)
	require.True(t, coa.IsActive())
}

func TestChangeOwnerAddress_SetNewGasConfig(t *testing.T) {
	t.Parallel()

	coa, _ := NewChangeOwnerAddressFunc(100, &mock.EnableEpochsHandlerStub{})

	newCost := uint64(37)
	expectedGasConfig := &vmcommon.GasCost{BuiltInCost: vmcommon.BuiltInCost{ChangeOwnerAddress: newCost}}
	coa.SetNewGasConfig(expectedGasConfig)

	require.Equal(t, newCost, coa.gasCost)
}

func TestChangeOwnerAddress_ProcessBuiltinFunction(t *testing.T) {
	t.Parallel()

	coa := changeOwnerAddress{
		enableEpochsHandler: &mock.EnableEpochsHandlerStub{},
	}

	owner := []byte("send")
	addr := []byte("addr")

	acc := mock.NewUserAccount(addr)
	vmInput := &vmcommon.ContractCallInput{
		Function: core.BuiltInFunctionChangeOwnerAddress,
		VMInput:  vmcommon.VMInput{CallerAddr: owner, CallValue: big.NewInt(0)},
	}

	_, err := coa.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Equal(t, ErrInvalidArguments, err)

	newAddr := []byte("0000")
	vmInput.Arguments = [][]byte{newAddr}
	_, err = coa.ProcessBuiltinFunction(nil, acc, nil)
	require.Equal(t, ErrNilVmInput, err)

	_, err = coa.ProcessBuiltinFunction(nil, nil, vmInput)
	require.Nil(t, err)

	var vmOutput *vmcommon.VMOutput
	acc.OwnerAddress = owner
	vmInput.GasProvided = 10
	vmOutput, err = coa.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Nil(t, err)
	require.Equal(t, vmOutput.GasRemaining, uint64(0))

	contractAddress := []byte("contract")
	vmInput.RecipientAddr = contractAddress
	coa.gasCost = 1
	vmInput.GasProvided = 10
	acc.OwnerAddress = owner
	vmOutput, err = coa.ProcessBuiltinFunction(acc, acc, vmInput)
	require.Nil(t, err)
	require.Equal(t, vmOutput.GasRemaining, vmInput.GasProvided-coa.gasCost)

	require.Equal(t, &vmcommon.LogEntry{
		Identifier: []byte(core.BuiltInFunctionChangeOwnerAddress),
		Address:    contractAddress,
		Topics:     [][]byte{newAddr},
	}, vmOutput.Logs[0])
}

func TestProcessBuiltInFunctionCallThroughSC(t *testing.T) {
	t.Parallel()

	coa := changeOwnerAddress{
		enableEpochsHandler: &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return flag == IsChangeOwnerAddressCrossShardThroughSCFlag
			},
		},
	}

	owner := []byte("00000000000")
	addr := []byte("addraddradd")
	rcvAddr := []byte("contract")

	acc := mock.NewUserAccount(addr)
	acc.OwnerAddress = owner
	vmInput := &vmcommon.ContractCallInput{
		Function:      core.BuiltInFunctionChangeOwnerAddress,
		RecipientAddr: rcvAddr,
		VMInput: vmcommon.VMInput{
			CallerAddr: make([]byte, 11),
			CallValue:  big.NewInt(0),
			Arguments:  [][]byte{[]byte("00000000000")},
		},
	}

	vmOutput, err := coa.ProcessBuiltinFunction(acc, nil, vmInput)
	require.Nil(t, err)
	require.NotNil(t, vmOutput)
	require.Equal(t, 1, len(vmOutput.OutputAccounts))

	outputTransfer := vmOutput.OutputAccounts[string(rcvAddr)].OutputTransfers[0]
	require.Equal(t, []byte("ChangeOwnerAddress@3030303030303030303030"), outputTransfer.Data)
	require.Equal(t, vm.DirectCall, outputTransfer.CallType)
}
