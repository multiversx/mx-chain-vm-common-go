package datafield

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"unicode"

	"github.com/multiversx/mx-chain-core-go/core"
)

const (
	esdtIdentifierSeparator  = "-"
	esdtRandomSequenceLength = 6
)

// TODO refactor this part to use the built-in container for the list of all the built-in functions
func getAllBuiltInFunctions() []string {
	return []string{
		core.BuiltInFunctionClaimDeveloperRewards,
		core.BuiltInFunctionChangeOwnerAddress,
		core.BuiltInFunctionSetUserName,
		core.BuiltInFunctionSaveKeyValue,
		core.BuiltInFunctionESDTTransfer,
		core.BuiltInFunctionESDTBurn,
		core.BuiltInFunctionESDTFreeze,
		core.BuiltInFunctionESDTUnFreeze,
		core.BuiltInFunctionESDTWipe,
		core.BuiltInFunctionESDTPause,
		core.BuiltInFunctionESDTUnPause,
		core.BuiltInFunctionSetESDTRole,
		core.BuiltInFunctionUnSetESDTRole,
		core.BuiltInFunctionESDTSetLimitedTransfer,
		core.BuiltInFunctionESDTUnSetLimitedTransfer,
		core.BuiltInFunctionESDTLocalMint,
		core.BuiltInFunctionESDTLocalBurn,
		core.BuiltInFunctionESDTNFTTransfer,
		core.BuiltInFunctionESDTNFTCreate,
		core.BuiltInFunctionESDTNFTAddQuantity,
		core.BuiltInFunctionESDTNFTCreateRoleTransfer,
		core.BuiltInFunctionESDTNFTBurn,
		core.BuiltInFunctionESDTNFTAddURI,
		core.BuiltInFunctionESDTNFTUpdateAttributes,
		core.BuiltInFunctionMultiESDTNFTTransfer,
		core.BuiltInFunctionMigrateDataTrie,
		core.ESDTRoleLocalMint,
		core.ESDTRoleLocalBurn,
		core.ESDTRoleNFTCreate,
		core.ESDTRoleNFTCreateMultiShard,
		core.ESDTRoleNFTAddQuantity,
		core.ESDTRoleNFTBurn,
		core.ESDTRoleNFTAddURI,
		core.ESDTRoleNFTUpdateAttributes,
		core.ESDTRoleTransfer,
		core.BuiltInFunctionSetGuardian,
		core.BuiltInFunctionUnGuardAccount,
		core.BuiltInFunctionGuardAccount,
	}
}

func isBuiltInFunction(builtInFunctionsList []string, function string) bool {
	for _, builtInFunction := range builtInFunctionsList {
		if builtInFunction == function {
			return true
		}
	}

	return false
}

func computeTokenIdentifier(token string, nonce uint64) string {
	if token == "" || nonce == 0 {
		return ""
	}

	nonceBig := big.NewInt(0).SetUint64(nonce)
	hexEncodedNonce := hex.EncodeToString(nonceBig.Bytes())
	return fmt.Sprintf("%s-%s", token, hexEncodedNonce)
}

func extractTokenAndNonce(arg []byte) (string, uint64) {
	argsSplit := bytes.Split(arg, []byte(esdtIdentifierSeparator))
	if len(argsSplit) < 2 {
		return string(arg), 0
	}

	if len(argsSplit[1]) <= esdtRandomSequenceLength {
		return string(arg), 0
	}

	identifier := []byte(fmt.Sprintf("%s-%s", argsSplit[0], argsSplit[1][:esdtRandomSequenceLength]))
	nonce := big.NewInt(0).SetBytes(argsSplit[1][esdtRandomSequenceLength:])

	return string(identifier), nonce.Uint64()
}

func isEmptyAddr(addrLength int, address []byte) bool {
	emptyAddr := make([]byte, addrLength)

	return bytes.Equal(address, emptyAddr)
}

func isASCIIString(input string) bool {
	for i := 0; i < len(input); i++ {
		if input[i] > unicode.MaxASCII {
			return false
		}
	}

	return true
}
