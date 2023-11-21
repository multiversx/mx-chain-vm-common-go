package vmcommon

// BaseOperationCost defines cost for base operation cost
type BaseOperationCost struct {
	StorePerByte      uint64
	ReleasePerByte    uint64
	DataCopyPerByte   uint64
	PersistPerByte    uint64
	CompilePerByte    uint64
	AoTPreparePerByte uint64
}

// BuiltInCost defines cost for built-in methods
type BuiltInCost struct {
	ChangeOwnerAddress       uint64
	ClaimDeveloperRewards    uint64
	SaveUserName             uint64
	SaveKeyValue             uint64
	ESDTTransfer             uint64
	ESDTBurn                 uint64
	ESDTLocalMint            uint64
	ESDTLocalBurn            uint64
	ESDTModifyRoyalties      uint64
	ESDTModifyCreator        uint64
	ESDTNFTCreate            uint64
	ESDTNFTRecreate          uint64
	ESDTNFTUpdate            uint64
	ESDTNFTAddQuantity       uint64
	ESDTNFTBurn              uint64
	ESDTNFTTransfer          uint64
	ESDTNFTChangeCreateOwner uint64
	ESDTNFTMultiTransfer     uint64
	ESDTNFTAddURI            uint64
	ESDTNFTSetNewURIs        uint64
	ESDTNFTUpdateAttributes  uint64
	SetGuardian              uint64
	GuardAccount             uint64
	TrieLoadPerNode          uint64
	TrieStorePerNode         uint64
}

// GasCost holds all the needed gas costs for system smart contracts
type GasCost struct {
	BaseOperationCost BaseOperationCost
	BuiltInCost       BuiltInCost
}

// SafeSubUint64 performs subtraction on uint64 and returns an error if it overflows
func SafeSubUint64(a, b uint64) (uint64, error) {
	if a < b {
		return 0, ErrSubtractionOverflow
	}
	return a - b, nil
}
