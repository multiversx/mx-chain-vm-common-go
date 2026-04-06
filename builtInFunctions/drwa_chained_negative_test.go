package builtInFunctions

// drwa_chained_negative_test.go — G9 Chained Negative Coverage
//
// These tests cover denial scenarios where multiple compliance attributes are
// relevant, verifying that the gate returns the correct denial code at the
// correct priority step even when other attributes are satisfied.
//
// Each test drives validateDRWASender or validateDRWAReceiver directly using
// inline drwaTokenPolicyView / drwaHolderMirrorView values — no I/O, no mocks.
//
// Scenarios covered:
//   TestDRWAChainedKYCThenAML                 — passes KYC but fails AML
//   TestDRWAChainedPausedOverridesKYC          — global pause beats KYC approval
//   TestDRWAChainedExpiryBlocksTransfer        — holder expiry_round passed
//   TestDRWAChainedInvestorClassBlocksJurOK    — jurisdiction OK but investor class blocked
//   TestDRWAChainedTransferLockedWhenCompliant — KYC+AML approved but transfer_locked
//   TestDRWAChainedReceiveLockedSenderOK       — sender OK but receiver receive_locked
//   TestDRWAChainedMetadataWithoutAuditor      — strict auditor mode, caller not authorized
//   TestDRWAChainedNilHolderRequiresKYC        — missing holder mirror maps to KYC_REQUIRED
//   TestDRWAChainedInvestorClassPrecedesJurisdiction — investor class check wins when both fail

import "testing"

// TestDRWAChainedKYCThenAML proves that a holder who passes KYC but is AML-blocked
// is denied with DRWA_AML_BLOCKED, not DRWA_KYC_REQUIRED.
func TestDRWAChainedKYCThenAML(t *testing.T) {
	policy := &drwaTokenPolicyView{DRWAEnabled: true}
	holder := &drwaHolderMirrorView{
		KYCStatus: "approved",
		AMLStatus: "blocked",
	}

	decision := validateDRWASender(policy, holder, 0)
	if decision.Allowed {
		t.Fatalf("expected denial, got allowed")
	}
	if decision.DenialCode != errDRWAAMLBlockedSender {
		t.Fatalf("expected DRWA_AML_BLOCKED_SENDER, got %v", decision.DenialCode)
	}
}

// TestDRWAChainedPausedOverridesKYC proves that a globally paused token denies
// the transfer with DRWA_TOKEN_PAUSED even when the holder has full KYC+AML approval.
func TestDRWAChainedPausedOverridesKYC(t *testing.T) {
	policy := &drwaTokenPolicyView{DRWAEnabled: true, GlobalPause: true}
	holder := &drwaHolderMirrorView{
		KYCStatus: "approved",
		AMLStatus: "approved",
	}

	decision := validateDRWASender(policy, holder, 0)
	if decision.Allowed {
		t.Fatalf("expected denial, got allowed")
	}
	if decision.DenialCode != errDRWATokenPaused {
		t.Fatalf("expected DRWA_TOKEN_PAUSED, got %v", decision.DenialCode)
	}
}

// TestDRWAChainedExpiryBlocksTransfer proves that a holder whose expiry_round has
// passed is denied with DRWA_ASSET_EXPIRED.  The token is not paused and the holder
// has valid KYC+AML, so only the expiry check triggers.
func TestDRWAChainedExpiryBlocksTransfer(t *testing.T) {
	policy := &drwaTokenPolicyView{DRWAEnabled: true}
	holder := &drwaHolderMirrorView{
		KYCStatus:   "approved",
		AMLStatus:   "approved",
		ExpiryRound: 50,
	}
	const currentRound = uint64(100) // now > ExpiryRound

	decision := validateDRWASender(policy, holder, currentRound)
	if decision.Allowed {
		t.Fatalf("expected denial, got allowed")
	}
	if decision.DenialCode != errDRWAAssetExpired {
		t.Fatalf("expected DRWA_ASSET_EXPIRED, got %v", decision.DenialCode)
	}
}

// TestDRWAChainedInvestorClassBlocksJurOK proves that DRWA_INVESTOR_CLASS_BLOCKED
// is returned when the holder's jurisdiction is in the allowed set but their
// investor class is not.
func TestDRWAChainedInvestorClassBlocksJurOK(t *testing.T) {
	policy := &drwaTokenPolicyView{
		DRWAEnabled:            true,
		AllowedInvestorClasses: map[string]bool{"QIB": true},
		AllowedJurisdictions:   map[string]bool{"US": true},
	}
	holder := &drwaHolderMirrorView{
		KYCStatus:        "approved",
		AMLStatus:        "approved",
		InvestorClass:    "RETAIL", // not in allowed set
		JurisdictionCode: "US",     // in allowed set
	}

	decision := validateDRWASender(policy, holder, 0)
	if decision.Allowed {
		t.Fatalf("expected denial, got allowed")
	}
	if decision.DenialCode != errDRWAInvestorClass {
		t.Fatalf("expected DRWA_INVESTOR_CLASS_BLOCKED, got %v", decision.DenialCode)
	}
}

// TestDRWAChainedTransferLockedWhenCompliant proves that a holder who is fully
// KYC+AML approved is still denied with DRWA_TRANSFER_LOCKED when transfer_locked=true.
func TestDRWAChainedTransferLockedWhenCompliant(t *testing.T) {
	policy := &drwaTokenPolicyView{DRWAEnabled: true}
	holder := &drwaHolderMirrorView{
		KYCStatus:      "approved",
		AMLStatus:      "approved",
		TransferLocked: true,
	}

	decision := validateDRWASender(policy, holder, 0)
	if decision.Allowed {
		t.Fatalf("expected denial, got allowed")
	}
	if decision.DenialCode != errDRWATransferLocked {
		t.Fatalf("expected DRWA_TRANSFER_LOCKED, got %v", decision.DenialCode)
	}
}

// TestDRWAChainedReceiveLockedSenderOK proves that when the sender is fully
// compliant but the receiver has receive_locked=true, the receiver gate returns
// DRWA_RECEIVE_LOCKED.
func TestDRWAChainedReceiveLockedSenderOK(t *testing.T) {
	policy := &drwaTokenPolicyView{DRWAEnabled: true}

	senderHolder := &drwaHolderMirrorView{
		KYCStatus: "approved",
		AMLStatus: "approved",
	}
	receiverHolder := &drwaHolderMirrorView{
		KYCStatus:    "approved",
		AMLStatus:    "approved",
		ReceiveLocked: true,
	}

	senderDecision := validateDRWASender(policy, senderHolder, 0)
	if !senderDecision.Allowed {
		t.Fatalf("expected sender allowed, got %v", senderDecision.DenialCode)
	}

	receiverDecision := validateDRWAReceiver(policy, receiverHolder, 0)
	if receiverDecision.Allowed {
		t.Fatalf("expected receiver denial, got allowed")
	}
	if receiverDecision.DenialCode != errDRWAReceiveLocked {
		t.Fatalf("expected DRWA_RECEIVE_LOCKED, got %v", receiverDecision.DenialCode)
	}
}

// TestDRWAChainedMetadataWithoutAuditor proves that in strict auditor mode a caller
// whose auditor_authorized=false is denied with DRWA_AUDITOR_REQUIRED.
func TestDRWAChainedMetadataWithoutAuditor(t *testing.T) {
	policy := &drwaTokenPolicyView{
		DRWAEnabled:               true,
		MetadataProtectionEnabled: true,
		StrictAuditorMode:         true,
	}
	const auditorAuthorized = false

	decision := validateDRWAMetadataUpdate(policy, auditorAuthorized)
	if decision.Allowed {
		t.Fatalf("expected denial, got allowed")
	}
	if decision.DenialCode != errDRWAAuditorRequired {
		t.Fatalf("expected DRWA_AUDITOR_REQUIRED, got %v", decision.DenialCode)
	}
}

func TestDRWAChainedNilHolderRequiresKYC(t *testing.T) {
	policy := &drwaTokenPolicyView{DRWAEnabled: true}

	decision := validateDRWASender(policy, nil, 0)
	if decision.Allowed {
		t.Fatalf("expected denial, got allowed")
	}
	if decision.DenialCode != errDRWAKYCRequiredSender {
		t.Fatalf("expected DRWA_KYC_REQUIRED_SENDER, got %v", decision.DenialCode)
	}
}

func TestDRWAChainedInvestorClassPrecedesJurisdiction(t *testing.T) {
	policy := &drwaTokenPolicyView{
		DRWAEnabled:            true,
		AllowedInvestorClasses: map[string]bool{"QIB": true},
		AllowedJurisdictions:   map[string]bool{"US": true},
	}
	holder := &drwaHolderMirrorView{
		KYCStatus:        "approved",
		AMLStatus:        "approved",
		InvestorClass:    "RETAIL",
		JurisdictionCode: "FR",
	}

	decision := validateDRWASender(policy, holder, 0)
	if decision.Allowed {
		t.Fatalf("expected denial, got allowed")
	}
	if decision.DenialCode != errDRWAInvestorClass {
		t.Fatalf("expected DRWA_INVESTOR_CLASS_BLOCKED, got %v", decision.DenialCode)
	}
}
