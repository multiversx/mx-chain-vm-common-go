package builtInFunctions

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/data/esdt"
	vm "github.com/multiversx/mx-chain-core-go/data/vm"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
)

func BenchmarkDRWASenderTransfer_Allowed(b *testing.B) {
	accounts, state := createDRWATestAccounts()
	reader, err := newDRWAAccountsReader(accounts)
	if err != nil {
		b.Fatalf("new reader: %v", err)
	}
	systemAcc := state[string(vmcommon.SystemAccountAddress)]
	mustSaveDRWATokenPolicyForBenchmark(b, systemAcc, "CARBON-123", &drwaTokenPolicyView{
		DRWAEnabled: true,
	})
	mustSaveDRWAHolderForBenchmark(b, state["sender"], "CARBON-123", "sender", &drwaHolderMirrorView{
		KYCStatus: "approved",
		AMLStatus: "approved",
	})

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := checkDRWASenderTransfer(reader, []byte("CARBON-123"), []byte("sender"), state["sender"], 10); err != nil {
			b.Fatalf("sender check: %v", err)
		}
	}
}

func BenchmarkESDTTransfer_ProcessBuiltinFunction_DRWAAllowed(b *testing.B) {
	accounts, state := createDRWATestAccounts()
	transferFunc, _ := NewESDTTransferFunc(
		10,
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.ShardCoordinatorStub{},
		&mock.ESDTRoleHandlerStub{},
		drwaEnabledEpochsHandler(),
	)
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})
	reader, err := newDRWAAccountsReader(accounts)
	if err != nil {
		b.Fatalf("new reader: %v", err)
	}
	transferFunc.SetDRWAReader(reader)

	systemAcc := state[string(vmcommon.SystemAccountAddress)]
	mustSaveDRWATokenPolicyForBenchmark(b, systemAcc, "CARBON-123", &drwaTokenPolicyView{DRWAEnabled: true})
	mustSaveESDTBalanceForBenchmark(b, state["sender"], "CARBON-123", 1000000)
	mustSaveDRWAHolderForBenchmark(b, state["sender"], "CARBON-123", "sender", &drwaHolderMirrorView{
		KYCStatus: "approved",
		AMLStatus: "approved",
	})
	mustSaveDRWAHolderForBenchmark(b, state["receiver"], "CARBON-123", "receiver", &drwaHolderMirrorView{
		KYCStatus: "approved",
		AMLStatus: "approved",
	})

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: []byte("sender"),
			Arguments: [][]byte{
				[]byte("CARBON-123"),
				big.NewInt(1).Bytes(),
			},
			CallValue:   big.NewInt(0),
			GasProvided: 100,
			CallType:    vm.DirectCall,
		},
		RecipientAddr: []byte("receiver"),
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := transferFunc.ProcessBuiltinFunction(state["sender"], state["receiver"], vmInput)
		if err != nil {
			b.Fatalf("process transfer: %v", err)
		}
	}
}

func BenchmarkESDTNFTTransfer_ProcessBuiltinFunction_DRWACrossShardSenderDenied(b *testing.B) {
	for i := 0; i < b.N; i++ {
		vmInput, sender, nftTransfer, _, _, _ := createSetupToSendNFTCrossShard(&testing.T{})
		nftTransfer.enableEpochsHandler = &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return flag == DRWAEnforcementFlag || flag == SendAlwaysFlag || flag == SaveToSystemAccountFlag || flag == CheckFrozenCollectionFlag
			},
		}

		reader, err := newDRWAAccountsReader(nftTransfer.accounts)
		if err != nil {
			b.Fatalf("new reader: %v", err)
		}
		nftTransfer.SetDRWAReader(reader)

		systemAcc, err := nftTransfer.accounts.LoadAccount(vmcommon.SystemAccountAddress)
		if err != nil {
			b.Fatalf("load system account: %v", err)
		}
		mustSaveDRWATokenPolicyForBenchmark(b, systemAcc.(vmcommon.UserAccountHandler), "token", &drwaTokenPolicyView{
			DRWAEnabled: true,
		})

		mustSaveDRWAHolderForBenchmark(b, sender.(vmcommon.UserAccountHandler), "token", string(vmInput.CallerAddr), &drwaHolderMirrorView{
			KYCStatus: "pending",
			AMLStatus: "approved",
		})

		_, err = nftTransfer.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), nil, vmInput)
		if err == nil {
			b.Fatalf("expected cross-shard sender denial")
		}
	}
}

func mustSaveDRWATokenPolicyForBenchmark(b *testing.B, account vmcommon.UserAccountHandler, tokenID string, policy *drwaTokenPolicyView) {
	b.Helper()

	body, err := json.Marshal(policy)
	if err != nil {
		b.Fatalf("marshal token policy body: %v", err)
	}
	data, err := json.Marshal(&drwaStoredValue{Version: 1, Body: body})
	if err != nil {
		b.Fatalf("marshal token policy stored value: %v", err)
	}
	if err = account.AccountDataHandler().SaveKeyValue(BuildDRWATokenPolicyKey([]byte(tokenID)), data); err != nil {
		b.Fatalf("save token policy: %v", err)
	}
}

func mustSaveDRWAHolderForBenchmark(b *testing.B, account vmcommon.UserAccountHandler, tokenID string, address string, holder *drwaHolderMirrorView) {
	b.Helper()

	data, err := json.Marshal(holder)
	if err != nil {
		b.Fatalf("marshal holder mirror: %v", err)
	}
	if err = account.AccountDataHandler().SaveKeyValue(BuildDRWAHolderMirrorKey([]byte(tokenID), []byte(address)), data); err != nil {
		b.Fatalf("save holder mirror: %v", err)
	}
}

func mustSaveESDTBalanceForBenchmark(b *testing.B, account vmcommon.UserAccountHandler, tokenID string, value int64) {
	b.Helper()

	token := &esdt.ESDigitalToken{Value: big.NewInt(value)}
	data, err := (&mock.MarshalizerMock{}).Marshal(token)
	if err != nil {
		b.Fatalf("marshal balance: %v", err)
	}
	if err = account.AccountDataHandler().SaveKeyValue([]byte(baseESDTKeyPrefix+tokenID), data); err != nil {
		b.Fatalf("save balance: %v", err)
	}
}
