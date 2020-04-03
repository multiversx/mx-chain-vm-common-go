package vmcommon

import (
	"encoding/hex"
	"strings"
)

func tokenize(data string) ([]string, error) {
	tokens := strings.Split(data, atSeparator)

	if len(tokens) == 0 || len(tokens[0]) == 0 {
		return nil, ErrTokenizeFailed
	}

	return tokens, nil
}

func decodeToken(token string) ([]byte, error) {
	token = ensureEvenLength(token)
	decoded, err := hex.DecodeString(token)
	if err != nil {
		return nil, err
	}

	return decoded, nil
}

func ensureEvenLength(str string) string {
	if len(str)%2 != 0 {
		return "0" + str
	}
	return str
}

func trimLeadingSeparatorChar(data string) string {
	if len(data) > 0 && data[0] == atSeparatorChar {
		data = data[1:]
	}

	return data
}

func requireNumTokensIsEven(tokens []string) error {
	if len(tokens)%2 == 0 {
		return nil
	}

	return ErrInvalidDataString
}
