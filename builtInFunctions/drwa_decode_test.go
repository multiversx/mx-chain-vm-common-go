package builtInFunctions

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"testing"

	teststate "github.com/multiversx/mx-chain-go/testscommon/state"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/require"
)

func TestNewDRWAAccountsReaderRejectsNilAccounts(t *testing.T) {
	t.Parallel()

	reader, err := newDRWAAccountsReader(nil)
	require.Nil(t, reader)
	require.EqualError(t, err, "nil DRWA accounts adapter")
}

func TestDRWAAccountsReaderLoadUserAccountUsesCurrentAccount(t *testing.T) {
	t.Parallel()

	current := mock.NewUserAccount([]byte("holder"))
	reader, err := newDRWAAccountsReader(&mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			t.Fatalf("load should not be called for current account")
			return nil, nil
		},
	})
	require.NoError(t, err)

	loaded, err := reader.loadUserAccount([]byte("holder"), current)
	require.NoError(t, err)
	require.Same(t, current, loaded)
	require.False(t, reader.IsInterfaceNil())

	var nilReader *drwaAccountsReader
	require.True(t, nilReader.IsInterfaceNil())
}

func TestDRWAAccountsReaderLoadUserAccountRejectsWrongType(t *testing.T) {
	t.Parallel()

	reader, err := newDRWAAccountsReader(&mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return &teststate.StateUserAccountHandlerStub{}, nil
		},
	})
	require.NoError(t, err)

	loaded, err := reader.loadUserAccount([]byte("holder"), nil)
	require.Nil(t, loaded)
	require.ErrorIs(t, err, ErrWrongTypeAssertion)
}

func TestDecodeDRWAStoredJSONUsesWrappedBody(t *testing.T) {
	t.Parallel()

	wrapped, err := json.Marshal(&drwaStoredValue{
		Version: 3,
		Body:    []byte(`{"drwa_enabled":true}`),
	})
	require.NoError(t, err)

	view := &drwaTokenPolicyView{}
	require.NoError(t, decodeDRWAStoredJSON(wrapped, view))
	require.True(t, view.DRWAEnabled)
}

func TestDecodeDRWAStoredJSONRejectsMalformedWrapperInsteadOfFallingBackToRawBody(t *testing.T) {
	t.Parallel()

	view := &drwaTokenPolicyView{}
	require.Error(t, decodeDRWAStoredJSON([]byte(`{"drwa_enabled":true}`), view))
}

func TestDecodeDRWABodyBinaryProfileAndAuthorization(t *testing.T) {
	t.Parallel()

	profilePayload := make([]byte, 0, 64)
	profilePayload = append(profilePayload, make([]byte, 8)...)
	profilePayload = appendLenPrefixed(profilePayload, []byte("approved"))
	profilePayload = appendLenPrefixed(profilePayload, []byte("approved"))
	profilePayload = appendLenPrefixed(profilePayload, []byte("accredited"))
	profilePayload = appendLenPrefixed(profilePayload, []byte("SG"))
	expiry := make([]byte, 8)
	binary.BigEndian.PutUint64(expiry, 42)
	profilePayload = append(profilePayload, expiry...)

	profile := &drwaHolderProfileView{}
	require.NoError(t, decodeDRWABody(profilePayload, profile))
	require.Equal(t, "approved", profile.KYCStatus)
	require.Equal(t, "approved", profile.AMLStatus)
	require.Equal(t, "accredited", profile.InvestorClass)
	require.Equal(t, "SG", profile.JurisdictionCode)
	require.Equal(t, uint64(42), profile.ExpiryRound)

	authPayload := make([]byte, 9)
	authPayload[8] = 1
	auth := &drwaHolderAuditorAuthorizationView{}
	require.NoError(t, decodeDRWABody(authPayload, auth))
	require.True(t, auth.AuditorAuthorized)

	// T-5: Non-zero bytes in positions 0-7 must be ignored; only byte 8 determines authorization.
	authPayloadNonZeroPrefix := make([]byte, 9)
	authPayloadNonZeroPrefix[0] = 0xFF
	authPayloadNonZeroPrefix[1] = 0xAB
	authPayloadNonZeroPrefix[2] = 0xCD
	authPayloadNonZeroPrefix[3] = 0xEF
	authPayloadNonZeroPrefix[4] = 0x12
	authPayloadNonZeroPrefix[5] = 0x34
	authPayloadNonZeroPrefix[6] = 0x56
	authPayloadNonZeroPrefix[7] = 0x78
	authPayloadNonZeroPrefix[8] = 1 // authorized
	authNonZero := &drwaHolderAuditorAuthorizationView{}
	require.NoError(t, decodeDRWABody(authPayloadNonZeroPrefix, authNonZero))
	require.True(t, authNonZero.AuditorAuthorized, "non-zero prefix bytes 0-7 must be ignored")

	// Same prefix but byte 8 = 0 → not authorized
	authPayloadNonZeroPrefix[8] = 0
	authNotAuthorized := &drwaHolderAuditorAuthorizationView{}
	require.NoError(t, decodeDRWABody(authPayloadNonZeroPrefix, authNotAuthorized))
	require.False(t, authNotAuthorized.AuditorAuthorized, "byte 8 = 0 must yield unauthorized regardless of prefix")
}

func TestDecodeDRWABodyRejectsBrokenJSONWithoutBinaryFallback(t *testing.T) {
	t.Parallel()

	tokenPolicy := &drwaTokenPolicyView{}
	err := decodeDRWABody([]byte("{not-json"), tokenPolicy)
	require.Error(t, err)
}

func TestDecodeDRWABodyDoesNotTreatAccidentallyValidJSONScalarAsStructuredState(t *testing.T) {
	t.Parallel()

	holder := &drwaHolderMirrorView{}
	err := decodeDRWABody([]byte("0"), holder)
	require.Error(t, err)
}

func TestDecodeDRWABodyFallsBackToJSONForUnknownDestination(t *testing.T) {
	t.Parallel()

	destination := &struct {
		Value string `json:"value"`
	}{}

	require.NoError(t, decodeDRWABody([]byte(`{"value":"ok"}`), destination))
	require.Equal(t, "ok", destination.Value)
}

// NOTE: This test must NOT use t.Parallel() — it mutates shared package-level metric counters.
func TestDecodeDRWAStoredJSONRecordsFailureMetricsByType(t *testing.T) {
	resetDRWAGateMetrics()

	require.Error(t, decodeDRWAStoredJSON([]byte("{not-json"), &drwaTokenPolicyView{}))
	require.Error(t, decodeDRWAStoredJSON([]byte{}, &drwaHolderMirrorView{}))
	require.Error(t, decodeDRWAStoredJSON([]byte{0, 0, 0, 0}, &drwaHolderProfileView{}))

	snapshot := SnapshotDRWAGateMetrics()
	require.Equal(t, uint64(3), snapshot[drwaGateMetricDecodeFailure])
	require.Equal(t, uint64(1), snapshot[drwaGateMetricDecodeFailureJSON])
	require.Equal(t, uint64(1), snapshot[drwaGateMetricDecodeFailureMissing])
	require.Equal(t, uint64(1), snapshot[drwaGateMetricDecodeFailureBinary])
}

func TestDecodeDRWABinaryTokenPolicyRejectsShortPayload(t *testing.T) {
	t.Parallel()

	view := &drwaTokenPolicyView{}
	require.Error(t, decodeDRWABinaryTokenPolicy(make([]byte, 11), view))
}

func TestDecodeDRWABinaryHolderMirrorRejectsShortPayload(t *testing.T) {
	t.Parallel()

	view := &drwaHolderMirrorView{}
	require.Error(t, decodeDRWABinaryHolderMirror(make([]byte, 13), view))
}

func TestReadDRWABinaryFieldRejectsShortBodies(t *testing.T) {
	t.Parallel()

	_, _, err := readDRWABinaryField([]byte{0, 0, 0}, 0)
	require.Error(t, err)

	profile := &drwaHolderProfileView{}
	err = decodeDRWABinaryHolderProfile(make([]byte, 8), profile)
	require.Error(t, err)

	auth := &drwaHolderAuditorAuthorizationView{}
	err = decodeDRWABinaryHolderAuditorAuthorization(make([]byte, 8), auth)
	require.Error(t, err)
}

func TestDecodeDRWABinaryHolderMirrorRejectsFieldAndTrailerCorruption(t *testing.T) {
	t.Parallel()

	prefixOnly := make([]byte, 8)
	err := decodeDRWABinaryHolderMirror(prefixOnly, &drwaHolderMirrorView{})
	require.Error(t, err)

	invalidFieldBody := make([]byte, 12)
	binary.BigEndian.PutUint32(invalidFieldBody[8:12], 4)
	err = decodeDRWABinaryHolderMirror(invalidFieldBody, &drwaHolderMirrorView{})
	require.Error(t, err)

	shortTrailer := make([]byte, 0, 40)
	shortTrailer = append(shortTrailer, make([]byte, 8)...)
	shortTrailer = appendLenPrefixed(shortTrailer, []byte("approved"))
	shortTrailer = appendLenPrefixed(shortTrailer, []byte("approved"))
	shortTrailer = appendLenPrefixed(shortTrailer, []byte("qib"))
	shortTrailer = appendLenPrefixed(shortTrailer, []byte("US"))
	shortTrailer = append(shortTrailer, make([]byte, 10)...)
	err = decodeDRWABinaryHolderMirror(shortTrailer, &drwaHolderMirrorView{})
	require.Error(t, err)
}

func TestDecodeDRWABinaryHolderProfileRejectsFieldAndTrailerCorruption(t *testing.T) {
	t.Parallel()

	prefixOnly := make([]byte, 8)
	err := decodeDRWABinaryHolderProfile(prefixOnly, &drwaHolderProfileView{})
	require.Error(t, err)

	invalidFieldBody := make([]byte, 12)
	binary.BigEndian.PutUint32(invalidFieldBody[8:12], 4)
	err = decodeDRWABinaryHolderProfile(invalidFieldBody, &drwaHolderProfileView{})
	require.Error(t, err)

	shortTrailer := make([]byte, 0, 40)
	shortTrailer = append(shortTrailer, make([]byte, 8)...)
	shortTrailer = appendLenPrefixed(shortTrailer, []byte("approved"))
	shortTrailer = appendLenPrefixed(shortTrailer, []byte("approved"))
	shortTrailer = appendLenPrefixed(shortTrailer, []byte("qib"))
	shortTrailer = appendLenPrefixed(shortTrailer, []byte("US"))
	shortTrailer = append(shortTrailer, make([]byte, 7)...)
	err = decodeDRWABinaryHolderProfile(shortTrailer, &drwaHolderProfileView{})
	require.Error(t, err)
}

func TestIsDRWAEnforcementEnabledAndReadGasCost(t *testing.T) {
	t.Parallel()

	require.False(t, isDRWAEnforcementEnabled(nil))
	require.True(t, isDRWAEnforcementEnabled(drwaEnabledEpochsHandler()))
	require.Equal(t, uint64(0), computeDRWAReadGasCost(vmcommon.BaseOperationCost{}, 7, 0))
	require.Equal(t, uint64(0), computeDRWAReadGasCost(vmcommon.BaseOperationCost{}, 0, 3))
	require.Equal(t, uint64(21), computeDRWAReadGasCost(vmcommon.BaseOperationCost{StorePerByte: 5}, 7, 3))
	require.Equal(t, uint64(21), computeDRWAReadGasCost(vmcommon.BaseOperationCost{}, 7, 3))
}

func TestComputeDRWAReadGasCostOverflow(t *testing.T) {
	t.Parallel()

	// When fallbackCost * reads * drwaReadGasUnits would overflow uint64, should return MaxUint64.
	result := computeDRWAReadGasCost(vmcommon.BaseOperationCost{}, math.MaxUint64, 2)
	require.Equal(t, uint64(math.MaxUint64), result)

	// Normal case should compute correctly: fallbackCost=100, reads=3, drwaReadGasUnits=1 => 300
	result = computeDRWAReadGasCost(vmcommon.BaseOperationCost{}, 100, 3)
	require.Equal(t, uint64(100*3*drwaReadGasUnits), result)
}

func TestReadDRWABinaryFieldRejectsOversizedLength(t *testing.T) {
	t.Parallel()

	// Construct a payload where the 4-byte length prefix exceeds 64*1024 (drwaSyncMaxFieldBytes cap).
	oversizedLength := uint32(64*1024 + 1)
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, oversizedLength)

	_, _, err := readDRWABinaryField(payload, 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), fmt.Sprintf("DRWA binary field length %d exceeds max 65536", oversizedLength))
}

func appendLenPrefixed(buffer []byte, value []byte) []byte {
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(len(value)))
	buffer = append(buffer, length...)
	buffer = append(buffer, value...)
	return buffer
}
