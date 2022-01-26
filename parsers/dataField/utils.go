package datafield

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
)

const (
	esdtIdentifierSeparator  = "-"
	esdtRandomSequenceLength = 6
)

func computeTokenIdentifier(token string, nonce uint64) string {
	if token == "" || nonce == 0 {
		return ""
	}

	nonceBig := big.NewInt(0).SetUint64(nonce)
	hexEncodedNonce := hex.EncodeToString(nonceBig.Bytes())
	return fmt.Sprintf("%s-%s", token, hexEncodedNonce)
}

func extractTokenIdentifierAndNonce(arg []byte) ([]byte, uint64) {
	argsSplit := bytes.Split(arg, []byte(esdtIdentifierSeparator))
	if len(argsSplit) < 2 {
		return arg, 0
	}

	if len(argsSplit[1]) <= esdtRandomSequenceLength {
		return arg, 0
	}

	identifier := []byte(fmt.Sprintf("%s-%s", argsSplit[0], argsSplit[1][:esdtRandomSequenceLength]))
	nonce := big.NewInt(0).SetBytes(argsSplit[1][esdtRandomSequenceLength:])

	return identifier, nonce.Uint64()
}
