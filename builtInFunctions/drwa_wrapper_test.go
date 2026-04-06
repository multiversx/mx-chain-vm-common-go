package builtInFunctions

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/require"
)

type drwaReaderStub struct {
	getTokenPolicy func(tokenIdentifier []byte) (*drwaTokenPolicyView, error)
	getHolder      func(tokenIdentifier []byte, address []byte, currentAccount vmcommon.UserAccountHandler) (*drwaHolderMirrorView, error)
}

func (d *drwaReaderStub) GetTokenPolicy(tokenIdentifier []byte) (*drwaTokenPolicyView, error) {
	return d.getTokenPolicy(tokenIdentifier)
}

func (d *drwaReaderStub) GetHolderMirror(tokenIdentifier []byte, address []byte, currentAccount vmcommon.UserAccountHandler) (*drwaHolderMirrorView, error) {
	return d.getHolder(tokenIdentifier, address, currentAccount)
}

func TestCheckDRWAWrappersAndRegulatedTokenEvaluation(t *testing.T) {
	t.Parallel()

	reader := &drwaReaderStub{
		getTokenPolicy: func(tokenIdentifier []byte) (*drwaTokenPolicyView, error) {
			switch string(tokenIdentifier) {
			case "regulated":
				return &drwaTokenPolicyView{DRWAEnabled: true, MetadataProtectionEnabled: true, StrictAuditorMode: true}, nil
			case "plain":
				return nil, nil
			default:
				return &drwaTokenPolicyView{DRWAEnabled: false}, nil
			}
		},
		getHolder: func(tokenIdentifier []byte, address []byte, currentAccount vmcommon.UserAccountHandler) (*drwaHolderMirrorView, error) {
			switch string(tokenIdentifier) {
			case "regulated":
				return &drwaHolderMirrorView{
					KYCStatus:         "approved",
					AMLStatus:         "approved",
					InvestorClass:     "QIB",
					JurisdictionCode:  "US",
					AuditorAuthorized: true,
				}, nil
			case "receiver-blocked":
				return &drwaHolderMirrorView{KYCStatus: "approved", AMLStatus: "blocked"}, nil
			default:
				return nil, nil
			}
		},
	}

	regulated, policy, err := isDRWARegulatedToken(reader, []byte("regulated"))
	require.NoError(t, err)
	require.True(t, regulated)
	require.True(t, policy.DRWAEnabled)

	regulated, policy, err = isDRWARegulatedToken(reader, []byte("plain"))
	require.NoError(t, err)
	require.False(t, regulated)
	require.Nil(t, policy)

	// nil reader → not regulated, no error (fail-open for non-DRWA nodes)
	regulated, policy, err = isDRWARegulatedToken(nil, []byte("regulated"))
	require.False(t, regulated)
	require.Nil(t, policy)
	require.NoError(t, err)

	err = checkDRWASenderTransfer(reader, []byte("regulated"), []byte("holder"), mock.NewUserAccount([]byte("holder")), 1)
	require.NoError(t, err)

	regulated, err = evaluateDRWAReceiverTransfer(&drwaReaderStub{
		getTokenPolicy: func(tokenIdentifier []byte) (*drwaTokenPolicyView, error) {
			return &drwaTokenPolicyView{DRWAEnabled: true}, nil
		},
		getHolder: func(tokenIdentifier []byte, address []byte, currentAccount vmcommon.UserAccountHandler) (*drwaHolderMirrorView, error) {
			return &drwaHolderMirrorView{KYCStatus: "approved", AMLStatus: "blocked"}, nil
		},
	}, []byte("receiver-blocked"), []byte("holder"), nil, 1)
	require.True(t, regulated)
	require.ErrorIs(t, err, errDRWAAMLBlockedReceiver)

	err = checkDRWAMetadataUpdate(reader, []byte("regulated"), []byte("holder"), mock.NewUserAccount([]byte("holder")))
	require.NoError(t, err)

	regulated, err = evaluateDRWAMetadataUpdate(&drwaReaderStub{
		getTokenPolicy: func(tokenIdentifier []byte) (*drwaTokenPolicyView, error) {
			return nil, errors.New("policy failed")
		},
		getHolder: func(tokenIdentifier []byte, address []byte, currentAccount vmcommon.UserAccountHandler) (*drwaHolderMirrorView, error) {
			return nil, nil
		},
	}, []byte("broken"), []byte("holder"), nil)
	require.False(t, regulated)
	require.EqualError(t, err, "policy failed")

	regulated, err = evaluateDRWAMetadataUpdate(&drwaReaderStub{
		getTokenPolicy: func(tokenIdentifier []byte) (*drwaTokenPolicyView, error) {
			return &drwaTokenPolicyView{DRWAEnabled: false}, nil
		},
		getHolder: func(tokenIdentifier []byte, address []byte, currentAccount vmcommon.UserAccountHandler) (*drwaHolderMirrorView, error) {
			t.Fatalf("holder should not be loaded for unregulated token")
			return nil, nil
		},
	}, []byte("plain"), []byte("holder"), nil)
	require.False(t, regulated)
	require.NoError(t, err)
}

func TestGetTokenPolicyAndHolderMirrorErrorPaths(t *testing.T) {
	t.Parallel()

	retrieveErr := errors.New("retrieve failed")
	systemAccount := mock.NewAccountWrapMock(core.SystemAccountAddress)
	holderAccount := mock.NewAccountWrapMock([]byte("holder"))

	accounts := &mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			switch string(address) {
			case string(core.SystemAccountAddress):
				return systemAccount, nil
			case "holder":
				return holderAccount, nil
			default:
				return mock.NewUserAccount(address), nil
			}
		},
	}

	reader, err := newDRWAAccountsReader(accounts)
	require.NoError(t, err)

	systemAccount.RetrieveValueCalled = func(key []byte) ([]byte, uint32, error) {
		return nil, 0, retrieveErr
	}
	_, err = reader.GetTokenPolicy([]byte("CARBON-1"))
	require.ErrorIs(t, err, retrieveErr)

	systemAccount.RetrieveValueCalled = nil
	malformedWrapped, err := json.Marshal(&drwaStoredValue{Version: 1, Body: []byte("{")})
	require.NoError(t, err)
	require.NoError(t, systemAccount.SaveKeyValue(BuildDRWATokenPolicyKey([]byte("CARBON-2")), malformedWrapped))
	_, err = reader.GetTokenPolicy([]byte("CARBON-2"))
	require.Error(t, err)

	holderAccount.RetrieveValueCalled = func(key []byte) ([]byte, uint32, error) {
		return nil, 0, retrieveErr
	}
	_, err = reader.GetHolderMirror([]byte("CARBON-1"), []byte("holder"), nil)
	require.ErrorIs(t, err, retrieveErr)

	holderAccount.RetrieveValueCalled = nil
	mustSaveDRWAHolder(t, holderAccount, "CARBON-3", "holder", &drwaHolderMirrorView{
		KYCStatus: "approved",
		AMLStatus: "approved",
	})
	profileWrapped, err := json.Marshal(&drwaStoredValue{Version: 1, Body: []byte("{")})
	require.NoError(t, err)
	require.NoError(t, holderAccount.SaveKeyValue(BuildDRWAHolderProfileKey([]byte("holder")), profileWrapped))
	_, err = reader.GetHolderMirror([]byte("CARBON-3"), []byte("holder"), nil)
	require.Error(t, err)

	holderAccount = mock.NewAccountWrapMock([]byte("holder-auditor"))
	accounts.LoadAccountCalled = func(address []byte) (vmcommon.AccountHandler, error) {
		switch string(address) {
		case string(core.SystemAccountAddress):
			return systemAccount, nil
		case "holder-auditor":
			return holderAccount, nil
		default:
			return mock.NewUserAccount(address), nil
		}
	}
	mustSaveDRWAHolder(t, holderAccount, "CARBON-4", "holder-auditor", &drwaHolderMirrorView{
		KYCStatus: "approved",
		AMLStatus: "approved",
	})
	auditorWrapped, err := json.Marshal(&drwaStoredValue{Version: 1, Body: []byte("{")})
	require.NoError(t, err)
	require.NoError(t, holderAccount.SaveKeyValue(BuildDRWAHolderAuditorAuthorizationKey([]byte("CARBON-4"), []byte("holder-auditor")), auditorWrapped))
	_, err = reader.GetHolderMirror([]byte("CARBON-4"), []byte("holder-auditor"), nil)
	require.Error(t, err)
}

func TestGetHolderMirrorMergesProfileAndAuditorAuthorization(t *testing.T) {
	t.Parallel()

	holderAccount := mock.NewAccountWrapMock([]byte("holder"))
	accounts := &mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return holderAccount, nil
		},
	}

	reader, err := newDRWAAccountsReader(accounts)
	require.NoError(t, err)

	mustSaveDRWAHolder(t, holderAccount, "CARBON-1", "holder", &drwaHolderMirrorView{
		KYCStatus:         "pending",
		AMLStatus:         "approved",
		InvestorClass:     "RETAIL",
		JurisdictionCode:  "FR",
		ExpiryRound:       55,
		TransferLocked:    true,
		AuditorAuthorized: false,
	})
	mustSaveDRWAHolderProfile(t, holderAccount, "holder", &drwaHolderProfileView{
		KYCStatus:        "approved",
		AMLStatus:        "approved",
		InvestorClass:    "QIB",
		JurisdictionCode: "US",
		ExpiryRound:      99,
	})
	mustSaveDRWAHolderAuditorAuthorization(t, holderAccount, "CARBON-1", "holder", true)

	merged, err := reader.GetHolderMirror([]byte("CARBON-1"), []byte("holder"), nil)
	require.NoError(t, err)
	require.NotNil(t, merged)
	// At equal storedVersion (both 0), holder mirror wins for shared fields.
	// Profile fields (KYC=approved, InvestorClass=QIB, Jurisdiction=US) do NOT override
	// the mirror (KYC=pending, InvestorClass=RETAIL, Jurisdiction=FR).
	require.Equal(t, "pending", merged.KYCStatus)
	require.Equal(t, "RETAIL", merged.InvestorClass)
	require.Equal(t, "FR", merged.JurisdictionCode)
	require.Equal(t, uint64(55), merged.ExpiryRound)
	// IdentityExpiryRound falls through to profile when merged value is 0 (L-5)
	require.Equal(t, uint64(99), merged.IdentityExpiryRound)
	require.True(t, merged.TransferLocked)
	require.True(t, merged.AuditorAuthorized)
}

func TestValidateDRWASenderChecksBothIdentityAndTokenExpiry(t *testing.T) {
	t.Parallel()

	policy := &drwaTokenPolicyView{DRWAEnabled: true}
	holder := &drwaHolderMirrorView{
		KYCStatus:           "approved",
		AMLStatus:           "approved",
		ExpiryRound:         200,
		IdentityExpiryRound: 100,
	}

	decision := validateDRWASender(policy, holder, 150)
	require.ErrorIs(t, decision.DenialCode, errDRWAAssetExpired)

	holder.IdentityExpiryRound = 0
	decision = validateDRWASender(policy, holder, 150)
	require.True(t, decision.Allowed)

	decision = validateDRWASender(policy, holder, 250)
	require.ErrorIs(t, decision.DenialCode, errDRWAAssetExpired)
}

func TestValidateDRWAReceiverChecksIdentityExpiry(t *testing.T) {
	t.Parallel()

	decision := validateDRWAReceiver(nil, nil, 1)
	require.True(t, decision.Allowed)

	decision = validateDRWAReceiver(&drwaTokenPolicyView{DRWAEnabled: false}, nil, 1)
	require.True(t, decision.Allowed)

	decision = validateDRWAReceiver(&drwaTokenPolicyView{DRWAEnabled: true}, &drwaHolderMirrorView{
		KYCStatus:           "approved",
		AMLStatus:           "approved",
		IdentityExpiryRound: 10,
	}, 11)
	require.ErrorIs(t, decision.DenialCode, errDRWAAssetExpired)
}

func TestDRWAEvaluationErrorPaths(t *testing.T) {
	t.Parallel()

	// nil reader → not regulated, no error (fail-open)
	regulated, err := evaluateDRWASenderTransfer(nil, []byte("regulated"), []byte("holder"), nil, 1)
	require.False(t, regulated)
	require.NoError(t, err)

	regulated, err = evaluateDRWASenderTransfer(&drwaReaderStub{
		getTokenPolicy: func(tokenIdentifier []byte) (*drwaTokenPolicyView, error) {
			return &drwaTokenPolicyView{DRWAEnabled: true}, nil
		},
		getHolder: func(tokenIdentifier []byte, address []byte, currentAccount vmcommon.UserAccountHandler) (*drwaHolderMirrorView, error) {
			return nil, errors.New("sender holder failed")
		},
	}, []byte("regulated"), []byte("holder"), nil, 1)
	require.True(t, regulated)
	require.EqualError(t, err, "sender holder failed")

	regulated, err = evaluateDRWAReceiverTransfer(&drwaReaderStub{
		getTokenPolicy: func(tokenIdentifier []byte) (*drwaTokenPolicyView, error) {
			return &drwaTokenPolicyView{DRWAEnabled: true}, nil
		},
		getHolder: func(tokenIdentifier []byte, address []byte, currentAccount vmcommon.UserAccountHandler) (*drwaHolderMirrorView, error) {
			return nil, errors.New("receiver holder failed")
		},
	}, []byte("regulated"), []byte("holder"), nil, 1)
	require.True(t, regulated)
	require.EqualError(t, err, "receiver holder failed")

	regulated, err = evaluateDRWAMetadataUpdate(&drwaReaderStub{
		getTokenPolicy: func(tokenIdentifier []byte) (*drwaTokenPolicyView, error) {
			return &drwaTokenPolicyView{DRWAEnabled: true, MetadataProtectionEnabled: true, StrictAuditorMode: true}, nil
		},
		getHolder: func(tokenIdentifier []byte, address []byte, currentAccount vmcommon.UserAccountHandler) (*drwaHolderMirrorView, error) {
			return nil, errors.New("metadata holder failed")
		},
	}, []byte("regulated"), []byte("holder"), nil)
	require.True(t, regulated)
	require.EqualError(t, err, "metadata holder failed")
}

func TestDRWAAccountsReaderLoadUserAccountPropagatesLoadErrors(t *testing.T) {
	t.Parallel()

	reader, err := newDRWAAccountsReader(&mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return nil, errors.New("load failed")
		},
	})
	require.NoError(t, err)

	account, err := reader.loadUserAccount([]byte("missing"), nil)
	require.Nil(t, account)
	require.EqualError(t, err, "load failed")
}
