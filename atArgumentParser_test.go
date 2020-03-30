package vmcommon

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewAtArgumentParser(t *testing.T) {
	t.Parallel()

	parser := NewAtArgumentParser()
	require.NotNil(t, parser)

	args, err := parser.GetFunctionArguments()
	require.Nil(t, args)
	require.Equal(t, ErrNilArguments, err)

	args, err = parser.GetConstructorArguments()
	require.Nil(t, args)
	require.Equal(t, ErrNilArguments, err)

	code, err := parser.GetCode()
	require.Nil(t, code)
	require.Equal(t, ErrInvalidDeployArguments, err)

	codeMetadata, err := parser.GetCodeMetadata()
	require.Equal(t, CodeMetadata{}, codeMetadata)
	require.Equal(t, ErrInvalidDeployArguments, err)

	function, err := parser.GetFunction()
	require.Equal(t, "", function)
	require.Equal(t, ErrNilFunction, err)
}

func TestAtArgumentParser_GetArguments(t *testing.T) {
	t.Parallel()

	parser := NewAtArgumentParser()
	require.NotNil(t, parser)

	err := parser.ParseData("some_c///ode@aa@bb@bc")
	require.Nil(t, err)

	args, err := parser.GetFunctionArguments()
	require.Nil(t, err)
	require.Equal(t, 3, len(args))

	args, err = parser.GetConstructorArguments()
	require.Nil(t, err)
	require.Equal(t, 1, len(args))
}

func TestAtArgumentParser_GetArgumentsOddLength(t *testing.T) {
	t.Parallel()

	parser := NewAtArgumentParser()
	require.NotNil(t, parser)

	err := parser.ParseData("aaaa@a@bb@bc@d")
	require.Nil(t, err)

	args, err := parser.GetFunctionArguments()
	require.Nil(t, err)
	require.Equal(t, 4, len(args))

	args, err = parser.GetConstructorArguments()
	require.Nil(t, err)
	require.Equal(t, 2, len(args))
}

func TestAtArgumentParser_GetArgumentsEmpty(t *testing.T) {
	t.Parallel()

	parser := NewAtArgumentParser()
	require.NotNil(t, parser)

	err := parser.ParseData("aaaa")
	require.Nil(t, err)

	args, err := parser.GetFunctionArguments()
	require.Nil(t, err)
	require.Equal(t, 0, len(args))

	args, err = parser.GetConstructorArguments()
	require.Equal(t, err, ErrNilArguments)
}

func TestAtArgumentParser_GetEmptyArgument1(t *testing.T) {
	t.Parallel()

	parser := NewAtArgumentParser()
	require.NotNil(t, parser)

	err := parser.ParseData("aaaa@")
	require.Nil(t, err)

	args, err := parser.GetFunctionArguments()
	require.Nil(t, err)
	require.NotNil(t, args)
	require.Equal(t, 1, len(args))
	require.Equal(t, 0, len(args[0]))
}

func TestAtArgumentParser_GetEmptyArgument2(t *testing.T) {
	t.Parallel()

	parser := NewAtArgumentParser()
	require.NotNil(t, parser)

	err := parser.ParseData("aaaa@@0123")
	require.Nil(t, err)

	args, err := parser.GetFunctionArguments()
	require.Nil(t, err)
	require.NotNil(t, args)
	require.Equal(t, 2, len(args))
	require.Equal(t, 0, len(args[0]))
}

func TestAtArgumentParser_GetEmptyArgument3(t *testing.T) {
	t.Parallel()

	parser := NewAtArgumentParser()
	require.NotNil(t, parser)

	err := parser.ParseData("aaaa@12@@0123@@")
	require.Nil(t, err)

	args, err := parser.GetFunctionArguments()
	require.Nil(t, err)
	require.NotNil(t, args)
	require.Equal(t, 5, len(args))
	require.Equal(t, 1, len(args[0]))
	require.Equal(t, 0, len(args[1]))
	require.Equal(t, 2, len(args[2]))
	require.Equal(t, 0, len(args[3]))
	require.Equal(t, 0, len(args[4]))
}

func TestAtArgumentParser_GetCode(t *testing.T) {
	t.Parallel()

	parser := NewAtArgumentParser()
	require.NotNil(t, parser)

	err := parser.ParseData("abba@0123@0100@64")
	require.Nil(t, err)

	code, err := parser.GetCode()
	require.Nil(t, err)
	require.Equal(t, []byte("abba"), code)

	vmType, err := parser.GetVMType()
	require.Nil(t, err)
	require.Equal(t, []byte{0x01, 0x23}, vmType)

	codeMetadata, err := parser.GetCodeMetadata()
	require.Nil(t, err)
	require.True(t, codeMetadata.Upgradeable)

	constructorArgs, err := parser.GetConstructorArguments()
	require.Nil(t, err)
	require.EqualValues(t, [][]byte{[]byte{100}}, constructorArgs)
}

func TestAtArgumentParser_GetCodeEmpty(t *testing.T) {
	t.Parallel()

	parser := NewAtArgumentParser()
	require.NotNil(t, parser)

	err := parser.ParseData("@aaaa")
	require.Equal(t, ErrTokenizeFailed, err)

	code, err := parser.GetCode()
	require.Equal(t, ErrInvalidDeployArguments, err)
	require.Nil(t, code)
}

func TestAtArgumentParser_GetFunction(t *testing.T) {
	t.Parallel()

	parser := NewAtArgumentParser()
	require.NotNil(t, parser)

	err := parser.ParseData("fooBar@aaaa")
	require.Nil(t, err)

	function, err := parser.GetFunction()
	require.Nil(t, err)
	require.Equal(t, []byte("fooBar"), []byte(function))
}

func TestAtArgumentParser_GetFunctionEmpty(t *testing.T) {
	t.Parallel()

	parser := NewAtArgumentParser()
	require.NotNil(t, parser)

	err := parser.ParseData("@a")
	require.Equal(t, ErrTokenizeFailed, err)

	function, err := parser.GetFunction()
	require.Equal(t, ErrNilFunction, err)
	require.Equal(t, 0, len(function))
}

func TestAtArgumentParser_ParseData(t *testing.T) {
	t.Parallel()

	parser := NewAtArgumentParser()
	require.NotNil(t, parser)

	err := parser.ParseData("ab")
	require.Nil(t, err)
}

func TestAtArgumentParser_ParseDataEmpty(t *testing.T) {
	t.Parallel()

	parser := NewAtArgumentParser()
	require.NotNil(t, parser)

	err := parser.ParseData("")
	require.Equal(t, ErrTokenizeFailed, err)
}

func TestAtArgumentParser_CreateDataFromStorageUpdate(t *testing.T) {
	t.Parallel()

	parser := NewAtArgumentParser()
	require.NotNil(t, parser)

	data := parser.CreateDataFromStorageUpdate(nil)
	require.Equal(t, 0, len(data))

	test := []byte("aaaa")
	stUpd := StorageUpdate{Offset: test, Data: test}
	stUpdates := make([]*StorageUpdate, 0)
	stUpdates = append(stUpdates, &stUpd, &stUpd, &stUpd)
	result := ""
	sep := "@"
	result = result + hex.EncodeToString(test)
	result = result + sep
	result = result + hex.EncodeToString(test)
	result = result + sep
	result = result + hex.EncodeToString(test)
	result = result + sep
	result = result + hex.EncodeToString(test)
	result = result + sep
	result = result + hex.EncodeToString(test)
	result = result + sep
	result = result + hex.EncodeToString(test)

	data = parser.CreateDataFromStorageUpdate(stUpdates)

	require.Equal(t, result, data)
}

func TestAtArgumentParser_GetStorageUpdatesEmptyData(t *testing.T) {
	t.Parallel()

	parser := NewAtArgumentParser()
	require.NotNil(t, parser)

	stUpdates, err := parser.GetStorageUpdates("")

	require.Nil(t, stUpdates)
	require.Equal(t, ErrTokenizeFailed, err)
}

func TestAtArgumentParser_GetStorageUpdatesWrongData(t *testing.T) {
	t.Parallel()

	parser := NewAtArgumentParser()
	require.NotNil(t, parser)

	test := "test"
	result := ""
	sep := "@"
	result = result + test
	result = result + sep
	result = result + test
	result = result + sep
	result = result + test
	result = result + sep
	result = result + test
	result = result + sep
	result = result + test

	stUpdates, err := parser.GetStorageUpdates(result)

	require.Nil(t, stUpdates)
	require.Equal(t, ErrInvalidDataString, err)
}

func TestAtArgumentParser_GetStorageUpdates(t *testing.T) {
	t.Parallel()

	parser := NewAtArgumentParser()
	require.NotNil(t, parser)

	test := "aaaa"
	result := ""
	sep := "@"
	result = result + test
	result = result + sep
	result = result + test
	result = result + sep
	result = result + test
	result = result + sep
	result = result + test
	result = result + sep
	result = result + test
	result = result + sep
	result = result + test
	stUpdates, err := parser.GetStorageUpdates(result)

	require.Nil(t, err)
	for i := 0; i < 2; i++ {
		require.Equal(t, test, hex.EncodeToString(stUpdates[i].Data))
		require.Equal(t, test, hex.EncodeToString(stUpdates[i].Offset))
	}
}
