package builtInFunctions

const lengthOfESDTMetadata = 2

const (
	// MetadataPaused is the location of paused flag in the esdt global meta data
	MetadataPaused = 1
	// MetadataLimitedTransfer is the location of limited transfer flag in the esdt global meta data
	MetadataLimitedTransfer = 2
	// BurnRoleForAll is the location of burn role for all flag in the esdt global meta data
	BurnRoleForAll = 4
)

const (
	// MetadataFrozen is the location of frozen flag in the esdt user meta data
	MetadataFrozen = 1
)

const (
	flagsByte     = 0
	tokenTypeByte = 1
)

// ESDTGlobalMetadata represents esdt global metadata saved on system account
type ESDTGlobalMetadata struct {
	Paused          bool
	LimitedTransfer bool
	BurnRoleForAll  bool
	TokenType       byte
}

// ESDTGlobalMetadataFromBytes creates a metadata object from bytes
func ESDTGlobalMetadataFromBytes(bytes []byte) ESDTGlobalMetadata {
	if len(bytes) != lengthOfESDTMetadata {
		return ESDTGlobalMetadata{}
	}

	return ESDTGlobalMetadata{
		Paused:          (bytes[flagsByte] & MetadataPaused) != 0,
		LimitedTransfer: (bytes[flagsByte] & MetadataLimitedTransfer) != 0,
		BurnRoleForAll:  (bytes[flagsByte] & BurnRoleForAll) != 0,
		TokenType:       bytes[tokenTypeByte],
	}
}

// ToBytes converts the metadata to bytes
func (metadata *ESDTGlobalMetadata) ToBytes() []byte {
	bytes := make([]byte, lengthOfESDTMetadata)

	if metadata.Paused {
		bytes[flagsByte] |= MetadataPaused
	}
	if metadata.LimitedTransfer {
		bytes[flagsByte] |= MetadataLimitedTransfer
	}
	if metadata.BurnRoleForAll {
		bytes[flagsByte] |= BurnRoleForAll
	}
	bytes[tokenTypeByte] = metadata.TokenType

	return bytes
}

// ESDTUserMetadata represents esdt user metadata saved on every account
type ESDTUserMetadata struct {
	Frozen bool
}

// ESDTUserMetadataFromBytes creates a metadata object from bytes
func ESDTUserMetadataFromBytes(bytes []byte) ESDTUserMetadata {
	if len(bytes) != lengthOfESDTMetadata {
		return ESDTUserMetadata{}
	}

	return ESDTUserMetadata{
		Frozen: (bytes[flagsByte] & MetadataFrozen) != 0,
	}
}

// ToBytes converts the metadata to bytes
func (metadata *ESDTUserMetadata) ToBytes() []byte {
	bytes := make([]byte, lengthOfESDTMetadata)

	if metadata.Frozen {
		bytes[flagsByte] |= MetadataFrozen
	}

	return bytes
}
