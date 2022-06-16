package builtInFunctions

import (
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

const (
	defaultFlag                       = "defaultFlag"
	esdtTransferRoleFlag              = "esdtTransferRoleFlag"
	esdtMultiTransferFlag             = "esdtMultiTransferFlag"
	esdtMetadataContinuousCleanupFlag = "esdtMetadataContinuousCleanupFlag"
)

type baseAlwaysActive struct {
}

// IsActive returns true as this built in function was always active
func (b baseAlwaysActive) IsActive() bool {
	return true
}

// IsInterfaceNil returns true if there is no value under the interface
func (b baseAlwaysActive) IsInterfaceNil() bool {
	return false
}

type baseEnabled struct {
	function            string
	activationFlagName  string
	enableEpochsHandler vmcommon.EnableEpochsHandler
}

// IsActive returns true if function is activated
func (b *baseEnabled) IsActive() bool {
	switch b.activationFlagName {
	case defaultFlag:
		return true
	case esdtTransferRoleFlag:
		return b.enableEpochsHandler.IsESDTTransferRoleFlagEnabled()
	case esdtMultiTransferFlag:
		return b.enableEpochsHandler.IsESDTMultiTransferFlagEnabled()
	case esdtMetadataContinuousCleanupFlag:
		return b.enableEpochsHandler.IsESDTMetadataContinuousCleanupFlagEnabled()
	}

	return false
}

// IsInterfaceNil returns true if there is no value under the interface
func (b *baseEnabled) IsInterfaceNil() bool {
	return b == nil
}

type baseDisabled struct {
	function            string
	enableEpochsHandler vmcommon.EnableEpochsHandler
}

// IsActive returns true if function is activated
func (b *baseDisabled) IsActive() bool {
	return b.enableEpochsHandler.IsGlobalMintBurnFlagEnabled()
}

// IsInterfaceNil returns true if there is no value under the interface
func (b *baseDisabled) IsInterfaceNil() bool {
	return b == nil
}
