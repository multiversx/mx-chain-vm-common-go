package builtInFunctions

import vmcommon "github.com/multiversx/mx-chain-vm-common-go"

// GlobalMetadataHandler provides functions which handle global metadata
type GlobalMetadataHandler interface {
	vmcommon.ExtendedESDTGlobalSettingsHandler
	GetGlobalMetadata(esdtTokenKey []byte) (*vmcommon.ESDTGlobalMetadata, error)
	SaveGlobalMetadata(esdtTokenKey []byte, esdtMetaData *vmcommon.ESDTGlobalMetadata) error
	IsInterfaceNil() bool
}
