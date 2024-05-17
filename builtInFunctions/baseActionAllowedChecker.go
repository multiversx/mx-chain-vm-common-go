package builtInFunctions

import (
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

type baseCrossChainActionAllowedChecker struct {
	rolesHandler           vmcommon.ESDTRoleHandler
	crossChainTokenChecker CrossChainTokenCheckerHandler
}

func (b *baseCrossChainActionAllowedChecker) isAllowedToExecute(acntSnd vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
	if b.crossChainTokenChecker.IsAllowedToMint(acntSnd.AddressBytes(), tokenID) {
		return nil
	}

	return b.rolesHandler.CheckAllowedToExecute(acntSnd, tokenID, action)
}
