package dataTrieMigrator

import (
	"strings"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	"github.com/stretchr/testify/assert"
)

func TestNewDataTrieMigrator(t *testing.T) {
	t.Parallel()

	t.Run("gas provided is not enough", func(t *testing.T) {
		t.Parallel()

		dtm, err := NewDataTrieMigrator(5, vmcommon.BuiltInCost{TrieLoadPerLeaf: 10})
		assert.True(t, strings.Contains(err.Error(), "not enough gas"))
		assert.True(t, dtm.IsInterfaceNil())
	})

	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		dtm, err := NewDataTrieMigrator(10, vmcommon.BuiltInCost{TrieLoadPerLeaf: 5})
		assert.Nil(t, err)
		assert.False(t, dtm.IsInterfaceNil())
	})
}

func TestConsumeStorageLoadGas(t *testing.T) {
	t.Parallel()

	dtm, _ := NewDataTrieMigrator(15, vmcommon.BuiltInCost{TrieLoadPerLeaf: 6})
	assert.True(t, dtm.ConsumeStorageLoadGas())
	assert.Equal(t, uint64(9), dtm.gasRemaining)
	assert.False(t, dtm.ConsumeStorageLoadGas())
	assert.Equal(t, uint64(3), dtm.gasRemaining)
	assert.False(t, dtm.ConsumeStorageLoadGas())
	assert.Equal(t, uint64(3), dtm.gasRemaining)
}

func TestAddLeafToMigrationQueue(t *testing.T) {
	t.Parallel()

	t.Run("migrate to NotSpecified", func(t *testing.T) {
		t.Parallel()

		dtm, _ := NewDataTrieMigrator(15, vmcommon.BuiltInCost{TrieStorePerLeaf: 5})

		leafData := core.TrieData{
			Key:     []byte("key"),
			Value:   []byte("value"),
			Version: core.AutoBalanceEnabled,
		}
		shouldContinueMigration, err := dtm.AddLeafToMigrationQueue(leafData, core.NotSpecified)
		assert.True(t, shouldContinueMigration)
		assert.Nil(t, err)
		assert.Equal(t, uint64(15), dtm.gasRemaining)
	})

	t.Run("migrate to AutoBalanceEnabled", func(t *testing.T) {
		t.Parallel()

		dtm, _ := NewDataTrieMigrator(15, vmcommon.BuiltInCost{TrieStorePerLeaf: 5})

		leafData := core.TrieData{
			Key:     []byte("key"),
			Value:   []byte("value"),
			Version: core.AutoBalanceEnabled,
		}
		shouldContinueMigration, err := dtm.AddLeafToMigrationQueue(leafData, core.AutoBalanceEnabled)
		assert.True(t, shouldContinueMigration)
		assert.Nil(t, err)
		assert.Equal(t, uint64(15), dtm.gasRemaining)
	})

	t.Run("migrate consumes gas", func(t *testing.T) {
		t.Parallel()

		dtm, _ := NewDataTrieMigrator(11, vmcommon.BuiltInCost{TrieStorePerLeaf: 5})

		leafData := core.TrieData{
			Key:     []byte("key"),
			Value:   []byte("value"),
			Version: core.NotSpecified,
		}
		shouldContinueMigration, err := dtm.AddLeafToMigrationQueue(leafData, core.AutoBalanceEnabled)
		assert.True(t, shouldContinueMigration)
		assert.Nil(t, err)
		assert.Equal(t, uint64(6), dtm.gasRemaining)

		shouldContinueMigration, err = dtm.AddLeafToMigrationQueue(leafData, core.AutoBalanceEnabled)
		assert.False(t, shouldContinueMigration)
		assert.Nil(t, err)
		assert.Equal(t, uint64(1), dtm.gasRemaining)

		shouldContinueMigration, err = dtm.AddLeafToMigrationQueue(leafData, core.AutoBalanceEnabled)
		assert.False(t, shouldContinueMigration)
		assert.Nil(t, err)
		assert.Equal(t, uint64(1), dtm.gasRemaining)
	})
}

func TestGetLeavesToBeMigrated(t *testing.T) {
	t.Parallel()

	dtm, _ := NewDataTrieMigrator(11, vmcommon.BuiltInCost{TrieStorePerLeaf: 5})
	expectedLeaves := []core.TrieData{
		{
			Key:     []byte("key1"),
			Value:   []byte("value1"),
			Version: core.NotSpecified,
		},
		{
			Key:     []byte("key2"),
			Value:   []byte("value2"),
			Version: core.NotSpecified,
		},
		{
			Key:     []byte("key3"),
			Value:   []byte("value3"),
			Version: core.NotSpecified,
		},
	}
	dtm.leavesToBeMigrated = expectedLeaves

	leaves := dtm.GetLeavesToBeMigrated()
	assert.Equal(t, expectedLeaves, leaves)
}

func TestGetGasRemaining(t *testing.T) {
	t.Parallel()

	dtm, _ := NewDataTrieMigrator(11, vmcommon.BuiltInCost{TrieStorePerLeaf: 5})
	assert.Equal(t, uint64(11), dtm.GetGasRemaining())
	dtm.gasRemaining = 5
	assert.Equal(t, uint64(5), dtm.GetGasRemaining())
}
