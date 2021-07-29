package builtInFunctions

import (
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/mock"
	"github.com/stretchr/testify/require"
)

func TestNewChangeOwnerAddressFunc(t *testing.T) {
	t.Parallel()

	gasCost := uint64(100)
	coa := NewChangeOwnerAddressFunc(gasCost)
	require.False(t, check.IfNil(coa))
	require.Equal(t, gasCost, coa.gasCost)
	require.True(t, coa.IsActive())
}

func TestChangeOwnerAddress_SetNewGasConfig(t *testing.T) {
	t.Parallel()

	coa := NewChangeOwnerAddressFunc(100)

	newCost := uint64(37)
	expectedGasConfig := &vmcommon.GasCost{BuiltInCost: vmcommon.BuiltInCost{ChangeOwnerAddress: newCost}}
	coa.SetNewGasConfig(expectedGasConfig)

	require.Equal(t, newCost, coa.gasCost)
}

func TestChangeOwnerAddress_ProcessBuiltinFunction(t *testing.T) {
	t.Parallel()

	coa := changeOwnerAddress{}

	owner := []byte("send")
	addr := []byte("addr")

	acc := mock.NewUserAccount(addr)
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{CallerAddr: owner, CallValue: big.NewInt(0)},
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

	coa.gasCost = 1
	vmInput.GasProvided = 10
	acc.OwnerAddress = owner
	vmOutput, err = coa.ProcessBuiltinFunction(acc, acc, vmInput)
	require.Nil(t, err)
	require.Equal(t, vmOutput.GasRemaining, vmInput.GasProvided-coa.gasCost)
}
