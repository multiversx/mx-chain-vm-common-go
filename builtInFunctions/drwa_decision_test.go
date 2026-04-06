package builtInFunctions

import "testing"

func TestValidateDRWAReceiverBranches(t *testing.T) {
	t.Parallel()

	if decision := validateDRWAReceiver(nil, nil, 0); !decision.Allowed {
		t.Fatalf("nil policy should allow receiver")
	}

	paused := validateDRWAReceiver(&drwaTokenPolicyView{DRWAEnabled: true, GlobalPause: true}, &drwaHolderMirrorView{}, 0)
	if paused.DenialCode != errDRWATokenPaused {
		t.Fatalf("expected paused denial, got %v", paused.DenialCode)
	}

	kyc := validateDRWAReceiver(&drwaTokenPolicyView{DRWAEnabled: true}, nil, 0)
	if kyc.DenialCode != errDRWAKYCRequiredReceiver {
		t.Fatalf("expected receiver kyc denial, got %v", kyc.DenialCode)
	}

	aml := validateDRWAReceiver(&drwaTokenPolicyView{DRWAEnabled: true}, &drwaHolderMirrorView{
		KYCStatus: "approved",
		AMLStatus: "blocked",
	}, 0)
	if aml.DenialCode != errDRWAAMLBlockedReceiver {
		t.Fatalf("expected aml denial, got %v", aml.DenialCode)
	}

	expired := validateDRWAReceiver(&drwaTokenPolicyView{DRWAEnabled: true}, &drwaHolderMirrorView{
		KYCStatus:   "approved",
		AMLStatus:   "approved",
		ExpiryRound: 10,
	}, 11)
	if expired.DenialCode != errDRWAAssetExpired {
		t.Fatalf("expected expiry denial, got %v", expired.DenialCode)
	}

	investorClass := validateDRWAReceiver(&drwaTokenPolicyView{
		DRWAEnabled:            true,
		AllowedInvestorClasses: map[string]bool{"QIB": true},
	}, &drwaHolderMirrorView{
		KYCStatus:     "approved",
		AMLStatus:     "approved",
		InvestorClass: "RETAIL",
	}, 0)
	if investorClass.DenialCode != errDRWAInvestorClass {
		t.Fatalf("expected investor class denial, got %v", investorClass.DenialCode)
	}

	jurisdiction := validateDRWAReceiver(&drwaTokenPolicyView{
		DRWAEnabled:          true,
		AllowedJurisdictions: map[string]bool{"US": true},
	}, &drwaHolderMirrorView{
		KYCStatus:        "approved",
		AMLStatus:        "approved",
		JurisdictionCode: "FR",
	}, 0)
	if jurisdiction.DenialCode != errDRWAJurisdiction {
		t.Fatalf("expected jurisdiction denial, got %v", jurisdiction.DenialCode)
	}

	allowed := validateDRWAReceiver(&drwaTokenPolicyView{
		DRWAEnabled:            true,
		AllowedInvestorClasses: map[string]bool{"QIB": true},
		AllowedJurisdictions:   map[string]bool{"US": true},
	}, &drwaHolderMirrorView{
		KYCStatus:        "approved",
		AMLStatus:        "approved",
		InvestorClass:    "QIB",
		JurisdictionCode: "US",
	}, 0)
	if !allowed.Allowed {
		t.Fatalf("expected allowed receiver, got %v", allowed.DenialCode)
	}
}

func TestValidateDRWAMetadataUpdateBranches(t *testing.T) {
	t.Parallel()

	if decision := validateDRWAMetadataUpdate(nil, false); !decision.Allowed {
		t.Fatalf("nil policy should allow metadata update")
	}

	if decision := validateDRWAMetadataUpdate(&drwaTokenPolicyView{DRWAEnabled: true}, false); !decision.Allowed {
		t.Fatalf("metadata protection disabled should allow")
	}

	if decision := validateDRWAMetadataUpdate(&drwaTokenPolicyView{
		DRWAEnabled:               true,
		MetadataProtectionEnabled: true,
		StrictAuditorMode:         false,
	}, false); !decision.Allowed {
		t.Fatalf("non-strict metadata protection should allow")
	}

	denied := validateDRWAMetadataUpdate(&drwaTokenPolicyView{
		DRWAEnabled:               true,
		MetadataProtectionEnabled: true,
		StrictAuditorMode:         true,
	}, false)
	if denied.DenialCode != errDRWAAuditorRequired {
		t.Fatalf("expected auditor required denial, got %v", denied.DenialCode)
	}

	allowed := validateDRWAMetadataUpdate(&drwaTokenPolicyView{
		DRWAEnabled:               true,
		MetadataProtectionEnabled: true,
		StrictAuditorMode:         true,
	}, true)
	if !allowed.Allowed {
		t.Fatalf("expected auditor-authorized metadata update to pass")
	}
}
