package vmcommon

// MetachainShardId will be used to identify a shard ID as metachain
const MetachainShardId = uint32(0xFFFFFFFF)

// AllShardId will be used to identify that a message is for all shards
const AllShardId = uint32(0xFFFFFFF0)

// ElrondProtectedKeyPrefix is the key prefix which is protected from writing in the trie - only for special builtin functions
const ElrondProtectedKeyPrefix = "ELROND"

// ESDTKeyIdentifier is the key prefix for esdt tokens
const ESDTKeyIdentifier = "esdt"

// ESDTRoleIdentifier is the key prefix for esdt role identifier
const ESDTRoleIdentifier = "role"

// ESDTNFTLatestNonceIdentifier is the key prefix for esdt latest nonce identifier
const ESDTNFTLatestNonceIdentifier = "nonce"

// BuiltInFunctionSetUserName is the key for the set user name built-in function
const BuiltInFunctionSetUserName = "SetUserName"

// BuiltInFunctionESDTBurn is the key for the elrond standard digital token burn built-in function
const BuiltInFunctionESDTBurn = "ESDTBurn"

// BuiltInFunctionESDTNFTCreateRoleTransfer is the key for the elrond standard digital token create role transfer function
const BuiltInFunctionESDTNFTCreateRoleTransfer = "ESDTNFTCreateRoleTransfer"

// ESDTRoleLocalBurn is the constant string for the local role of burn for ESDT tokens
const ESDTRoleLocalBurn = "ESDTRoleLocalBurn"

// BuiltInFunctionESDTTransfer is the key for the elrond standard digital token transfer built-in function
const BuiltInFunctionESDTTransfer = "ESDTTransfer"

// BuiltInFunctionESDTNFTTransfer is the key for the elrond standard digital token NFT transfer built-in function
const BuiltInFunctionESDTNFTTransfer = "ESDTNFTTransfer"

// MinLenArgumentsESDTTransfer defines the min length of arguments for the ESDT transfer
const MinLenArgumentsESDTTransfer = 2

// MinLenArgumentsESDTNFTTransfer defines the minimum length for esdt nft transfer
const MinLenArgumentsESDTNFTTransfer = 4

// MaxLenForESDTIssueMint defines the maximum length in bytes for the issued/minted balance
const MaxLenForESDTIssueMint = 100

// ESDTRoleLocalMint is the constant string for the local role of mint for ESDT tokens
const ESDTRoleLocalMint = "ESDTRoleLocalMint"

// ESDTRoleNFTCreate is the constant string for the local role of create for ESDT NFT tokens
const ESDTRoleNFTCreate = "ESDTRoleNFTCreate"

// ESDTRoleNFTAddQuantity is the constant string for the local role of adding quantity for existing ESDT NFT tokens
const ESDTRoleNFTAddQuantity = "ESDTRoleNFTAddQuantity"

// ESDTRoleNFTBurn is the constant string for the local role of burn for ESDT NFT tokens
const ESDTRoleNFTBurn = "ESDTRoleNFTBurn"

// ESDTType defines the possible types in case of ESDT tokens
type ESDTType uint32

const (
	// Fungible defines the token type for ESDT fungible tokens
	Fungible ESDTType = iota
	// NonFungible defines the token type for ESDT non fungible tokens
	NonFungible
)

// FungibleESDT defines the string for the token type of fungible ESDT
const FungibleESDT = "FungibleESDT"

// NonFungibleESDT defines the string for the token type of non fungible ESDT
const NonFungibleESDT = "NonFungibleESDT"

// SemiFungibleESDT defines the string for the token type of semi fungible ESDT
const SemiFungibleESDT = "SemiFungibleESDT"

// MaxRoyalty defines 100% as uint32
const MaxRoyalty = uint32(10000)

// ESDTSCAddress is the hard-coded address for esdt issuing smart contract
var ESDTSCAddress = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 255, 255}
