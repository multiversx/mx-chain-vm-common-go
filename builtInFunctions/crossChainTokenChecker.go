package builtInFunctions

import (
	"bytes"
	"fmt"

	"github.com/multiversx/mx-chain-core-go/data/esdt"
)

type crossChainTokenChecker struct {
	selfESDTPrefix []byte
}

// NewCrossChainTokenChecker creates a new cross chain token checker
func NewCrossChainTokenChecker(selfESDTPrefix []byte) (*crossChainTokenChecker, error) {
	ctc := &crossChainTokenChecker{
		selfESDTPrefix: selfESDTPrefix,
	}

	if len(selfESDTPrefix) == 0 {
		return ctc, nil
	}

	if !esdt.IsValidTokenPrefix(string(selfESDTPrefix)) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidTokenPrefix, selfESDTPrefix)
	}

	return ctc, nil
}

// IsCrossChainOperation checks if the provided token comes from another chain/sovereign shard
func (ctc *crossChainTokenChecker) IsCrossChainOperation(tokenID []byte) bool {
	tokenPrefix, hasPrefix := esdt.IsValidPrefixedToken(string(tokenID))
	// no prefix or malformed token in main chain operation
	if !hasPrefix && len(ctc.selfESDTPrefix) == 0 {
		return false
	}

	return !bytes.Equal([]byte(tokenPrefix), ctc.selfESDTPrefix)
}

// IsSelfMainChain returns true if the current chain is the main chain
func (ctc *crossChainTokenChecker) IsSelfMainChain() bool {
	return len(ctc.selfESDTPrefix) == 0
}

// IsInterfaceNil checks if the underlying pointer is nil
func (ctc *crossChainTokenChecker) IsInterfaceNil() bool {
	return ctc == nil
}
