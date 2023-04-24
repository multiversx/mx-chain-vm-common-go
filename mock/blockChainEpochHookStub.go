package mock

// BlockChainEpochHookStub -
type BlockChainEpochHookStub struct {
	CurrentEpochCalled func() uint32
}

// CurrentEpoch -
func (b *BlockChainEpochHookStub) CurrentEpoch() uint32 {
	if b.CurrentEpochCalled != nil {
		return b.CurrentEpochCalled()
	}
	return 0
}

// IsInterfaceNil -
func (b *BlockChainEpochHookStub) IsInterfaceNil() bool {
	return b == nil
}
