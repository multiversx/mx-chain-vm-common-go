package mock

import (
	"bytes"
	"errors"
	"math/big"

	"github.com/multiversx/mx-chain-vm-common"
)

// ErrOperationNotPermitted indicates an operation rejected due to insufficient
// permissions.
var ErrOperationNotPermitted = errors.New("operation not permitted")

// ErrInvalidAddressLength indicates an incorrect length given for an address.
var ErrInvalidAddressLength = errors.New("invalid address length")

var zero = big.NewInt(0)

// Account holds the account info
type Account struct {
	Address         []byte
	Nonce           uint64
	Balance         *big.Int
	Storage         map[string][]byte
	RootHash        []byte
	Code            []byte
	CodeHash        []byte
	CodeMetadata    []byte
	OwnerAddress    []byte
	Username        []byte
	DeveloperReward *big.Int
}

var storageDefaultValue = []byte{}

func NewUserAccount(address []byte) *Account {
	return &Account{
		Address:         address,
		Nonce:           0,
		Balance:         big.NewInt(0),
		Storage:         make(map[string][]byte),
		RootHash:        nil,
		Code:            nil,
		CodeHash:        nil,
		CodeMetadata:    nil,
		OwnerAddress:    nil,
		Username:        nil,
		DeveloperReward: big.NewInt(0),
	}
}

// StorageValue yields the storage value for key, default 0
func (a *Account) StorageValue(key string) []byte {
	value, found := a.Storage[key]
	if !found {
		return storageDefaultValue
	}
	return value
}

// SetCodeAndMetadata changes the account code, as well as all fields depending on it:
// CodeHash, IsSmartContract, CodeMetadata.
// The code metadata must be given explicitly.
func (a *Account) SetCodeAndMetadata(code []byte, codeMetadata *vmcommon.CodeMetadata) {
	a.Code = code
	a.CodeHash = code
	a.CodeMetadata = codeMetadata.ToBytes()
}

// AddressBytes -
func (a *Account) AddressBytes() []byte {
	return a.Address
}

// GetNonce -
func (a *Account) GetNonce() uint64 {
	return a.Nonce
}

// GetCode -
func (a *Account) GetCode() []byte {
	return a.Code
}

// GetCodeMetadata -
func (a *Account) GetCodeMetadata() []byte {
	return a.CodeMetadata
}

// GetCodeHash -
func (a *Account) GetCodeHash() []byte {
	return a.CodeHash
}

// GetRootHash -
func (a *Account) GetRootHash() []byte {
	return a.RootHash
}

// GetBalance -
func (a *Account) GetBalance() *big.Int {
	return a.Balance
}

// SetBalance -
func (a *Account) SetBalance(balance int64) {
	a.Balance = big.NewInt(balance)
}

// GetDeveloperReward -
func (a *Account) GetDeveloperReward() *big.Int {
	return big.NewInt(0)
}

// GetOwnerAddress -
func (a *Account) GetOwnerAddress() []byte {
	return a.OwnerAddress
}

// GetUserName -
func (a *Account) GetUserName() []byte {
	return a.Username
}

// IsInterfaceNil -
func (a *Account) IsInterfaceNil() bool {
	return a == nil
}

// SetCode -
func (a *Account) SetCode(code []byte) {
	a.Code = code
	a.CodeHash = code
}

// SetCodeMetadata -
func (a *Account) SetCodeMetadata(codeMetadata []byte) {
	a.CodeMetadata = codeMetadata
}

// SetCodeHash -
func (a *Account) SetCodeHash(hash []byte) {
	a.CodeHash = hash
}

// SetRootHash -
func (a *Account) SetRootHash(hash []byte) {
	a.RootHash = hash
}

// AccountDataHandler -
func (a *Account) AccountDataHandler() vmcommon.AccountDataHandler {
	return a
}

// AddToBalance -
func (a *Account) AddToBalance(value *big.Int) error {
	newBalance := big.NewInt(0).Add(a.Balance, value)
	if newBalance.Cmp(zero) < 0 {
		return errors.New("insufficient funds")
	}

	a.Balance = newBalance
	return nil
}

// SubFromBalance -
func (a *Account) SubFromBalance(value *big.Int) error {
	newBalance := big.NewInt(0).Sub(a.Balance, value)
	if newBalance.Cmp(zero) < 0 {
		return errors.New("insufficient funds")
	}

	a.Balance = newBalance
	return nil
}

// ClaimDeveloperRewards -
func (a *Account) ClaimDeveloperRewards(sender []byte) (*big.Int, error) {
	if !bytes.Equal(sender, a.OwnerAddress) {
		return nil, ErrOperationNotPermitted
	}

	oldValue := big.NewInt(0).Set(a.DeveloperReward)
	a.DeveloperReward = big.NewInt(0)

	return oldValue, nil
}

// AddToDeveloperReward -
func (a *Account) AddToDeveloperReward(value *big.Int) {
	a.DeveloperReward = big.NewInt(0).Add(a.DeveloperReward, value)
}

// ChangeOwnerAddress -
func (a *Account) ChangeOwnerAddress(sender []byte, newAddress []byte) error {
	if !bytes.Equal(sender, a.OwnerAddress) {
		return ErrOperationNotPermitted
	}
	if len(newAddress) != len(a.Address) {
		return ErrInvalidAddressLength
	}

	a.OwnerAddress = newAddress

	return nil
}

// SetOwnerAddress -
func (a *Account) SetOwnerAddress(address []byte) {
	a.OwnerAddress = address
}

// SetUserName -
func (a *Account) SetUserName(userName []byte) {
	a.Username = make([]byte, len(userName))
	copy(a.Username, userName)
}

// IncreaseNonce -
func (a *Account) IncreaseNonce(nonce uint64) {
	a.Nonce += nonce
}

// RetrieveValue -
func (a *Account) RetrieveValue(key []byte) ([]byte, uint32, error) {
	return a.Storage[string(key)], 0, nil
}

// SaveKeyValue -
func (a *Account) SaveKeyValue(key []byte, value []byte) error {
	a.Storage[string(key)] = value
	return nil
}

// ClearDataCaches -
func (a *Account) ClearDataCaches() {
}

// DirtyData -
func (a *Account) DirtyData() map[string][]byte {
	return a.Storage
}

// Clone -
func (a *Account) Clone() *Account {
	return &Account{
		Address:         a.Address,
		Nonce:           a.Nonce,
		Balance:         big.NewInt(0).Set(a.Balance),
		Storage:         a.cloneStorage(),
		RootHash:        cloneBytes(a.RootHash),
		Code:            cloneBytes(a.Code),
		CodeHash:        cloneBytes(a.CodeHash),
		CodeMetadata:    cloneBytes(a.CodeMetadata),
		OwnerAddress:    cloneBytes(a.OwnerAddress),
		Username:        cloneBytes(a.Username),
		DeveloperReward: big.NewInt(0).Set(a.DeveloperReward),
	}
}

func (a *Account) cloneStorage() map[string][]byte {
	clone := make(map[string][]byte, len(a.Storage))
	for key, value := range a.Storage {
		clone[key] = cloneBytes(value)
	}

	return clone
}

func cloneBytes(b []byte) []byte {
	clone := make([]byte, len(b))
	copy(clone, b)
	return clone
}
