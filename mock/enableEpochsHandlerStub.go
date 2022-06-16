package mock

// EnableEpochsHandlerStub -
type EnableEpochsHandlerStub struct {
	IsESDTMultiTransferFlagEnabledField                  bool
	IsGlobalMintBurnFlagEnabledField                     bool
	IsESDTTransferRoleFlagEnabledField                   bool
	IsBuiltInFunctionOnMetaFlagEnabledField              bool
	IsOptimizeNFTStoreFlagEnabledField                   bool
	IsCheckCorrectTokenIDForTransferRoleFlagEnabledField bool
	IsESDTMetadataContinuousCleanupFlagEnabledField      bool
}

// IsESDTMultiTransferFlagEnabled -
func (stub *EnableEpochsHandlerStub) IsESDTMultiTransferFlagEnabled() bool {
	return stub.IsESDTMultiTransferFlagEnabledField
}

// IsGlobalMintBurnFlagEnabled -
func (stub *EnableEpochsHandlerStub) IsGlobalMintBurnFlagEnabled() bool {
	return stub.IsGlobalMintBurnFlagEnabledField
}

// IsESDTTransferRoleFlagEnabled -
func (stub *EnableEpochsHandlerStub) IsESDTTransferRoleFlagEnabled() bool {
	return stub.IsESDTTransferRoleFlagEnabledField
}

// IsBuiltInFunctionOnMetaFlagEnabled -
func (stub *EnableEpochsHandlerStub) IsBuiltInFunctionOnMetaFlagEnabled() bool {
	return stub.IsBuiltInFunctionOnMetaFlagEnabledField
}

// IsOptimizeNFTStoreFlagEnabled -
func (stub *EnableEpochsHandlerStub) IsOptimizeNFTStoreFlagEnabled() bool {
	return stub.IsOptimizeNFTStoreFlagEnabledField
}

// IsCheckCorrectTokenIDForTransferRoleFlagEnabled -
func (stub *EnableEpochsHandlerStub) IsCheckCorrectTokenIDForTransferRoleFlagEnabled() bool {
	return stub.IsCheckCorrectTokenIDForTransferRoleFlagEnabledField
}

// IsESDTMetadataContinuousCleanupFlagEnabled -
func (stub *EnableEpochsHandlerStub) IsESDTMetadataContinuousCleanupFlagEnabled() bool {
	return stub.IsESDTMetadataContinuousCleanupFlagEnabledField
}

// IsInterfaceNil -
func (stub *EnableEpochsHandlerStub) IsInterfaceNil() bool {
	return stub == nil
}
