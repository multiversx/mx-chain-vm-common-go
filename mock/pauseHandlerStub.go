package mock

// GlobalSettingsHandlerStub -
type GlobalSettingsHandlerStub struct {
	IsPausedCalled          func(token []byte) bool
	IsLimiterTransferCalled func(token []byte) bool
}

// IsPaused -
func (p *GlobalSettingsHandlerStub) IsPaused(token []byte) bool {
	if p.IsPausedCalled != nil {
		return p.IsPausedCalled(token)
	}
	return false
}

// IsPaused -
func (p *GlobalSettingsHandlerStub) IsLimitedTransfer(token []byte) bool {
	if p.IsLimiterTransferCalled != nil {
		return p.IsLimiterTransferCalled(token)
	}
	return false
}

// IsInterfaceNil -
func (p *GlobalSettingsHandlerStub) IsInterfaceNil() bool {
	return p == nil
}
