package parsers

import (
	"encoding/hex"

	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

// atArgumentParser is a parser that splits arguments by @ character
// [NotConcurrentSafe]
type atArgumentParser struct {
	// First argument is a string (function name or hex-encoded bytecode), the rest are raw bytes
	arguments [][]byte
}

// NewAtArgumentParser creates a new parser
func NewAtArgumentParser() *atArgumentParser {
	parser := &atArgumentParser{}
	parser.clearArguments()
	return parser
}

func (parser *atArgumentParser) clearArguments() {
	parser.arguments = make([][]byte, 0)
}

// ParseData parses strings of the following formats:
// contract deploy: codeHex@vmTypeHex@codeMetadataHex@argFooHex@argBarHex...
// contract call: functionRaw@argFooHex@argBarHex...
func (parser *atArgumentParser) ParseData(data string) error {
	parser.clearArguments()

	tokens, err := tokenize(data)
	if err != nil {
		return err
	}

	// First argument is not decoded, but left as it is (function or codeHex)
	parser.arguments = append(parser.arguments, []byte(tokens[0]))

	for i := 1; i < len(tokens); i++ {
		argument, err := decodeTokenPermissive(tokens[i])
		if err != nil {
			return err
		}

		parser.arguments = append(parser.arguments, argument)
	}

	return nil
}

// GetFunctionArguments returns the call arguments
func (parser *atArgumentParser) GetFunctionArguments() ([][]byte, error) {
	if len(parser.arguments) < startIndexOfFunctionArguments {
		return nil, ErrNilArguments
	}

	args := parser.arguments[startIndexOfFunctionArguments:]
	return args, nil
}

// GetConstructorArguments returns the deploy arguments
func (parser *atArgumentParser) GetConstructorArguments() ([][]byte, error) {
	if len(parser.arguments) < startIndexOfConstructorArguments {
		return nil, ErrNilArguments
	}

	args := parser.arguments[startIndexOfConstructorArguments:]
	return args, nil
}

// GetCode returns the hex-encoded code from the parsed data
func (parser *atArgumentParser) GetCode() ([]byte, error) {
	if len(parser.arguments) < minNumDeployArguments {
		return nil, ErrInvalidDeployArguments
	}

	hexCode := parser.arguments[indexOfCode]
	return hexCode, nil
}

// GetCodeDecoded returns the code from the parsed data, hex-decoded
func (parser *atArgumentParser) GetCodeDecoded() ([]byte, error) {
	codeHex, err := parser.GetCode()
	if err != nil {
		return nil, err
	}

	code, err := hex.DecodeString(string(codeHex))
	if err != nil {
		return nil, err
	}

	return code, err
}

// GetVMType returns the VM type from the parsed data
func (parser *atArgumentParser) GetVMType() ([]byte, error) {
	if len(parser.arguments) < minNumDeployArguments {
		return nil, ErrInvalidDeployArguments
	}

	vmType := parser.arguments[indexOfVMType]
	if len(vmType) == 0 {
		return nil, ErrInvalidVMType
	}

	return vmType, nil
}

// GetCodeMetadata returns the code metadata from the parsed data
func (parser *atArgumentParser) GetCodeMetadata() (vmcommon.CodeMetadata, error) {
	if len(parser.arguments) < minNumDeployArguments {
		return vmcommon.CodeMetadata{}, ErrInvalidDeployArguments
	}

	codeMetadataBytes := parser.arguments[indexOfCodeMetadata]
	codeMetadata := vmcommon.CodeMetadataFromBytes(codeMetadataBytes)
	return codeMetadata, nil
}

// GetFunction returns the function from the parsed data
func (parser *atArgumentParser) GetFunction() (string, error) {
	if len(parser.arguments) < minNumCallArguments {
		return "", ErrNilFunction
	}

	function := string(parser.arguments[indexOfFunction])
	return function, nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (parser *atArgumentParser) IsInterfaceNil() bool {
	return parser == nil
}
