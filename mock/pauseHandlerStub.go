package mock

// GlobalSettingsHandlerStub -
type GlobalSettingsHandlerStub struct {
	IsPausedCalled                              func(token []byte) bool
	IsLimiterTransferCalled                     func(token []byte) bool
	IsBurnForAllCalled                          func(token []byte) bool
	IsSenderOrDestinationWithTransferRoleCalled func(sender, destionation, tokenID []byte) bool
}

// IsPaused -
func (p *GlobalSettingsHandlerStub) IsPaused(token []byte) bool {
	if p.IsPausedCalled != nil {
		return p.IsPausedCalled(token)
	}
	return false
}

// IsLimitedTransfer -
func (p *GlobalSettingsHandlerStub) IsLimitedTransfer(token []byte) bool {
	if p.IsLimiterTransferCalled != nil {
		return p.IsLimiterTransferCalled(token)
	}
	return false
}

// IsBurnForAll -
func (p *GlobalSettingsHandlerStub) IsBurnForAll(token []byte) bool {
	if p.IsBurnForAllCalled != nil {
		return p.IsBurnForAllCalled(token)
	}
	return false
}

// IsSenderOrDestinationWithTransferRole -
func (p *GlobalSettingsHandlerStub) IsSenderOrDestinationWithTransferRole(sender, destination, tokenID []byte) bool {
	if p.IsSenderOrDestinationWithTransferRoleCalled != nil {
		return p.IsSenderOrDestinationWithTransferRoleCalled(sender, destination, tokenID)
	}
	return false
}

// IsInterfaceNil -
func (p *GlobalSettingsHandlerStub) IsInterfaceNil() bool {
	return p == nil
}
