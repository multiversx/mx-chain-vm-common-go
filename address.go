package vmcommon

import (
	"bytes"

	"github.com/multiversx/mx-chain-core-go/core"
)

// SystemAccountAddress is the hard-coded address in which we save global settings on all shards
var SystemAccountAddress = bytes.Repeat([]byte{255}, 32)

// NumInitCharactersForScAddress numbers of characters for smart contract address identifier
const NumInitCharactersForScAddress = 10

// VMTypeLen number of characters with VMType identifier in an address, these are the last 2 characters from the
// initial identifier
const VMTypeLen = 2

// ShardIdentiferLen number of characters for shard identifier in an address
const ShardIdentiferLen = 2

const metaChainShardIdentifier uint8 = 255
const numInitCharactersForOnMetachainSC = 15

const numInitCharactersForSystemAccountAddress = 30

// IsSystemAccountAddress returns true if given address is system account address
func IsSystemAccountAddress(address []byte) bool {
	if len(address) < numInitCharactersForSystemAccountAddress {
		return false
	}
	return bytes.Equal(address[:numInitCharactersForSystemAccountAddress], SystemAccountAddress[:numInitCharactersForSystemAccountAddress])
}

// IsSmartContractAddress verifies if a set address is of type smart contract
func IsSmartContractAddress(rcvAddress []byte) bool {
	if len(rcvAddress) <= NumInitCharactersForScAddress {
		return false
	}

	if IsEmptyAddress(rcvAddress) {
		return true
	}

	numOfZeros := NumInitCharactersForScAddress - VMTypeLen
	isSCAddress := bytes.Equal(rcvAddress[:numOfZeros], make([]byte, numOfZeros))
	return isSCAddress
}

// IsEmptyAddress returns whether an address is empty
func IsEmptyAddress(address []byte) bool {
	isEmptyAddress := bytes.Equal(address, make([]byte, len(address)))
	return isEmptyAddress
}

// IsMetachainIdentifier verifies if the identifier is of type metachain
func IsMetachainIdentifier(identifier []byte) bool {
	if len(identifier) == 0 {
		return false
	}

	for i := 0; i < len(identifier); i++ {
		if identifier[i] != metaChainShardIdentifier {
			return false
		}
	}

	return true
}

// IsSmartContractOnMetachain verifies if an address is smart contract on metachain
func IsSmartContractOnMetachain(identifier []byte, rcvAddress []byte) bool {
	if len(rcvAddress) <= NumInitCharactersForScAddress+numInitCharactersForOnMetachainSC {
		return false
	}

	if !IsMetachainIdentifier(identifier) {
		return false
	}

	if !IsSmartContractAddress(rcvAddress) {
		return false
	}

	leftSide := rcvAddress[NumInitCharactersForScAddress:(NumInitCharactersForScAddress + numInitCharactersForOnMetachainSC)]
	isOnMetaChainSCAddress := bytes.Equal(leftSide,
		make([]byte, numInitCharactersForOnMetachainSC))
	return isOnMetaChainSCAddress
}

// IsAllowedToSaveUnderKey returns if saving key-value in data tries under given key is allowed
func IsAllowedToSaveUnderKey(key []byte) bool {
	prefixLen := len(core.ProtectedKeyPrefix)
	if len(key) < prefixLen {
		return true
	}

	trimmedKey := key[:prefixLen]
	return !bytes.Equal(trimmedKey, []byte(core.ProtectedKeyPrefix))
}
