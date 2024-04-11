package mock

// BlockDataHandlerStub -
type BlockDataHandlerStub struct {
	CurrentRoundCalled func() uint64
}

// CurrentRound -
func (b *BlockDataHandlerStub) CurrentRound() uint64 {
	if b.CurrentRoundCalled != nil {
		return b.CurrentRoundCalled()
	}
	return 0
}

// IsInterfaceNil returns true if there is no value under the interface
func (b *BlockDataHandlerStub) IsInterfaceNil() bool {
	return b == nil
}
