package parsers

import "strings"

type callArgsParser struct {
}

// NewCallArgsParser creates a new parser
func NewCallArgsParser() *callArgsParser {
	return &callArgsParser{}
}

// ParseData parses strings of the following format:
// functionRaw@argFooHex@argBarHex...
func (parser *callArgsParser) ParseData(data string) (string, [][]byte, error) {
	var function string
	var arguments [][]byte

	tokens, err := tokenize(data)
	if err != nil {
		return "", nil, err
	}

	function, err = parser.parseFunction(tokens)
	if err != nil {
		return "", nil, err
	}

	arguments, err = parser.parseArguments(tokens)
	if err != nil {
		return "", nil, err
	}

	return function, arguments, nil
}

// ParseArguments parses strings of the following format:
// argFoo@hex(argBarHex)...
func (parser *callArgsParser) ParseArguments(data string) ([][]byte, error) {
	tokens := strings.Split(data, atSeparator)
	arguments := make([][]byte, 0, len(tokens))
	arguments = append(arguments, []byte(tokens[0]))
	parsedArgs, err := parser.parseArguments(tokens)
	if err != nil {
		return nil, err
	}
	arguments = append(arguments, parsedArgs...)

	return arguments, nil
}

func (parser *callArgsParser) parseFunction(tokens []string) (string, error) {
	if len(tokens) < minNumCallArguments {
		return "", ErrNilFunction
	}

	function := tokens[indexOfFunction]
	return function, nil
}

func (parser *callArgsParser) parseArguments(tokens []string) ([][]byte, error) {
	arguments := make([][]byte, 0)

	for i := minNumCallArguments; i < len(tokens); i++ {
		argument, err := decodeToken(tokens[i])
		if err != nil {
			return nil, err
		}

		arguments = append(arguments, argument)
	}

	return arguments, nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (parser *callArgsParser) IsInterfaceNil() bool {
	return parser == nil
}
