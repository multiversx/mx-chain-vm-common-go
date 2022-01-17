package mock

import (
	"math/big"

	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

// UserAccountStub -
type UserAccountStub struct {
	Address                  []byte
	AddToBalanceCalled       func(value *big.Int) error
	AccountDataHandlerCalled func() vmcommon.AccountDataHandler
	SetCodeMetaDataCalled    func(codeMetaData []byte)
}

// HasNewCode -
func (u *UserAccountStub) HasNewCode() bool {
	return false
}

// SetUserName -
func (u *UserAccountStub) SetUserName(_ []byte) {
}

// GetUserName -
func (u *UserAccountStub) GetUserName() []byte {
	return nil
}

// AddToBalance -
func (u *UserAccountStub) AddToBalance(value *big.Int) error {
	if u.AddToBalanceCalled != nil {
		return u.AddToBalanceCalled(value)
	}
	return nil
}

// SubFromBalance -
func (u *UserAccountStub) SubFromBalance(_ *big.Int) error {
	return nil
}

// GetBalance -
func (u *UserAccountStub) GetBalance() *big.Int {
	return nil
}

// ClaimDeveloperRewards -
func (u *UserAccountStub) ClaimDeveloperRewards([]byte) (*big.Int, error) {
	return nil, nil
}

// AddToDeveloperReward -
func (u *UserAccountStub) AddToDeveloperReward(*big.Int) {

}

// GetDeveloperReward -
func (u *UserAccountStub) GetDeveloperReward() *big.Int {
	return nil
}

// ChangeOwnerAddress -
func (u *UserAccountStub) ChangeOwnerAddress([]byte, []byte) error {
	return nil
}

// SetOwnerAddress -
func (u *UserAccountStub) SetOwnerAddress([]byte) {

}

// GetOwnerAddress -
func (u *UserAccountStub) GetOwnerAddress() []byte {
	return nil
}

// AddressBytes -
func (u *UserAccountStub) AddressBytes() []byte {
	return u.Address
}

//IncreaseNonce -
func (u *UserAccountStub) IncreaseNonce(_ uint64) {
}

// GetNonce -
func (u *UserAccountStub) GetNonce() uint64 {
	return 0
}

// SetCode -
func (u *UserAccountStub) SetCode(_ []byte) {

}

// SetCodeMetadata -
func (u *UserAccountStub) SetCodeMetadata(codeMetaData []byte) {
	if u.SetCodeMetaDataCalled != nil {
		u.SetCodeMetaDataCalled(codeMetaData)
	}
}

// GetCodeMetadata -
func (u *UserAccountStub) GetCodeMetadata() []byte {
	return nil
}

// SetCodeHash -
func (u *UserAccountStub) SetCodeHash(_ []byte) {

}

// GetCodeHash -
func (u *UserAccountStub) GetCodeHash() []byte {
	return nil
}

// SetRootHash -
func (u *UserAccountStub) SetRootHash(_ []byte) {

}

// GetRootHash -
func (u *UserAccountStub) GetRootHash() []byte {
	return nil
}

// DataTrieTracker -
func (u *UserAccountStub) AccountDataHandler() vmcommon.AccountDataHandler {
	if u.AccountDataHandlerCalled != nil {
		return u.AccountDataHandlerCalled()
	}
	return nil
}

// IsInterfaceNil -
func (u *UserAccountStub) IsInterfaceNil() bool {
	return u == nil
}
