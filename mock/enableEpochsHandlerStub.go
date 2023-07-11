package mock

// EnableEpochsHandlerStub -
type EnableEpochsHandlerStub struct {
	IsGlobalMintBurnFlagEnabledField                     bool
	IsESDTTransferRoleFlagEnabledField                   bool
	IsCheckCorrectTokenIDForTransferRoleFlagEnabledField bool
	IsCheckFunctionArgumentFlagEnabledField              bool
	IsFixAsyncCallbackCheckFlagEnabledField              bool
	IsSaveToSystemAccountFlagEnabledField                bool
	IsCheckFrozenCollectionFlagEnabledField              bool
	IsSendAlwaysFlagEnabledField                         bool
	IsValueLengthCheckFlagEnabledField                   bool
	IsCheckTransferFlagEnabledField                      bool
	IsTransferToMetaFlagEnabledField                     bool
	IsESDTNFTImprovementV1FlagEnabledField               bool
	IsFixOldTokenLiquidityEnabledField                   bool
	IsWipeSingleNFTLiquidityDecreaseEnabledField         bool
	IsAlwaysSaveTokenMetaDataEnabledField                bool
	IsChangeUsernameEnabledEpochField                    bool
	IsSetGuardianEnabledField                            bool
	IsConsistentTokensValuesLengthCheckEnabledField      bool
	IsAutoBalanceDataTriesEnabledField                   bool
	CurrentEpochField                                    uint32
}

// IsGlobalMintBurnFlagEnabledInEpoch -
func (stub *EnableEpochsHandlerStub) IsGlobalMintBurnFlagEnabledInEpoch(_ uint32) bool {
	return stub.IsGlobalMintBurnFlagEnabledField
}

// IsESDTTransferRoleFlagEnabledInEpoch -
func (stub *EnableEpochsHandlerStub) IsESDTTransferRoleFlagEnabledInEpoch(_ uint32) bool {
	return stub.IsESDTTransferRoleFlagEnabledField
}

// IsCheckCorrectTokenIDForTransferRoleFlagEnabledInEpoch -
func (stub *EnableEpochsHandlerStub) IsCheckCorrectTokenIDForTransferRoleFlagEnabledInEpoch(_ uint32) bool {
	return stub.IsCheckCorrectTokenIDForTransferRoleFlagEnabledField
}

// IsCheckFunctionArgumentFlagEnabledInEpoch -
func (stub *EnableEpochsHandlerStub) IsCheckFunctionArgumentFlagEnabledInEpoch(_ uint32) bool {
	return stub.IsCheckFunctionArgumentFlagEnabledField
}

// IsFixAsyncCallbackCheckFlagEnabledInEpoch -
func (stub *EnableEpochsHandlerStub) IsFixAsyncCallbackCheckFlagEnabledInEpoch(_ uint32) bool {
	return stub.IsFixAsyncCallbackCheckFlagEnabledField
}

// IsSaveToSystemAccountFlagEnabledInEpoch -
func (stub *EnableEpochsHandlerStub) IsSaveToSystemAccountFlagEnabledInEpoch(_ uint32) bool {
	return stub.IsSaveToSystemAccountFlagEnabledField
}

// IsCheckFrozenCollectionFlagEnabledInEpoch -
func (stub *EnableEpochsHandlerStub) IsCheckFrozenCollectionFlagEnabledInEpoch(_ uint32) bool {
	return stub.IsCheckFrozenCollectionFlagEnabledField
}

// IsSendAlwaysFlagEnabledInEpoch -
func (stub *EnableEpochsHandlerStub) IsSendAlwaysFlagEnabledInEpoch(_ uint32) bool {
	return stub.IsSendAlwaysFlagEnabledField
}

// IsValueLengthCheckFlagEnabledInEpoch -
func (stub *EnableEpochsHandlerStub) IsValueLengthCheckFlagEnabledInEpoch(_ uint32) bool {
	return stub.IsValueLengthCheckFlagEnabledField
}

// IsCheckTransferFlagEnabledInEpoch -
func (stub *EnableEpochsHandlerStub) IsCheckTransferFlagEnabledInEpoch(_ uint32) bool {
	return stub.IsCheckTransferFlagEnabledField
}

// IsTransferToMetaFlagEnabledInEpoch -
func (stub *EnableEpochsHandlerStub) IsTransferToMetaFlagEnabledInEpoch(_ uint32) bool {
	return stub.IsTransferToMetaFlagEnabledField
}

// IsESDTNFTImprovementV1FlagEnabledInEpoch -
func (stub *EnableEpochsHandlerStub) IsESDTNFTImprovementV1FlagEnabledInEpoch(_ uint32) bool {
	return stub.IsESDTNFTImprovementV1FlagEnabledField
}

// IsFixOldTokenLiquidityEnabledInEpoch -
func (stub *EnableEpochsHandlerStub) IsFixOldTokenLiquidityEnabledInEpoch(_ uint32) bool {
	return stub.IsFixOldTokenLiquidityEnabledField
}

// IsWipeSingleNFTLiquidityDecreaseEnabledInEpoch -
func (stub *EnableEpochsHandlerStub) IsWipeSingleNFTLiquidityDecreaseEnabledInEpoch(_ uint32) bool {
	return stub.IsWipeSingleNFTLiquidityDecreaseEnabledField
}

// IsAlwaysSaveTokenMetaDataEnabledInEpoch -
func (stub *EnableEpochsHandlerStub) IsAlwaysSaveTokenMetaDataEnabledInEpoch(_ uint32) bool {
	return stub.IsAlwaysSaveTokenMetaDataEnabledField
}

// IsChangeUsernameEnabledInEpoch -
func (stub *EnableEpochsHandlerStub) IsChangeUsernameEnabledInEpoch(_ uint32) bool {
	return stub.IsChangeUsernameEnabledEpochField
}

// IsSetGuardianEnabledInEpoch -
func (stub *EnableEpochsHandlerStub) IsSetGuardianEnabledInEpoch(_ uint32) bool {
	return stub.IsSetGuardianEnabledField
}

// IsConsistentTokensValuesLengthCheckEnabledInEpoch -
func (stub *EnableEpochsHandlerStub) IsConsistentTokensValuesLengthCheckEnabledInEpoch(_ uint32) bool {
	return stub.IsConsistentTokensValuesLengthCheckEnabledField
}

// IsAutoBalanceDataTriesEnabledInEpoch -
func (stub *EnableEpochsHandlerStub) IsAutoBalanceDataTriesEnabledInEpoch(_ uint32) bool {
	return stub.IsAutoBalanceDataTriesEnabledField
}

// GetCurrentEpoch -
func (stub *EnableEpochsHandlerStub) GetCurrentEpoch() uint32 {
	return stub.CurrentEpochField
}

// IsInterfaceNil -
func (stub *EnableEpochsHandlerStub) IsInterfaceNil() bool {
	return stub == nil
}
