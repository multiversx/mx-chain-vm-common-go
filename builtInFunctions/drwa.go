package builtInFunctions

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	logger "github.com/multiversx/mx-chain-logger-go"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

var logDRWA = logger.GetOrCreate("builtInFunctions/drwa")

const (
	drwaTokenPolicyPrefix       = "drwa:token:"
	drwaHolderMirrorPrefix      = "drwa:holder:"
	drwaHolderProfilePrefix     = "drwa:profile:"
	drwaHolderAuditorAuthPrefix = "drwa:auditor:"
	drwaReadGasUnits            = 1
)

// Exported constants for cross-module validation.
// The sync adapter in mx-chain-go MUST use identical prefix values.
// Any divergence silently breaks the entire enforcement system.
const (
	DRWATokenPolicyPrefix       = drwaTokenPolicyPrefix
	DRWAHolderMirrorPrefix      = drwaHolderMirrorPrefix
	DRWAHolderProfilePrefix     = drwaHolderProfilePrefix
	DRWAHolderAuditorAuthPrefix = drwaHolderAuditorAuthPrefix
	DRWAAssetRecordPrefix       = "drwa:asset:"
)

var (
	errDRWATokenPaused          = errors.New("DRWA_TOKEN_PAUSED")
	errDRWAKYCRequiredSender    = errors.New("DRWA_KYC_REQUIRED_SENDER")
	errDRWAAMLBlockedSender     = errors.New("DRWA_AML_BLOCKED_SENDER")
	errDRWAAssetExpired         = errors.New("DRWA_ASSET_EXPIRED")
	errDRWATransferLocked       = errors.New("DRWA_TRANSFER_LOCKED")
	errDRWAKYCRequiredReceiver  = errors.New("DRWA_KYC_REQUIRED_RECEIVER")
	errDRWAAMLBlockedReceiver   = errors.New("DRWA_AML_BLOCKED_RECEIVER")
	errDRWAReceiveLocked        = errors.New("DRWA_RECEIVE_LOCKED")
	errDRWAInvestorClass        = errors.New("DRWA_INVESTOR_CLASS_BLOCKED")
	errDRWAJurisdiction         = errors.New("DRWA_JURISDICTION_BLOCKED")
	errDRWAAuditorRequired      = errors.New("DRWA_AUDITOR_REQUIRED")
	errDRWAStateReaderMissing   = errors.New("DRWA_STATE_READER_MISSING")
	errDRWANilAccountsAdapter   = errors.New("nil DRWA accounts adapter")
)

type drwaDecision struct {
	Allowed    bool
	DenialCode error
}

type drwaTokenPolicyView struct {
	DRWAEnabled               bool            `json:"drwa_enabled"`
	GlobalPause               bool            `json:"global_pause"`
	StrictAuditorMode         bool            `json:"strict_auditor_mode"`
	MetadataProtectionEnabled bool            `json:"metadata_protection_enabled"`
	AllowedInvestorClasses    map[string]bool `json:"allowed_investor_classes,omitempty"`
	AllowedJurisdictions      map[string]bool `json:"allowed_jurisdictions,omitempty"`
}

type drwaHolderMirrorView struct {
	KYCStatus           string `json:"kyc_status"`
	AMLStatus           string `json:"aml_status"`
	InvestorClass       string `json:"investor_class,omitempty"`
	JurisdictionCode    string `json:"jurisdiction_code,omitempty"`
	ExpiryRound         uint64 `json:"expiry_round,omitempty"`
	IdentityExpiryRound uint64 `json:"-"`
	// storedVersion is populated during decode from the drwaStoredValue wrapper.
	// Used to resolve merge precedence when both holder mirror and profile exist.
	storedVersion uint64
	// TransferLocked and ReceiveLocked are top-level fields whose JSON tags
	// match the DrwaHolderMirror struct fields serialized by the Rust contracts.
	// They must NOT be stored in a nested map — json.Unmarshal cannot populate
	// map entries from top-level JSON keys.
	TransferLocked    bool `json:"transfer_locked,omitempty"`
	ReceiveLocked     bool `json:"receive_locked,omitempty"`
	AuditorAuthorized bool `json:"auditor_authorized,omitempty"`
}

type drwaHolderProfileView struct {
	KYCStatus        string `json:"kyc_status"`
	AMLStatus        string `json:"aml_status"`
	InvestorClass    string `json:"investor_class,omitempty"`
	JurisdictionCode string `json:"jurisdiction_code,omitempty"`
	ExpiryRound      uint64 `json:"expiry_round,omitempty"`
	storedVersion    uint64
}

type drwaHolderAuditorAuthorizationView struct {
	AuditorAuthorized bool `json:"auditor_authorized,omitempty"`
}

type drwaStateReader interface {
	GetTokenPolicy(tokenIdentifier []byte) (*drwaTokenPolicyView, error)
	GetHolderMirror(tokenIdentifier []byte, address []byte, currentAccount vmcommon.UserAccountHandler) (*drwaHolderMirrorView, error)
}

type drwaAccountsReader struct {
	accounts vmcommon.AccountsAdapter
}

type drwaStoredValue struct {
	Version uint64 `json:"version"`
	Body    []byte `json:"body"`
}

func newDRWAAccountsReader(accounts vmcommon.AccountsAdapter) (*drwaAccountsReader, error) {
	if accounts == nil || accounts.IsInterfaceNil() {
		return nil, errDRWANilAccountsAdapter
	}

	return &drwaAccountsReader{accounts: accounts}, nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (d *drwaAccountsReader) IsInterfaceNil() bool {
	return d == nil
}

// BuildDRWATokenPolicyKey constructs the storage key for a token's policy entry.
// Exported so mx-chain-go sync adapter can reuse the canonical key builder.
func BuildDRWATokenPolicyKey(tokenIdentifier []byte) []byte {
	return []byte(drwaTokenPolicyPrefix + hex.EncodeToString(tokenIdentifier) + ":policy")
}

// BuildDRWAHolderMirrorKey constructs the storage key for a holder's mirror entry.
// Exported so mx-chain-go sync adapter can reuse the canonical key builder.
func BuildDRWAHolderMirrorKey(tokenIdentifier []byte, address []byte) []byte {
	return []byte(drwaHolderMirrorPrefix + hex.EncodeToString(tokenIdentifier) + ":" + hex.EncodeToString(address))
}

// BuildDRWAHolderProfileKey constructs the storage key for a holder's profile entry.
// Exported so mx-chain-go sync adapter can reuse the canonical key builder.
func BuildDRWAHolderProfileKey(address []byte) []byte {
	return []byte(drwaHolderProfilePrefix + hex.EncodeToString(address))
}

// BuildDRWAHolderAuditorAuthorizationKey constructs the storage key for a holder's auditor authorization entry.
// Exported so mx-chain-go sync adapter can reuse the canonical key builder.
func BuildDRWAHolderAuditorAuthorizationKey(tokenIdentifier []byte, address []byte) []byte {
	return []byte(drwaHolderAuditorAuthPrefix + hex.EncodeToString(tokenIdentifier) + ":" + hex.EncodeToString(address))
}

func (d *drwaAccountsReader) GetTokenPolicy(tokenIdentifier []byte) (*drwaTokenPolicyView, error) {
	systemAccount, err := getSystemAccount(d.accounts)
	if err != nil {
		return nil, err
	}

	data, _, err := systemAccount.AccountDataHandler().RetrieveValue(BuildDRWATokenPolicyKey(tokenIdentifier))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	policy := &drwaTokenPolicyView{}
	err = decodeDRWAStoredJSON(data, policy)
	if err != nil {
		return nil, fmt.Errorf("drwa token policy unmarshal: %w", err)
	}

	return policy, nil
}

func (d *drwaAccountsReader) GetHolderMirror(tokenIdentifier []byte, address []byte, currentAccount vmcommon.UserAccountHandler) (*drwaHolderMirrorView, error) {
	account, err := d.loadUserAccount(address, currentAccount)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, nil
	}

	var holder *drwaHolderMirrorView
	data, _, err := account.AccountDataHandler().RetrieveValue(BuildDRWAHolderMirrorKey(tokenIdentifier, address))
	if err != nil {
		return nil, err
	}
	if len(data) > 0 {
		holder = &drwaHolderMirrorView{}
		err = decodeDRWAStoredJSON(data, holder)
		if err != nil {
			return nil, fmt.Errorf("drwa holder mirror unmarshal: %w", err)
		}
	}

	var profile *drwaHolderProfileView
	profileData, _, err := account.AccountDataHandler().RetrieveValue(BuildDRWAHolderProfileKey(address))
	if err != nil {
		return nil, err
	}
	if len(profileData) > 0 {
		profile = &drwaHolderProfileView{}
		err = decodeDRWAStoredJSON(profileData, profile)
		if err != nil {
			return nil, fmt.Errorf("drwa holder profile unmarshal: %w", err)
		}
	}

	var auditorAuth *drwaHolderAuditorAuthorizationView
	auditorData, _, err := account.AccountDataHandler().RetrieveValue(BuildDRWAHolderAuditorAuthorizationKey(tokenIdentifier, address))
	if err != nil {
		return nil, err
	}
	if len(auditorData) > 0 {
		auditorAuth = &drwaHolderAuditorAuthorizationView{}
		err = decodeDRWAStoredJSON(auditorData, auditorAuth)
		if err != nil {
			return nil, fmt.Errorf("drwa holder auditor auth unmarshal: %w", err)
		}
	}

	if holder == nil && profile == nil && auditorAuth == nil {
		return nil, nil
	}

	merged := &drwaHolderMirrorView{}
	if holder != nil {
		if holder.KYCStatus != "" {
			merged.KYCStatus = holder.KYCStatus
		}
		if holder.AMLStatus != "" {
			merged.AMLStatus = holder.AMLStatus
		}
		if holder.InvestorClass != "" {
			merged.InvestorClass = holder.InvestorClass
		}
		if holder.JurisdictionCode != "" {
			merged.JurisdictionCode = holder.JurisdictionCode
		}
		if holder.ExpiryRound != 0 {
			merged.ExpiryRound = holder.ExpiryRound
		}
		merged.TransferLocked = holder.TransferLocked
		merged.ReceiveLocked = holder.ReceiveLocked
		merged.AuditorAuthorized = holder.AuditorAuthorized
	}
	if profile != nil {
		// Only let profile overwrite shared compliance fields if its
		// version is >= the holder mirror version. This prevents a stale
		// profile sync from reverting a newer holder mirror update.
		// Strict > (not >=). Equal versions means both were written
		// in the same sync cycle — holder mirror is authoritative for shared
		// fields because asset-manager writes it with token-specific context.
		profileWins := holder == nil || profile.storedVersion > holder.storedVersion
		if profileWins {
			merged.KYCStatus = profile.KYCStatus
			merged.AMLStatus = profile.AMLStatus
			merged.InvestorClass = profile.InvestorClass
			merged.JurisdictionCode = profile.JurisdictionCode
		}
		// L-5: At equal versions, holder mirror wins for shared fields (ExpiryRound, KycStatus, etc.)
		// but IdentityExpiryRound falls through to the profile if not yet set on the merged result.
		// This is intentional: IdentityExpiryRound is identity-registry-specific and may not be
		// present in the holder mirror, while ExpiryRound is policy-level.
		if profileWins || merged.IdentityExpiryRound == 0 {
			merged.IdentityExpiryRound = profile.ExpiryRound
		}
		if merged.ExpiryRound == 0 {
			merged.ExpiryRound = profile.ExpiryRound
		}
	}
	if auditorAuth != nil {
		merged.AuditorAuthorized = auditorAuth.AuditorAuthorized
	}

	return merged, nil
}

func (d *drwaAccountsReader) loadUserAccount(address []byte, currentAccount vmcommon.UserAccountHandler) (vmcommon.UserAccountHandler, error) {
	if currentAccount != nil && !currentAccount.IsInterfaceNil() && bytes.Equal(currentAccount.AddressBytes(), address) {
		return currentAccount, nil
	}

	accountHandler, err := d.accounts.LoadAccount(address)
	if err != nil {
		return nil, err
	}

	userAccount, ok := accountHandler.(vmcommon.UserAccountHandler)
	if !ok {
		return nil, ErrWrongTypeAssertion
	}

	return userAccount, nil
}

func decodeDRWAStoredJSON(data []byte, destination interface{}) error {
	storedValue := &drwaStoredValue{}
	jsonErr := json.Unmarshal(data, storedValue)
	var err error
	if jsonErr == nil && len(storedValue.Body) > 0 {
		err = decodeDRWABody(storedValue.Body, destination)
		// Propagate stored version for merge precedence
		if err == nil {
			switch typed := destination.(type) {
			case *drwaHolderMirrorView:
				typed.storedVersion = storedValue.Version
			case *drwaHolderProfileView:
				typed.storedVersion = storedValue.Version
			}
		}
	} else {
		if jsonErr == nil {
			err = errors.New("missing drwa wrapped body")
		} else {
			err = jsonErr
		}
	}
	if err != nil {
		recordDRWADecodeFailure(data, destination, err)
	}
	return err
}

func recordDRWADecodeFailure(data []byte, destination interface{}, err error) {
	logDRWA.Warn("drwa stored value decode failure", "error", err, "metric", classifyDRWADecodeFailureMetric(data, destination))
	recordDRWAGateMetric(drwaGateMetricDecodeFailure)
	recordDRWAGateMetric(classifyDRWADecodeFailureMetric(data, destination))
}

func classifyDRWADecodeFailureMetric(data []byte, destination interface{}) string {
	if len(data) == 0 {
		return drwaGateMetricDecodeFailureMissing
	}
	if data[0] == '{' {
		return drwaGateMetricDecodeFailureJSON
	}

	switch destination.(type) {
	case *drwaTokenPolicyView, *drwaHolderMirrorView, *drwaHolderProfileView, *drwaHolderAuditorAuthorizationView:
		return drwaGateMetricDecodeFailureBinary
	default:
		return drwaGateMetricDecodeFailure
	}
}

func decodeDRWABody(data []byte, destination interface{}) error {
	// Bodies that start with '{' are JSON-format.  Do not fall back to the
	// binary decoder when JSON parsing fails: a corrupt JSON body must surface
	// as a parse error.  Silently re-routing to the binary decoder would drop
	// AllowedInvestorClasses / AllowedJurisdictions enforcement entirely.
	if len(data) > 0 && data[0] == '{' {
		return json.Unmarshal(data, destination)
	}

	switch typedDestination := destination.(type) {
	case *drwaTokenPolicyView:
		return decodeDRWABinaryTokenPolicy(data, typedDestination)
	case *drwaHolderMirrorView:
		return decodeDRWABinaryHolderMirror(data, typedDestination)
	case *drwaHolderProfileView:
		return decodeDRWABinaryHolderProfile(data, typedDestination)
	case *drwaHolderAuditorAuthorizationView:
		return decodeDRWABinaryHolderAuditorAuthorization(data, typedDestination)
	default:
		return json.Unmarshal(data, destination)
	}
}

func decodeDRWABinaryTokenPolicy(data []byte, destination *drwaTokenPolicyView) error {
	if len(data) < 12 {
		return errors.New("invalid DRWA binary token policy payload")
	}

	destination.DRWAEnabled = data[0] == 1
	destination.GlobalPause = data[1] == 1
	destination.StrictAuditorMode = data[2] == 1
	destination.MetadataProtectionEnabled = data[3] == 1

	// Validate bytes 4-11 are zero. Reserved bytes must not contain
	// arbitrary data (prevents covert channels and future format confusion).
	for i := 4; i < 12 && i < len(data); i++ {
		if data[i] != 0 {
			return fmt.Errorf("DRWA binary policy byte %d is non-zero (%d): reserved bytes must be 0", i, data[i])
		}
	}

	// Binary format cannot encode AllowedInvestorClasses or
	// AllowedJurisdictions maps. These remain nil after binary decode.
	// The enforcement gate checks `len(map) > 0` before using them,
	// so nil maps mean "no restriction" — all classes/jurisdictions allowed.
	// This is SAFE because the Rust policy-registry contract serializes
	// token policies as JSON when investor_classes or jurisdictions are set.
	// Binary format is only used for policies with boolean-only flags.

	return nil
}

func decodeDRWABinaryHolderMirror(data []byte, destination *drwaHolderMirrorView) error {
	cursor := 0
	if len(data) < 8 {
		return errors.New("invalid DRWA binary holder payload")
	}

	// First 8 bytes are the holder_policy_version (big-endian uint64).
	// Stored for storedVersion propagation in merge precedence resolution.
	holderVersion := binary.BigEndian.Uint64(data[0:8])
	destination.storedVersion = holderVersion
	cursor += 8

	kycStatus, nextCursor, err := readDRWABinaryField(data, cursor)
	if err != nil {
		return err
	}
	cursor = nextCursor

	amlStatus, nextCursor, err := readDRWABinaryField(data, cursor)
	if err != nil {
		return err
	}
	cursor = nextCursor

	investorClass, nextCursor, err := readDRWABinaryField(data, cursor)
	if err != nil {
		return err
	}
	cursor = nextCursor

	jurisdictionCode, nextCursor, err := readDRWABinaryField(data, cursor)
	if err != nil {
		return err
	}
	cursor = nextCursor

	if len(data[cursor:]) < 11 {
		return errors.New("invalid DRWA binary holder trailer")
	}

	destination.KYCStatus = string(kycStatus)
	destination.AMLStatus = string(amlStatus)
	destination.InvestorClass = string(investorClass)
	destination.JurisdictionCode = string(jurisdictionCode)
	destination.ExpiryRound = binary.BigEndian.Uint64(data[cursor : cursor+8])
	// Validate boolean trailer bytes are 0 or 1 (reject malformed payloads)
	for _, idx := range []int{cursor + 8, cursor + 9, cursor + 10} {
		if data[idx] != 0 && data[idx] != 1 {
			return fmt.Errorf("DRWA binary holder byte %d invalid: %d (expected 0 or 1)", idx, data[idx])
		}
	}
	destination.TransferLocked = data[cursor+8] == 1
	destination.ReceiveLocked = data[cursor+9] == 1
	destination.AuditorAuthorized = data[cursor+10] == 1

	return nil
}

func decodeDRWABinaryHolderProfile(data []byte, destination *drwaHolderProfileView) error {
	cursor := 0
	if len(data) < 8 {
		return errors.New("invalid DRWA binary holder profile payload")
	}

	// Extract stored version for merge precedence resolution.
	destination.storedVersion = binary.BigEndian.Uint64(data[0:8])
	cursor += 8

	kycStatus, nextCursor, err := readDRWABinaryField(data, cursor)
	if err != nil {
		return err
	}
	cursor = nextCursor

	amlStatus, nextCursor, err := readDRWABinaryField(data, cursor)
	if err != nil {
		return err
	}
	cursor = nextCursor

	investorClass, nextCursor, err := readDRWABinaryField(data, cursor)
	if err != nil {
		return err
	}
	cursor = nextCursor

	jurisdictionCode, nextCursor, err := readDRWABinaryField(data, cursor)
	if err != nil {
		return err
	}
	cursor = nextCursor

	if len(data[cursor:]) < 8 {
		return errors.New("invalid DRWA binary holder profile trailer")
	}

	destination.KYCStatus = string(kycStatus)
	destination.AMLStatus = string(amlStatus)
	destination.InvestorClass = string(investorClass)
	destination.JurisdictionCode = string(jurisdictionCode)
	destination.ExpiryRound = binary.BigEndian.Uint64(data[cursor : cursor+8])

	return nil
}

// decodeDRWABinaryHolderAuditorAuthorization decodes the binary auditor
// authorization payload. Bytes 0-7 are reserved for future use and are
// intentionally ignored — auditor authorization carries no version tracking
// (unlike holder mirror and profile records which embed a version in the
// first 8 bytes). The authorization boolean is at byte offset 8.
func decodeDRWABinaryHolderAuditorAuthorization(data []byte, destination *drwaHolderAuditorAuthorizationView) error {
	if len(data) < 9 {
		return errors.New("invalid DRWA binary holder auditor authorization payload")
	}

	// Validate boolean byte is 0 or 1 (reject other values)
	if data[8] != 0 && data[8] != 1 {
		return fmt.Errorf("DRWA binary auditor auth byte invalid: %d (expected 0 or 1)", data[8])
	}
	destination.AuditorAuthorized = data[8] == 1
	return nil
}

func readDRWABinaryField(data []byte, cursor int) ([]byte, int, error) {
	if len(data[cursor:]) < 4 {
		return nil, cursor, errors.New("invalid DRWA binary field length")
	}

	fieldLength := int(binary.BigEndian.Uint32(data[cursor : cursor+4]))
	// Cap field length before allocation to prevent memory exhaustion
	// from crafted payloads. Matches drwaSyncMaxFieldBytes in the sync layer.
	if fieldLength > 64*1024 {
		return nil, cursor, fmt.Errorf("DRWA binary field length %d exceeds max 65536", fieldLength)
	}
	cursor += 4
	if len(data[cursor:]) < fieldLength {
		return nil, cursor, errors.New("invalid DRWA binary field body")
	}

	field := append([]byte(nil), data[cursor:cursor+fieldLength]...)
	cursor += fieldLength

	return field, cursor, nil
}

func isDRWAEnforcementEnabled(enableEpochsHandler vmcommon.EnableEpochsHandler) bool {
	if enableEpochsHandler == nil || enableEpochsHandler.IsInterfaceNil() {
		return false
	}

	return enableEpochsHandler.IsFlagEnabled(DRWAEnforcementFlag)
}

func computeDRWAReadGasCost(baseCost vmcommon.BaseOperationCost, fallbackCost uint64, reads uint64) uint64 {
	if reads == 0 {
		return 0
	}

	unitCost := baseCost.DataCopyPerByte
	if unitCost == 0 {
		unitCost = fallbackCost
	}
	if unitCost == 0 {
		return 0
	}

	// Overflow protection for gas calculation.
	if unitCost > 0 && reads > math.MaxUint64/(unitCost*drwaReadGasUnits) {
		return math.MaxUint64
	}
	return reads * unitCost * drwaReadGasUnits
}

func isDRWARegulatedToken(reader drwaStateReader, tokenIdentifier []byte) (bool, *drwaTokenPolicyView, error) {
	if reader == nil {
		// No DRWA reader attached — token is not under DRWA regulation.
		// This is the expected state for nodes that have not enabled DRWA enforcement.
		return false, nil, nil
	}

	policy, err := reader.GetTokenPolicy(tokenIdentifier)
	if err != nil {
		return false, nil, err
	}
	if policy == nil || !policy.DRWAEnabled {
		return false, nil, nil
	}

	return true, policy, nil
}

// validateDRWASender enforces compliance on the sending side.
// InvestorClass and Jurisdiction are checked on BOTH sender and
// receiver, which is stricter than spec S4.1 (which positions codes 9-10 as
// receiver-only). Intentional design choice: prevents non-qualified
// intermediaries from acting as distribution conduits.
func validateDRWASender(policy *drwaTokenPolicyView, holder *drwaHolderMirrorView, now uint64) drwaDecision {
	if policy == nil || !policy.DRWAEnabled {
		return drwaDecision{Allowed: true}
	}
	if policy.GlobalPause {
		return drwaDecision{DenialCode: errDRWATokenPaused}
	}
	if holder == nil || !strings.EqualFold(holder.KYCStatus, "approved") {
		return drwaDecision{DenialCode: errDRWAKYCRequiredSender}
	}
	// Deny-by-default for AML — only "clear" or "approved" passes.
	if !strings.EqualFold(holder.AMLStatus, "clear") && !strings.EqualFold(holder.AMLStatus, "approved") {
		return drwaDecision{DenialCode: errDRWAAMLBlockedSender}
	}
	// If now==0 (blockchain hook not yet initialized) and holder has
	// an expiry set, deny by default — cannot validate expiry without a valid round.
	if now == 0 && (holder.IdentityExpiryRound > 0 || holder.ExpiryRound > 0) {
		return drwaDecision{DenialCode: errDRWAAssetExpired}
	}
	if holder.IdentityExpiryRound > 0 && now > holder.IdentityExpiryRound {
		return drwaDecision{DenialCode: errDRWAAssetExpired}
	}
	if holder.ExpiryRound > 0 && now > holder.ExpiryRound {
		return drwaDecision{DenialCode: errDRWAAssetExpired}
	}
	if holder.TransferLocked {
		return drwaDecision{DenialCode: errDRWATransferLocked}
	}
	if len(policy.AllowedInvestorClasses) > 0 && !policy.AllowedInvestorClasses[holder.InvestorClass] {
		return drwaDecision{DenialCode: errDRWAInvestorClass}
	}
	if len(policy.AllowedJurisdictions) > 0 && !policy.AllowedJurisdictions[holder.JurisdictionCode] {
		return drwaDecision{DenialCode: errDRWAJurisdiction}
	}
	// Strict auditor mode requires holder to have auditor
	// authorization for transfers, not just metadata updates.
	if policy.StrictAuditorMode && !holder.AuditorAuthorized {
		return drwaDecision{DenialCode: errDRWAAuditorRequired}
	}

	return drwaDecision{Allowed: true}
}

func validateDRWAReceiver(policy *drwaTokenPolicyView, holder *drwaHolderMirrorView, now uint64) drwaDecision {
	if policy == nil || !policy.DRWAEnabled {
		return drwaDecision{Allowed: true}
	}
	if policy.GlobalPause {
		return drwaDecision{DenialCode: errDRWATokenPaused}
	}
	if holder == nil || !strings.EqualFold(holder.KYCStatus, "approved") {
		return drwaDecision{DenialCode: errDRWAKYCRequiredReceiver}
	}
	if !strings.EqualFold(holder.AMLStatus, "clear") && !strings.EqualFold(holder.AMLStatus, "approved") {
		return drwaDecision{DenialCode: errDRWAAMLBlockedReceiver}
	}
	// Deny-by-default when round unknown and expiry is set (receiver side).
	if now == 0 && (holder.IdentityExpiryRound > 0 || holder.ExpiryRound > 0) {
		return drwaDecision{DenialCode: errDRWAAssetExpired}
	}
	if holder.IdentityExpiryRound > 0 && now > holder.IdentityExpiryRound {
		return drwaDecision{DenialCode: errDRWAAssetExpired}
	}
	if holder.ExpiryRound > 0 && now > holder.ExpiryRound {
		return drwaDecision{DenialCode: errDRWAAssetExpired}
	}
	if holder.ReceiveLocked {
		return drwaDecision{DenialCode: errDRWAReceiveLocked}
	}
	if len(policy.AllowedInvestorClasses) > 0 && !policy.AllowedInvestorClasses[holder.InvestorClass] {
		return drwaDecision{DenialCode: errDRWAInvestorClass}
	}
	if len(policy.AllowedJurisdictions) > 0 && !policy.AllowedJurisdictions[holder.JurisdictionCode] {
		return drwaDecision{DenialCode: errDRWAJurisdiction}
	}
	// Receiver must also have auditor authorization when
	// strict_auditor_mode is enabled.
	if policy.StrictAuditorMode && !holder.AuditorAuthorized {
		return drwaDecision{DenialCode: errDRWAAuditorRequired}
	}

	return drwaDecision{Allowed: true}
}

func validateDRWAMetadataUpdate(policy *drwaTokenPolicyView, auditorAuthorized bool) drwaDecision {
	if policy == nil || !policy.DRWAEnabled {
		return drwaDecision{Allowed: true}
	}
	if !policy.MetadataProtectionEnabled {
		return drwaDecision{Allowed: true}
	}
	if policy.StrictAuditorMode && !auditorAuthorized {
		return drwaDecision{DenialCode: errDRWAAuditorRequired}
	}

	return drwaDecision{Allowed: true}
}

func evaluateDRWASenderTransfer(reader drwaStateReader, tokenID []byte, senderAddr []byte, senderAccount vmcommon.UserAccountHandler, now uint64) (bool, error) {
	regulated, policy, err := isDRWARegulatedToken(reader, tokenID)
	if err != nil || !regulated {
		return regulated, err
	}

	holder, err := reader.GetHolderMirror(tokenID, senderAddr, senderAccount)
	if err != nil {
		return true, err
	}

	decision := validateDRWASender(policy, holder, now)
	if !decision.Allowed {
		if m := drwaDenialMetric(decision.DenialCode); m != "" {
			recordDRWAGateMetric(m)
		}
		return true, decision.DenialCode
	}

	return true, nil
}

func checkDRWASenderTransfer(reader drwaStateReader, tokenID []byte, senderAddr []byte, senderAccount vmcommon.UserAccountHandler, now uint64) error {
	_, err := evaluateDRWASenderTransfer(reader, tokenID, senderAddr, senderAccount, now)
	return err
}

func evaluateDRWAReceiverTransfer(reader drwaStateReader, tokenID []byte, receiverAddr []byte, receiverAccount vmcommon.UserAccountHandler, now uint64) (bool, error) {
	regulated, policy, err := isDRWARegulatedToken(reader, tokenID)
	if err != nil || !regulated {
		return regulated, err
	}

	holder, err := reader.GetHolderMirror(tokenID, receiverAddr, receiverAccount)
	if err != nil {
		return true, err
	}

	decision := validateDRWAReceiver(policy, holder, now)
	if !decision.Allowed {
		if m := drwaDenialMetric(decision.DenialCode); m != "" {
			recordDRWAGateMetric(m)
		}
		return true, decision.DenialCode
	}

	return true, nil
}

func checkDRWAReceiverTransfer(reader drwaStateReader, tokenID []byte, receiverAddr []byte, receiverAccount vmcommon.UserAccountHandler, now uint64) error {
	_, err := evaluateDRWAReceiverTransfer(reader, tokenID, receiverAddr, receiverAccount, now)
	return err
}

func evaluateDRWAMetadataUpdate(reader drwaStateReader, tokenID []byte, callerAddr []byte, callerAccount vmcommon.UserAccountHandler) (bool, error) {
	regulated, policy, err := isDRWARegulatedToken(reader, tokenID)
	if err != nil || !regulated {
		return regulated, err
	}

	holder, err := reader.GetHolderMirror(tokenID, callerAddr, callerAccount)
	if err != nil {
		return true, err
	}

	auditorAuthorized := holder != nil && holder.AuditorAuthorized
	decision := validateDRWAMetadataUpdate(policy, auditorAuthorized)
	if !decision.Allowed {
		if m := drwaDenialMetric(decision.DenialCode); m != "" {
			recordDRWAGateMetric(m)
		}
		return true, decision.DenialCode
	}

	return true, nil
}

func checkDRWAMetadataUpdate(reader drwaStateReader, tokenID []byte, callerAddr []byte, callerAccount vmcommon.UserAccountHandler) error {
	_, err := evaluateDRWAMetadataUpdate(reader, tokenID, callerAddr, callerAccount)
	return err
}
