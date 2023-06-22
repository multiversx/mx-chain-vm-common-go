package dataTrieMigrator

import (
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/stretchr/testify/assert"
)

func getDefaultArgs() ArgsNewDataTrieMigrator {
	return ArgsNewDataTrieMigrator{
		GasProvided: 15,
		DataTrieGasCost: DataTrieGasCost{
			TrieLoadPerNode:  2,
			TrieStorePerNode: 5,
		},
	}
}

func TestNewDataTrieMigrator(t *testing.T) {
	t.Parallel()

	dtm := NewDataTrieMigrator(getDefaultArgs())
	assert.False(t, dtm.IsInterfaceNil())
}

func TestConsumeStorageLoadGas(t *testing.T) {
	t.Parallel()

	args := getDefaultArgs()
	args.TrieLoadPerNode = 6
	dtm := NewDataTrieMigrator(args)
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

		args := getDefaultArgs()
		dtm := NewDataTrieMigrator(args)

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

		args := getDefaultArgs()
		dtm := NewDataTrieMigrator(args)

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

		args := getDefaultArgs()
		args.GasProvided = 11
		dtm := NewDataTrieMigrator(args)

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

	args := getDefaultArgs()
	args.GasProvided = 11
	dtm := NewDataTrieMigrator(args)
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

	args := getDefaultArgs()
	args.GasProvided = 11
	dtm := NewDataTrieMigrator(args)
	assert.Equal(t, uint64(11), dtm.GetGasRemaining())
	dtm.gasRemaining = 5
	assert.Equal(t, uint64(5), dtm.GetGasRemaining())
}
