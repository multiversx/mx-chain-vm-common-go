package builtInFunctions

import (
	"errors"
	"testing"
)

func TestDRWAMetricConstants_NonEmpty(t *testing.T) {
	constants := []struct {
		name  string
		value string
	}{
		{"drwaGateMetricDeniedPaused", drwaGateMetricDeniedPaused},
		{"drwaGateMetricDeniedKYCSender", drwaGateMetricDeniedKYCSender},
		{"drwaGateMetricDeniedAMLSender", drwaGateMetricDeniedAMLSender},
		{"drwaGateMetricDeniedExpiry", drwaGateMetricDeniedExpiry},
		{"drwaGateMetricDeniedTransferLock", drwaGateMetricDeniedTransferLock},
		{"drwaGateMetricDeniedKYCReceiver", drwaGateMetricDeniedKYCReceiver},
		{"drwaGateMetricDeniedAMLReceiver", drwaGateMetricDeniedAMLReceiver},
		{"drwaGateMetricDeniedReceiveLock", drwaGateMetricDeniedReceiveLock},
		{"drwaGateMetricDeniedClass", drwaGateMetricDeniedClass},
		{"drwaGateMetricDeniedJurisdiction", drwaGateMetricDeniedJurisdiction},
		{"drwaGateMetricDeniedAuditor", drwaGateMetricDeniedAuditor},
		{"drwaGateMetricDecodeFailure", drwaGateMetricDecodeFailure},
		{"drwaGateMetricDecodeFailureJSON", drwaGateMetricDecodeFailureJSON},
		{"drwaGateMetricDecodeFailureBinary", drwaGateMetricDecodeFailureBinary},
		{"drwaGateMetricDecodeFailureMissing", drwaGateMetricDecodeFailureMissing},
	}
	for _, c := range constants {
		if c.value == "" {
			t.Errorf("metric constant %s must not be empty", c.name)
		}
	}
}

func TestDRWAMetricConstants_NoDuplicates(t *testing.T) {
	values := []string{
		drwaGateMetricDeniedPaused,
		drwaGateMetricDeniedKYCSender,
		drwaGateMetricDeniedAMLSender,
		drwaGateMetricDeniedExpiry,
		drwaGateMetricDeniedTransferLock,
		drwaGateMetricDeniedKYCReceiver,
		drwaGateMetricDeniedAMLReceiver,
		drwaGateMetricDeniedReceiveLock,
		drwaGateMetricDeniedClass,
		drwaGateMetricDeniedJurisdiction,
		drwaGateMetricDeniedAuditor,
		drwaGateMetricDecodeFailure,
		drwaGateMetricDecodeFailureJSON,
		drwaGateMetricDecodeFailureBinary,
		drwaGateMetricDecodeFailureMissing,
	}
	seen := make(map[string]int)
	for i, v := range values {
		if prev, exists := seen[v]; exists {
			t.Errorf("duplicate metric value %q at indices %d and %d", v, prev, i)
		}
		seen[v] = i
	}
}

func TestSnapshotDRWAGateMetrics_Smoke(t *testing.T) {
	resetDRWAGateMetrics()
	snap := SnapshotDRWAGateMetrics()
	if snap == nil {
		t.Fatal("SnapshotDRWAGateMetrics returned nil")
	}
	if len(snap) != 0 {
		t.Fatalf("expected empty snapshot after reset, got %d entries", len(snap))
	}
}

func TestRecordDRWAGateMetric_IncrementsCorrectCounter(t *testing.T) {
	resetDRWAGateMetrics()

	recordDRWAGateMetric(drwaGateMetricDeniedPaused)
	recordDRWAGateMetric(drwaGateMetricDeniedPaused)
	recordDRWAGateMetric(drwaGateMetricDeniedKYCSender)

	snap := SnapshotDRWAGateMetrics()
	if snap[drwaGateMetricDeniedPaused] != 2 {
		t.Errorf("expected %s=2, got %d", drwaGateMetricDeniedPaused, snap[drwaGateMetricDeniedPaused])
	}
	if snap[drwaGateMetricDeniedKYCSender] != 1 {
		t.Errorf("expected %s=1, got %d", drwaGateMetricDeniedKYCSender, snap[drwaGateMetricDeniedKYCSender])
	}
	// Other counters must remain absent.
	if snap[drwaGateMetricDeniedExpiry] != 0 {
		t.Errorf("expected %s=0, got %d", drwaGateMetricDeniedExpiry, snap[drwaGateMetricDeniedExpiry])
	}
}

func TestDrwaDenialMetric_KnownCodes(t *testing.T) {
	cases := []struct {
		code     error
		expected string
	}{
		{errDRWATokenPaused, drwaGateMetricDeniedPaused},
		{errDRWAKYCRequiredSender, drwaGateMetricDeniedKYCSender},
		{errDRWAAMLBlockedSender, drwaGateMetricDeniedAMLSender},
		{errDRWAAssetExpired, drwaGateMetricDeniedExpiry},
		{errDRWATransferLocked, drwaGateMetricDeniedTransferLock},
		{errDRWAKYCRequiredReceiver, drwaGateMetricDeniedKYCReceiver},
		{errDRWAAMLBlockedReceiver, drwaGateMetricDeniedAMLReceiver},
		{errDRWAReceiveLocked, drwaGateMetricDeniedReceiveLock},
		{errDRWAInvestorClass, drwaGateMetricDeniedClass},
		{errDRWAJurisdiction, drwaGateMetricDeniedJurisdiction},
		{errDRWAAuditorRequired, drwaGateMetricDeniedAuditor},
	}
	for _, tc := range cases {
		result := drwaDenialMetric(tc.code)
		if result == "" {
			t.Errorf("drwaDenialMetric(%v) returned empty string, expected %q", tc.code, tc.expected)
		}
		if result != tc.expected {
			t.Errorf("drwaDenialMetric(%v) = %q, expected %q", tc.code, result, tc.expected)
		}
	}
}

func TestDrwaDenialMetric_UnknownCodeReturnsEmpty(t *testing.T) {
	unknown := errors.New("UNKNOWN_ERROR")
	result := drwaDenialMetric(unknown)
	if result != "" {
		t.Errorf("drwaDenialMetric(unknown) = %q, expected empty string", result)
	}
}

func TestDrwaDenialMetric_NilCodeReturnsEmpty(t *testing.T) {
	result := drwaDenialMetric(nil)
	if result != "" {
		t.Errorf("drwaDenialMetric(nil) = %q, expected empty string", result)
	}
}
