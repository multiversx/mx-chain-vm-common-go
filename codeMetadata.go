package vmcommon

const lengthOfCodeMetadata = 2

// CodeMetadata represents smart contract code metadata
type CodeMetadata struct {
	Upgradeable bool
}

// CodeMetadataFromBytes creates a metadata object from bytes
func CodeMetadataFromBytes(bytes []byte) CodeMetadata {
	if len(bytes) == 0 {
		return CodeMetadata{}
	}

	return CodeMetadata{
		Upgradeable: bytes[0] == 1,
	}
}

// ToBytes converts the metadata to bytes
func (metadata *CodeMetadata) ToBytes() []byte {
	bytes := make([]byte, lengthOfCodeMetadata)

	if metadata.Upgradeable {
		bytes[0] = 1
	}

	return bytes
}
