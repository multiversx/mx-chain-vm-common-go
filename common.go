package vmcommon

import (
	"math/big"

	"github.com/multiversx/mx-chain-core-go/core"
)

const tickerMinLength = 3
const tickerMaxLength = 10
const additionalRandomCharsLength = 6
const identifierMinLength = tickerMinLength + additionalRandomCharsLength + 1
const identifierMaxLength = tickerMaxLength + additionalRandomCharsLength + 1

// ESDTDeleteMetadata represents the defined built in function name for esdt delete metadata
const ESDTDeleteMetadata = "ESDTDeleteMetadata"

// ESDTAddMetadata represents the defined built in function name for esdt add metadata
const ESDTAddMetadata = "ESDTAddMetadata"

// BuiltInFunctionESDTSetBurnRoleForAll represents the defined built in function name for esdt set burn role for all
const BuiltInFunctionESDTSetBurnRoleForAll = "ESDTSetBurnRoleForAll"

// BuiltInFunctionESDTUnSetBurnRoleForAll represents the defined built in function name for esdt unset burn role for all
const BuiltInFunctionESDTUnSetBurnRoleForAll = "ESDTUnSetBurnRoleForAll"

// BuiltInFunctionESDTTransferRoleAddAddress represents the defined built in function name for esdt transfer role add address
const BuiltInFunctionESDTTransferRoleAddAddress = "ESDTTransferRoleAddAddress"

// BuiltInFunctionESDTTransferRoleDeleteAddress represents the defined built in function name for transfer role delete address
const BuiltInFunctionESDTTransferRoleDeleteAddress = "ESDTTransferRoleDeleteAddress"

// ESDTRoleBurnForAll represents the role for burn for all
const ESDTRoleBurnForAll = "ESDTRoleBurnForAll"

// ValidateToken - validates the token ID
func ValidateToken(tokenID []byte) bool {
	tokenIDLen := len(tokenID)
	if tokenIDLen < identifierMinLength || tokenIDLen > identifierMaxLength {
		return false
	}

	tickerLen := tokenIDLen - additionalRandomCharsLength

	if !isTickerValid(tokenID[0 : tickerLen-1]) {
		return false
	}

	// dash char between the random chars and the ticker
	if tokenID[tickerLen-1] != '-' {
		return false
	}

	if !randomCharsAreValid(tokenID[tickerLen:tokenIDLen]) {
		return false
	}

	return true
}

// ticker must be all uppercase alphanumeric
func isTickerValid(tickerName []byte) bool {
	if len(tickerName) < tickerMinLength || len(tickerName) > tickerMaxLength {
		return false
	}
	for _, ch := range tickerName {
		isBigCharacter := ch >= 'A' && ch <= 'Z'
		isNumber := ch >= '0' && ch <= '9'
		isReadable := isBigCharacter || isNumber
		if !isReadable {
			return false
		}
	}

	return true
}

// random chars are alphanumeric lowercase
func randomCharsAreValid(chars []byte) bool {
	if len(chars) != additionalRandomCharsLength {
		return false
	}
	for _, ch := range chars {
		isSmallCharacter := ch >= 'a' && ch <= 'f'
		isNumber := ch >= '0' && ch <= '9'
		isReadable := isSmallCharacter || isNumber
		if !isReadable {
			return false
		}
	}

	return true
}

// ZeroValueIfNil returns 0 if the input is nil, otherwise returns the input
func ZeroValueIfNil(value *big.Int) *big.Int {
	if value == nil {
		return big.NewInt(0)
	}

	return value
}

// ArgsMigrateDataTrieLeaves is the argument structure for the MigrateDataTrieLeaves function
type ArgsMigrateDataTrieLeaves struct {
	OldVersion   core.TrieNodeVersion
	NewVersion   core.TrieNodeVersion
	TrieMigrator DataTrieMigrator
}
