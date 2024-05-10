package builtInFunctions

// CrossChainTokenCheckerHandler should check if token is from another chain/sovereign shard
type CrossChainTokenCheckerHandler interface {
	IsCrossChainOperation(tokenID []byte) bool
	IsSelfMainChain() bool
	IsInterfaceNil() bool
}
