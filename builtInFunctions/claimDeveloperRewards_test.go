package builtInFunctions

import (
	"math/big"
	"testing"

	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/require"
)

func TestClaimDeveloperRewards_ProcessBuiltinFunction(t *testing.T) {
	t.Parallel()

	cdr := claimDeveloperRewards{}

	sender := []byte("sender")
	acc := mock.NewUserAccount([]byte("addr12"))

	vmOutput, err := cdr.ProcessBuiltinFunction(nil, acc, nil)
	require.Nil(t, vmOutput)
	require.Equal(t, ErrNilVmInput, err)

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  sender,
			GasProvided: 100,
			CallValue:   big.NewInt(0),
		},
	}
	vmOutput, err = cdr.ProcessBuiltinFunction(nil, nil, vmInput)
	require.Nil(t, err)
	require.NotNil(t, vmOutput)
	require.Equal(t, 0, len(vmOutput.Logs))

	vmOutput, err = cdr.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Nil(t, vmOutput)
	require.Equal(t, ErrOperationNotPermitted, err)

	acc.OwnerAddress = sender
	value := big.NewInt(100)
	acc.AddToDeveloperReward(value)
	vmOutput, err = cdr.ProcessBuiltinFunction(nil, acc, vmInput)
	require.Nil(t, err)
	require.Equal(t, 1, len(vmOutput.OutputAccounts[string(vmInput.CallerAddr)].OutputTransfers))
	require.Equal(t, value, vmOutput.OutputAccounts[string(vmInput.CallerAddr)].OutputTransfers[0].Value)
	require.Equal(t, uint64(0), vmOutput.GasRemaining)
	require.Equal(t, 1, len(vmOutput.Logs))
	require.Equal(t, [][]byte{value.Bytes(), acc.OwnerAddress}, vmOutput.Logs[0].Topics)

	acc.OwnerAddress = sender
	acc.AddToDeveloperReward(value)
	cdr.gasCost = 50
	vmOutput, err = cdr.ProcessBuiltinFunction(acc, acc, vmInput)
	require.Nil(t, err)
	require.Equal(t, vmOutput.GasRemaining, vmInput.GasProvided-cdr.gasCost)
	require.Equal(t, 1, len(vmOutput.Logs))
	require.Equal(t, [][]byte{value.Bytes(), acc.OwnerAddress}, vmOutput.Logs[0].Topics)
}

func TestAddLogEntryForClaimDeveloperRewards(t *testing.T) {
	t.Parallel()

	vmInput := &vmcommon.ContractCallInput{
		Function:      "ClaimDeveloperRewards",
		RecipientAddr: []byte("contract"),
	}

	vmOutput := &vmcommon.VMOutput{}
	value := big.NewInt(42)
	developerAddress := []byte("developer")

	addLogEntryForClaimDeveloperRewards(vmInput, vmOutput, value, developerAddress)

	require.Equal(t, 1, len(vmOutput.Logs))
	require.Equal(t, []byte(vmInput.Function), vmOutput.Logs[0].Identifier)
	require.Equal(t, vmInput.RecipientAddr, vmOutput.Logs[0].Address)
	require.Equal(t, [][]byte{value.Bytes(), developerAddress}, vmOutput.Logs[0].Topics)
}
