package datafield

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"unicode"

	"github.com/ElrondNetwork/elrond-go-core/core"
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

func isEmptyAddr(pubKeyConverter core.PubkeyConverter, receiver []byte) bool {
	emptyAddr := make([]byte, pubKeyConverter.Len())

	return bytes.Equal(receiver, emptyAddr)
}

func isASCIIString(input string) bool {
	for i := 0; i < len(input); i++ {
		if input[i] > unicode.MaxASCII {
			return false
		}
	}

	return true
}
