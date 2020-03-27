package vmcommon

import (
	"encoding/hex"
)

// AtArgumentParser is a parser that splits arguments by @ character
type AtArgumentParser struct {
	// First argument is a string (function name or hex-encoded bytecode), the rest are raw bytes
	arguments [][]byte
}

const atSeparator = "@"
const atSeparatorChar = '@'
const startIndexOfConstructorArguments = 2
const startIndexOfFunctionArguments = 1
const minNumDeployArguments = 2
const minNumCallArguments = 1
const indexOfCode = 0
const indexOfCodeMetadata = 1
const indexOfFunction = indexOfCode

// NewAtArgumentParser creates a new parser
func NewAtArgumentParser() *AtArgumentParser {
	return &AtArgumentParser{
		arguments: make([][]byte, 0),
	}
}

// ParseData creates the code and the arguments from the input data
// format: code@arg1@arg2@arg3...
// Until the first @ all the bytes are for the code / function
// after that every argument start with an @
func (parser *AtArgumentParser) ParseData(data string) error {
	tokens := tokenize(data)
	err := requireAnyTokens(tokens)
	if err != nil {
		return err
	}

	// First argument is not decoded, but left as it is
	parser.arguments = append(parser.arguments, []byte(tokens[0]))

	for i := 1; i < len(tokens); i++ {
		argument, err := decodeToken(tokens[i])
		if err != nil {
			return err
		}

		parser.arguments = append(parser.arguments, argument)
	}

	return nil
}

// GetFunctionArguments returns the call arguments
func (parser *AtArgumentParser) GetFunctionArguments() ([][]byte, error) {
	if len(parser.arguments) < startIndexOfFunctionArguments {
		return nil, ErrNilArguments
	}

	return parser.arguments[startIndexOfFunctionArguments:], nil
}

// GetConstructorArguments returns the deploy arguments
func (parser *AtArgumentParser) GetConstructorArguments() ([][]byte, error) {
	if len(parser.arguments) < startIndexOfConstructorArguments {
		return nil, ErrNilArguments
	}

	return parser.arguments[startIndexOfConstructorArguments:], nil
}

// GetCode returns the code from the parsed data
func (parser *AtArgumentParser) GetCode() ([]byte, error) {
	if len(parser.arguments) < minNumDeployArguments {
		return nil, ErrNilCode
	}

	hexCode := parser.arguments[indexOfCode]
	return hexCode, nil
}

// GetCodeMetadata returns the code metadata from the parsed data
func (parser *AtArgumentParser) GetCodeMetadata() ([]byte, error) {
	if len(parser.arguments) < minNumDeployArguments {
		return nil, ErrNilCodeMetadata
	}

	return parser.arguments[indexOfCodeMetadata], nil
}

// GetFunction returns the function from the parsed data
func (parser *AtArgumentParser) GetFunction() (string, error) {
	if len(parser.arguments) < minNumDeployArguments {
		return "", ErrNilFunction
	}

	return string(parser.arguments[indexOfFunction]), nil
}

// GetSeparator returns the separator used for parsing the data
func (parser *AtArgumentParser) GetSeparator() string {
	return atSeparator
}

// GetStorageUpdates parse data into storage updates
func (parser *AtArgumentParser) GetStorageUpdates(data string) ([]*StorageUpdate, error) {
	data = trimLeadingSeparatorChar(data)

	tokens := tokenize(data)
	err := requireAnyTokens(tokens)
	if err != nil {
		return nil, err
	}
	err = requireNumTokensIsEven(tokens)
	if err != nil {
		return nil, err
	}

	storageUpdates := make([]*StorageUpdate, 0, len(tokens))
	for i := 0; i < len(tokens); i += 2 {
		offset, err := decodeToken(tokens[i])
		if err != nil {
			return nil, err
		}

		value, err := decodeToken(tokens[i+1])
		if err != nil {
			return nil, err
		}

		storageUpdate := &StorageUpdate{Offset: offset, Data: value}
		storageUpdates = append(storageUpdates, storageUpdate)
	}

	return storageUpdates, nil
}

// CreateDataFromStorageUpdate creates storage update from data
func (parser *AtArgumentParser) CreateDataFromStorageUpdate(storageUpdates []*StorageUpdate) string {
	data := ""
	for i := 0; i < len(storageUpdates); i++ {
		storageUpdate := storageUpdates[i]
		data = data + hex.EncodeToString(storageUpdate.Offset)
		data = data + parser.GetSeparator()
		data = data + hex.EncodeToString(storageUpdate.Data)

		if i < len(storageUpdates)-1 {
			data = data + parser.GetSeparator()
		}
	}
	return data
}

// IsInterfaceNil returns true if there is no value under the interface
func (parser *AtArgumentParser) IsInterfaceNil() bool {
	return parser == nil
}
