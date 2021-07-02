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

// ESDTRoleNFTAddURI is the constant string for the local role of adding a URI for ESDT NFT tokens
const ESDTRoleNFTAddURI = "ESDTRoleNFTAddURI"

// ESDTRoleNFTUpdateAttributes is the constant string for the local role of create for ESDT NFT tokens
const ESDTRoleNFTUpdateAttributes = "ESDTRoleNFTUpdateAttributes"

// BuiltInFunctionESDTNFTCreate is the key for the elrond standard digital token NFT create built-in function
const BuiltInFunctionESDTNFTCreate = "ESDTNFTCreate"

// BuiltInFunctionESDTNFTAddQuantity is the key for the elrond standard digital token NFT add quantity built-in function
const BuiltInFunctionESDTNFTAddQuantity = "ESDTNFTAddQuantity"

// BuiltInFunctionESDTNFTAddURI is the key for the elrond standard digital token NFT add URI built-in function
const BuiltInFunctionESDTNFTAddURI = "ESDTNFTAddURI"

// BuiltInFunctionESDTNFTUpdateAttributes is the key for the elrond standard digital token NFT update attributes built-in function
const BuiltInFunctionESDTNFTUpdateAttributes = "ESDTNFTUpdateAttributes"

// BuiltInFunctionMultiESDTNFTTransfer is the key for the elrond standard digital token multi transfer built-in function
const BuiltInFunctionMultiESDTNFTTransfer = "MultiESDTNFTTransfer"

// BuiltInFunctionClaimDeveloperRewards is the key for the claim developer rewards built-in function
const BuiltInFunctionClaimDeveloperRewards = "ClaimDeveloperRewards"

// BuiltInFunctionChangeOwnerAddress is the key for the change owner built in function built-in function
const BuiltInFunctionChangeOwnerAddress = "ChangeOwnerAddress"

// BuiltInFunctionSaveKeyValue is the key for the save key value built-in function
const BuiltInFunctionSaveKeyValue = "SaveKeyValue"

// BuiltInFunctionESDTFreeze is the key for the elrond standard digital token freeze built-in function
const BuiltInFunctionESDTFreeze = "ESDTFreeze"

// BuiltInFunctionESDTUnFreeze is the key for the elrond standard digital token unfreeze built-in function
const BuiltInFunctionESDTUnFreeze = "ESDTUnFreeze"

// BuiltInFunctionESDTWipe is the key for the elrond standard digital token wipe built-in function
const BuiltInFunctionESDTWipe = "ESDTWipe"

// BuiltInFunctionESDTPause is the key for the elrond standard digital token pause built-in function
const BuiltInFunctionESDTPause = "ESDTPause"

// BuiltInFunctionESDTUnPause is the key for the elrond standard digital token unpause built-in function
const BuiltInFunctionESDTUnPause = "ESDTUnPause"

// BuiltInFunctionSetESDTRole is the key for the elrond standard digital token set built-in function
const BuiltInFunctionSetESDTRole = "ESDTSetRole"

// BuiltInFunctionUnSetESDTRole is the key for the elrond standard digital token unset built-in function
const BuiltInFunctionUnSetESDTRole = "ESDTUnSetRole"

// BuiltInFunctionESDTLocalMint is the key for the elrond standard digital token local mint built-in function
const BuiltInFunctionESDTLocalMint = "ESDTLocalMint"

// BuiltInFunctionESDTLocalBurn is the key for the elrond standard digital token local burn built-in function
const BuiltInFunctionESDTLocalBurn = "ESDTLocalBurn"

// BuiltInFunctionESDTNFTBurn is the key for the elrond standard digital token NFT burn built-in function
const BuiltInFunctionESDTNFTBurn = "ESDTNFTBurn"

// BaseOperationCostString represents the field name for base operation costs
const BaseOperationCostString = "BaseOperationCost"

// BuiltInCostString represents the field name for built in operation costs
const BuiltInCostString = "BuiltInCost"

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
