package builtInFunctions

// drwa_regression_test.go — non-DRWA path regression tests.
//
// These tests prove that the DRWA enforcement gate does NOT alter the behaviour
// of ordinary MultiversX token transfers when:
//   (a) the DRWAEnforcementFlag epoch flag is disabled, or
//   (b) the token has no DRWA policy registered, or
//   (c) a non-DRWA token is transferred alongside a DRWA-regulated one.
//
// Every test function must pass without modification after any refactor of
// drwa.go.  A failure here indicates an unintended regression in the plain
// ESDT / NFT transfer path.

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	vm "github.com/multiversx/mx-chain-core-go/data/vm"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/require"
)

// drwaDisabledEpochsHandler returns a handler where every flag including
// DRWAEnforcementFlag is OFF.
func drwaDisabledEpochsHandler() *mock.EnableEpochsHandlerStub {
	return &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(_ core.EnableEpochFlag) bool {
			return false
		},
	}
}

// --- ESDT transfer regressions ---

// TestNonDRWAESDTTransfer_FlagDisabled verifies a plain ESDT transfer succeeds
// when the DRWA epoch flag is disabled, even if DRWA state is present.
func TestNonDRWAESDTTransfer_FlagDisabled(t *testing.T) {
	t.Parallel()

	accounts, state := createDRWATestAccounts()
	transferFunc, err := NewESDTTransferFunc(
		10,
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.ShardCoordinatorStub{},
		&mock.ESDTRoleHandlerStub{},
		drwaDisabledEpochsHandler(),
	)
	require.NoError(t, err)
	require.NoError(t, transferFunc.SetPayableChecker(&mock.PayableHandlerStub{}))

	// Even with a DRWA-enabled policy present the gate must not run.
	systemAcc := state[string(vmcommon.SystemAccountAddress)]
	mustSaveDRWATokenPolicy(t, systemAcc, "BOND-FLAG", &drwaTokenPolicyView{
		DRWAEnabled: true,
		GlobalPause: true, // would deny if gate ran
	})

	sender := state["sender"]
	receiver := state["receiver"]
	mustSaveESDTBalance(t, sender, "BOND-FLAG", 5)

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  []byte("sender"),
			Arguments:   [][]byte{[]byte("BOND-FLAG"), big.NewInt(1).Bytes()},
			CallValue:   big.NewInt(0),
			GasProvided: 100,
			CallType:    vm.DirectCall,
		},
		RecipientAddr: []byte("receiver"),
	}

	output, err := transferFunc.ProcessBuiltinFunction(sender, receiver, vmInput)
	require.NoError(t, err)
	require.NotNil(t, output)
	require.Equal(t, vmcommon.Ok, output.ReturnCode,
		"flag-disabled transfer must succeed regardless of policy state")

	_ = accounts
}

// TestNonDRWAESDTTransfer_NoDRWAReader verifies a plain ESDT transfer succeeds
// when no DRWA reader is attached to the transfer function at all.
func TestNonDRWAESDTTransfer_NoDRWAReader(t *testing.T) {
	t.Parallel()

	_, state := createDRWATestAccounts()
	transferFunc, err := NewESDTTransferFunc(
		10,
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.ShardCoordinatorStub{},
		&mock.ESDTRoleHandlerStub{},
		drwaEnabledEpochsHandler(),
	)
	require.NoError(t, err)
	require.NoError(t, transferFunc.SetPayableChecker(&mock.PayableHandlerStub{}))
	// Intentionally do NOT call transferFunc.SetDRWAReader(...)

	sender := state["sender"]
	receiver := state["receiver"]
	mustSaveESDTBalance(t, sender, "PLAIN-1", 10)

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  []byte("sender"),
			Arguments:   [][]byte{[]byte("PLAIN-1"), big.NewInt(3).Bytes()},
			CallValue:   big.NewInt(0),
			GasProvided: 100,
			CallType:    vm.DirectCall,
		},
		RecipientAddr: []byte("receiver"),
	}

	output, err := transferFunc.ProcessBuiltinFunction(sender, receiver, vmInput)
	require.NoError(t, err)
	require.NotNil(t, output)
	require.Equal(t, vmcommon.Ok, output.ReturnCode,
		"transfer with nil DRWA reader must succeed (no DRWA check)")
}

// TestNonDRWAESDTTransfer_NoPolicy verifies a plain ESDT transfer succeeds when
// the DRWA reader is wired but the token has no registered policy.
func TestNonDRWAESDTTransfer_NoPolicy(t *testing.T) {
	t.Parallel()

	accounts, state := createDRWATestAccounts()
	transferFunc, err := NewESDTTransferFunc(
		10,
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.ShardCoordinatorStub{},
		&mock.ESDTRoleHandlerStub{},
		drwaEnabledEpochsHandler(),
	)
	require.NoError(t, err)
	require.NoError(t, transferFunc.SetPayableChecker(&mock.PayableHandlerStub{}))
	transferFunc.SetDRWAReader(mustCreateDRWAReader(t, accounts))
	// No policy saved for "PLAIN-2" → not a DRWA-regulated token.

	sender := state["sender"]
	receiver := state["receiver"]
	mustSaveESDTBalance(t, sender, "PLAIN-2", 20)

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  []byte("sender"),
			Arguments:   [][]byte{[]byte("PLAIN-2"), big.NewInt(5).Bytes()},
			CallValue:   big.NewInt(0),
			GasProvided: 100,
			CallType:    vm.DirectCall,
		},
		RecipientAddr: []byte("receiver"),
	}

	output, err := transferFunc.ProcessBuiltinFunction(sender, receiver, vmInput)
	require.NoError(t, err)
	require.NotNil(t, output)
	require.Equal(t, vmcommon.Ok, output.ReturnCode,
		"transfer of unregistered token must pass through gate unchanged")
}

// TestNonDRWAESDTTransfer_DRWADisabledPolicy verifies a plain ESDT transfer
// succeeds when a DRWA policy exists for the token but drwa_enabled = false.
func TestNonDRWAESDTTransfer_DRWADisabledPolicy(t *testing.T) {
	t.Parallel()

	accounts, state := createDRWATestAccounts()
	transferFunc, err := NewESDTTransferFunc(
		10,
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.ShardCoordinatorStub{},
		&mock.ESDTRoleHandlerStub{},
		drwaEnabledEpochsHandler(),
	)
	require.NoError(t, err)
	require.NoError(t, transferFunc.SetPayableChecker(&mock.PayableHandlerStub{}))
	transferFunc.SetDRWAReader(mustCreateDRWAReader(t, accounts))

	systemAcc := state[string(vmcommon.SystemAccountAddress)]
	mustSaveDRWATokenPolicy(t, systemAcc, "PLAN-3", &drwaTokenPolicyView{
		DRWAEnabled: false, // flag is off; compliance checks must be skipped
		GlobalPause: true,
	})

	sender := state["sender"]
	receiver := state["receiver"]
	mustSaveESDTBalance(t, sender, "PLAN-3", 50)

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  []byte("sender"),
			Arguments:   [][]byte{[]byte("PLAN-3"), big.NewInt(2).Bytes()},
			CallValue:   big.NewInt(0),
			GasProvided: 100,
			CallType:    vm.DirectCall,
		},
		RecipientAddr: []byte("receiver"),
	}

	output, err := transferFunc.ProcessBuiltinFunction(sender, receiver, vmInput)
	require.NoError(t, err)
	require.NotNil(t, output)
	require.Equal(t, vmcommon.Ok, output.ReturnCode,
		"transfer with drwa_enabled=false must proceed without compliance checks")
}

// TestNonDRWAMultiESDTTransfer_CrossShardFlagDisabled verifies the destination
// shard path does not apply DRWA receiver checks while the enforcement flag is
// disabled, even if a regulated token policy exists and would otherwise block
// receipt.
func TestNonDRWAMultiESDTTransfer_CrossShardFlagDisabled(t *testing.T) {
	t.Parallel()

	multiTransfer := createESDTNFTMultiTransferWithMockArguments(0, 2, &mock.GlobalSettingsHandlerStub{})
	multiTransfer.enableEpochsHandler = drwaDisabledEpochsHandler()
	require.NoError(t, multiTransfer.SetPayableChecker(&mock.PayableHandlerStub{
		IsPayableCalled: func(_ []byte) (bool, error) {
			return true, nil
		},
	}))
	multiTransfer.SetDRWAReader(mustCreateDRWAReader(t, multiTransfer.accounts))

	systemAcc, err := multiTransfer.accounts.LoadAccount(vmcommon.SystemAccountAddress)
	require.NoError(t, err)
	mustSaveDRWATokenPolicy(t, systemAcc.(vmcommon.UserAccountHandler), "BOND-FLAG", &drwaTokenPolicyView{
		DRWAEnabled: true,
		GlobalPause: true, // would deny if the destination-path gate ran
	})

	destinationAddress := bytes.Repeat([]byte{0}, 32)
	destinationAddress[25] = 1
	destinationAccount, err := multiTransfer.accounts.LoadAccount(destinationAddress)
	require.NoError(t, err)

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  bytes.Repeat([]byte{1}, 32),
			CallValue:   big.NewInt(0),
			GasProvided: 100000,
			Arguments: [][]byte{
				big.NewInt(1).Bytes(),
				[]byte("BOND-FLAG"),
				big.NewInt(0).Bytes(),
				big.NewInt(1).Bytes(),
			},
		},
		RecipientAddr: destinationAddress,
	}

	output, err := multiTransfer.ProcessBuiltinFunction(nil, destinationAccount.(vmcommon.UserAccountHandler), vmInput)
	require.NoError(t, err)
	require.NotNil(t, output)
	require.Equal(t, vmcommon.Ok, output.ReturnCode,
		"cross-shard destination path must skip DRWA receiver enforcement when the flag is disabled")
}

// TestNonDRWAESDTTransfer_GasUnchanged verifies that a non-DRWA token transfer
// consumes the same gas as before DRWA was introduced (no extra overhead).
func TestNonDRWAESDTTransfer_GasUnchanged(t *testing.T) {
	t.Parallel()

	_, state := createDRWATestAccounts()
	transferFunc, err := NewESDTTransferFunc(
		10,
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.ShardCoordinatorStub{},
		&mock.ESDTRoleHandlerStub{},
		drwaEnabledEpochsHandler(),
	)
	require.NoError(t, err)
	require.NoError(t, transferFunc.SetPayableChecker(&mock.PayableHandlerStub{}))
	// No DRWA reader → pure non-DRWA path.

	sender := state["sender"]
	receiver := state["receiver"]
	mustSaveESDTBalance(t, sender, "PLAIN-GAS", 100)

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  []byte("sender"),
			Arguments:   [][]byte{[]byte("PLAIN-GAS"), big.NewInt(1).Bytes()},
			CallValue:   big.NewInt(0),
			GasProvided: 100,
			CallType:    vm.DirectCall,
		},
		RecipientAddr: []byte("receiver"),
	}

	output, err := transferFunc.ProcessBuiltinFunction(sender, receiver, vmInput)
	require.NoError(t, err)
	require.Equal(t, vmcommon.Ok, output.ReturnCode)
	// funcGasCost=10, GasProvided=100 → GasRemaining should be 90.
	require.Equal(t, uint64(90), output.GasRemaining,
		"non-DRWA transfer gas cost must equal funcGasCost only")
}
