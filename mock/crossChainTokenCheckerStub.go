package mock

// CrossChainTokenCheckerMock -
type CrossChainTokenCheckerMock struct {
	IsCrossChainOperationCalled        func(tokenID []byte) bool
	IsCrossChainOperationAllowedCalled func(address []byte, tokenID []byte) bool
}

// IsCrossChainOperation -
func (stub *CrossChainTokenCheckerMock) IsCrossChainOperation(tokenID []byte) bool {
	if stub.IsCrossChainOperationCalled != nil {
		return stub.IsCrossChainOperationCalled(tokenID)
	}

	return false
}

// IsCrossChainOperationAllowed -
func (stub *CrossChainTokenCheckerMock) IsCrossChainOperationAllowed(address []byte, tokenID []byte) bool {
	if stub.IsCrossChainOperationAllowedCalled != nil {
		return stub.IsCrossChainOperationAllowedCalled(address, tokenID)
	}

	return false
}

// IsInterfaceNil -
func (stub *CrossChainTokenCheckerMock) IsInterfaceNil() bool {
	return stub == nil
}
