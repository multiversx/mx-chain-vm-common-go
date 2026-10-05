package builtInFunctions

import "github.com/multiversx/mx-chain-core-go/core"

// Enable epoch flags definitions
const (
	GlobalMintBurnFlag                          core.EnableEpochFlag = "GlobalMintBurnFlag"
	ESDTTransferRoleFlag                        core.EnableEpochFlag = "ESDTTransferRoleFlag"
	CheckFunctionArgumentFlag                   core.EnableEpochFlag = "CheckFunctionArgumentFlag"
	CheckCorrectTokenIDForTransferRoleFlag      core.EnableEpochFlag = "CheckCorrectTokenIDForTransferRoleFlag"
	FixAsyncCallbackCheckFlag                   core.EnableEpochFlag = "FixAsyncCallbackCheckFlag"
	SaveToSystemAccountFlag                     core.EnableEpochFlag = "SaveToSystemAccountFlag"
	CheckFrozenCollectionFlag                   core.EnableEpochFlag = "CheckFrozenCollectionFlag"
	SendAlwaysFlag                              core.EnableEpochFlag = "SendAlwaysFlag"
	ValueLengthCheckFlag                        core.EnableEpochFlag = "ValueLengthCheckFlag"
	CheckTransferFlag                           core.EnableEpochFlag = "CheckTransferFlag"
	ESDTNFTImprovementV1Flag                    core.EnableEpochFlag = "ESDTNFTImprovementV1Flag"
	FixOldTokenLiquidityFlag                    core.EnableEpochFlag = "FixOldTokenLiquidityFlag"
	WipeSingleNFTLiquidityDecreaseFlag          core.EnableEpochFlag = "WipeSingleNFTLiquidityDecreaseFlag"
	AlwaysSaveTokenMetaDataFlag                 core.EnableEpochFlag = "AlwaysSaveTokenMetaDataFlag"
	SetGuardianFlag                             core.EnableEpochFlag = "SetGuardianFlag"
	ConsistentTokensValuesLengthCheckFlag       core.EnableEpochFlag = "ConsistentTokensValuesLengthCheckFlag"
	ChangeUsernameFlag                          core.EnableEpochFlag = "ChangeUsernameFlag"
	AutoBalanceDataTriesFlag                    core.EnableEpochFlag = "AutoBalanceDataTriesFlag"
	ScToScLogEventFlag                          core.EnableEpochFlag = "ScToScLogEventFlag"
	FixGasRemainingForSaveKeyValueFlag          core.EnableEpochFlag = "FixGasRemainingForSaveKeyValueFlag"
	IsChangeOwnerAddressCrossShardThroughSCFlag core.EnableEpochFlag = "IsChangeOwnerAddressCrossShardThroughSCFlag"
	MigrateDataTrieFlag                         core.EnableEpochFlag = "MigrateDataTrieFlag"
	DynamicEsdtFlag                             core.EnableEpochFlag = "DynamicEsdtFlag"
	EGLDInESDTMultiTransferFlag                 core.EnableEpochFlag = "EGLDInESDTMultiTransferFlag"
)

// allFlags must have all flags used by mx-chain-vm-common-go in the current version
var allFlags = []core.EnableEpochFlag{
	GlobalMintBurnFlag,
	ESDTTransferRoleFlag,
	CheckFunctionArgumentFlag,
	CheckCorrectTokenIDForTransferRoleFlag,
	FixAsyncCallbackCheckFlag,
	SaveToSystemAccountFlag,
	CheckFrozenCollectionFlag,
	SendAlwaysFlag,
	ValueLengthCheckFlag,
	CheckTransferFlag,
	ESDTNFTImprovementV1Flag,
	FixOldTokenLiquidityFlag,
	WipeSingleNFTLiquidityDecreaseFlag,
	AlwaysSaveTokenMetaDataFlag,
	SetGuardianFlag,
	ConsistentTokensValuesLengthCheckFlag,
	ChangeUsernameFlag,
	AutoBalanceDataTriesFlag,
	ScToScLogEventFlag,
	FixGasRemainingForSaveKeyValueFlag,
	IsChangeOwnerAddressCrossShardThroughSCFlag,
	MigrateDataTrieFlag,
	DynamicEsdtFlag,
	EGLDInESDTMultiTransferFlag,
}
