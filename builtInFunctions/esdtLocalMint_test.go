package builtInFunctions

import (
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/data/esdt"
	"github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewESDTLocalMintFunc(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		argsFunc func() ESDTLocalMintBurnFuncArgs
		exError  error
	}{
		{
			name: "NilMarshalizer",
			argsFunc: func() ESDTLocalMintBurnFuncArgs {
				args := createESDTLocalMintBurnArgs()
				args.Marshaller = nil

				return args
			},
			exError: ErrNilMarshalizer,
		},
		{
			name: "NilGlobalSettingsHandler",
			argsFunc: func() ESDTLocalMintBurnFuncArgs {
				args := createESDTLocalMintBurnArgs()
				args.GlobalSettingsHandler = nil

				return args
			},
			exError: ErrNilGlobalSettingsHandler,
		},
		{
			name: "NilRolesHandler",
			argsFunc: func() ESDTLocalMintBurnFuncArgs {
				args := createESDTLocalMintBurnArgs()
				args.RolesHandler = nil

				return args
			},
			exError: ErrNilRolesHandler,
		},
		{
			name: "NilEnableEpochsHandler",
			argsFunc: func() ESDTLocalMintBurnFuncArgs {
				args := createESDTLocalMintBurnArgs()
				args.EnableEpochsHandler = nil

				return args
			},
			exError: ErrNilEnableEpochsHandler,
		},
		{
			name: "NilCrossChainTokenChecker",
			argsFunc: func() ESDTLocalMintBurnFuncArgs {
				args := createESDTLocalMintBurnArgs()
				args.CrossChainTokenChecker = nil

				return args
			},
			exError: ErrNilCrossChainTokenChecker,
		},
		{
			name: "Ok",
			argsFunc: func() ESDTLocalMintBurnFuncArgs {
				return createESDTLocalMintBurnArgs()
			},
			exError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewESDTLocalMintFunc(tt.argsFunc())
			require.Equal(t, err, tt.exError)
		})
	}
}

func TestEsdtLocalMint_SetNewGasConfig(t *testing.T) {
	t.Parallel()

	esdtLocalMintF, _ := NewESDTLocalMintFunc(createESDTLocalMintBurnArgs())

	esdtLocalMintF.SetNewGasConfig(&vmcommon.GasCost{BuiltInCost: vmcommon.BuiltInCost{
		ESDTLocalMint: 500},
	})

	require.Equal(t, uint64(500), esdtLocalMintF.funcGasCost)
}

func TestEsdtLocalMint_ProcessBuiltinFunction_CalledWithValueShouldErr(t *testing.T) {
	t.Parallel()

	esdtLocalMintF, _ := NewESDTLocalMintFunc(createESDTLocalMintBurnArgs())

	_, err := esdtLocalMintF.ProcessBuiltinFunction(&mock.AccountWrapMock{}, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(1),
		},
	})
	require.Equal(t, ErrBuiltInFunctionCalledWithValue, err)
}

func TestEsdtLocalMint_ProcessBuiltinFunction_CheckAllowToExecuteShouldErr(t *testing.T) {
	t.Parallel()

	localErr := errors.New("local err")
	args := createESDTLocalMintBurnArgs()
	args.RolesHandler = &mock.ESDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			return localErr
		},
	}
	esdtLocalMintF, _ := NewESDTLocalMintFunc(args)

	_, err := esdtLocalMintF.ProcessBuiltinFunction(&mock.AccountWrapMock{}, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
			Arguments: [][]byte{[]byte("arg1"), []byte("arg2")},
		},
	})
	require.Equal(t, localErr, err)
}

func TestEsdtLocalMint_ProcessBuiltinFunction_CannotAddToEsdtBalanceShouldErr(t *testing.T) {
	t.Parallel()

	args := createESDTLocalMintBurnArgs()
	args.RolesHandler = &mock.ESDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			return nil
		},
	}
	esdtLocalMintF, _ := NewESDTLocalMintFunc(args)

	localErr := errors.New("local err")
	_, err := esdtLocalMintF.ProcessBuiltinFunction(&mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					return nil, 0, localErr
				},
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					return localErr
				},
			}
		},
	}, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
			Arguments: [][]byte{[]byte("arg1"), big.NewInt(1).Bytes()},
		},
	})
	require.Equal(t, localErr, err)
}

func TestEsdtLocalMint_ProcessBuiltinFunction_ValueTooLong(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	args := createESDTLocalMintBurnArgs()
	args.RolesHandler = &mock.ESDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			assert.Equal(t, core.ESDTRoleLocalMint, string(action))
			return nil
		},
	}
	args.FuncGasCost = 50
	esdtLocalMintF, _ := NewESDTLocalMintFunc(args)

	sndAccount := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					esdtData := &esdt.ESDigitalToken{Value: big.NewInt(100)}
					serializedEsdtData, err := marshaller.Marshal(esdtData)
					return serializedEsdtData, 0, err
				},
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					esdtData := &esdt.ESDigitalToken{}
					_ = marshaller.Unmarshal(esdtData, value)
					//require.Equal(t, big.NewInt(101), esdtData.Value)
					return nil
				},
			}
		},
	}
	bigValueStr := "1" + strings.Repeat("0", 1000)
	bigValue, _ := big.NewInt(0).SetString(bigValueStr, 10)
	vmOutput, err := esdtLocalMintF.ProcessBuiltinFunction(sndAccount, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("arg1"), bigValue.Bytes()},
			GasProvided: 500,
		},
	})
	require.Equal(t, "invalid arguments to process built-in function max length for esdt issue is 100", err.Error())
	require.Empty(t, vmOutput)

	// try again with the flag enabled
	esdtLocalMintF.enableEpochsHandler = &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ConsistentTokensValuesLengthCheckFlag
		},
	}
	vmOutput, err = esdtLocalMintF.ProcessBuiltinFunction(sndAccount, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("arg1"), bigValue.Bytes()},
			GasProvided: 500,
		},
	})
	require.Equal(t, "invalid arguments to process built-in function: max length for esdt local mint value is 100", err.Error())
	require.Empty(t, vmOutput)
}

func TestEsdtLocalMint_ProcessBuiltinFunction_ShouldWork(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}

	args := createESDTLocalMintBurnArgs()
	args.RolesHandler = &mock.ESDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			assert.Equal(t, core.ESDTRoleLocalMint, string(action))
			return nil
		},
	}
	args.FuncGasCost = 50
	esdtLocalMintF, _ := NewESDTLocalMintFunc(args)

	sndAccout := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					esdtData := &esdt.ESDigitalToken{Value: big.NewInt(100)}
					serializedEsdtData, err := marshaller.Marshal(esdtData)
					return serializedEsdtData, 0, err
				},
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					esdtData := &esdt.ESDigitalToken{}
					_ = marshaller.Unmarshal(esdtData, value)
					require.Equal(t, big.NewInt(101), esdtData.Value)
					return nil
				},
			}
		},
	}
	vmOutput, err := esdtLocalMintF.ProcessBuiltinFunction(sndAccout, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("arg1"), big.NewInt(1).Bytes()},
			GasProvided: 500,
		},
	})
	require.Equal(t, nil, err)

	expectedVMOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: 450,
		Logs: []*vmcommon.LogEntry{
			{
				Identifier: []byte("ESDTLocalMint"),
				Address:    nil,
				Topics:     [][]byte{[]byte("arg1"), big.NewInt(0).Bytes(), big.NewInt(1).Bytes()},
				Data:       nil,
			},
		},
	}
	require.Equal(t, expectedVMOutput, vmOutput)

	mintTooMuch := make([]byte, 101)
	mintTooMuch[0] = 1
	vmOutput, err = esdtLocalMintF.ProcessBuiltinFunction(sndAccout, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("arg1"), mintTooMuch},
			GasProvided: 500,
		},
	})
	require.True(t, errors.Is(err, ErrInvalidArguments))
	require.Nil(t, vmOutput)
}

func TestEsdtLocalMint_ProcessBuiltinFunction_ShouldMintCrossChainTokenInSelfMainChain(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}

	args := createESDTLocalMintBurnArgs()
	args.RolesHandler = &mock.ESDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			require.Fail(t, "should not check here, should only check if cross operation and self chain == main chain")
			return nil
		},
	}
	args.FuncGasCost = 50
	esdtLocalMintF, _ := NewESDTLocalMintFunc(args)

	tokenID := []byte("pref-TKNX-abcdef")
	initialSupply := big.NewInt(100)
	mintQuantity := big.NewInt(1)
	sndAccout := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					esdtData := &esdt.ESDigitalToken{Value: initialSupply}
					serializedEsdtData, err := marshaller.Marshal(esdtData)
					return serializedEsdtData, 0, err
				},
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					esdtData := &esdt.ESDigitalToken{}
					_ = marshaller.Unmarshal(esdtData, value)
					require.Equal(t, big.NewInt(0).Add(initialSupply, mintQuantity), esdtData.Value)
					return nil
				},
			}
		},
	}

	initialGas := uint64(500)
	vmOutput, err := esdtLocalMintF.ProcessBuiltinFunction(sndAccout, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{tokenID, mintQuantity.Bytes()},
			GasProvided: initialGas,
		},
	})
	require.Equal(t, nil, err)

	expectedVMOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: initialGas - args.FuncGasCost,
		Logs: []*vmcommon.LogEntry{
			{
				Identifier: []byte("ESDTLocalMint"),
				Address:    nil,
				Topics:     [][]byte{tokenID, big.NewInt(0).Bytes(), mintQuantity.Bytes()},
				Data:       nil,
			},
		},
	}
	require.Equal(t, expectedVMOutput, vmOutput)
}

func TestEsdtLocalMint_ProcessBuiltinFunction_ShouldNotMintCrossChainTokenInSovereignChain(t *testing.T) {
	t.Parallel()

	args := createESDTLocalMintBurnArgs()
	args.CrossChainTokenChecker, _ = NewCrossChainTokenChecker([]byte("self"))
	errNotAllowedToMint := errors.New("not allowed")
	args.RolesHandler = &mock.ESDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			return errNotAllowedToMint
		},
	}
	esdtLocalMintF, _ := NewESDTLocalMintFunc(args)

	// Cross chain token from another sovereign chain
	vmOutput, err := esdtLocalMintF.ProcessBuiltinFunction(&mock.AccountWrapMock{}, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("pref-TKNX-abcdef"), big.NewInt(1).Bytes()},
			GasProvided: 500,
		},
	})
	require.Equal(t, errNotAllowedToMint, err)
	require.Nil(t, vmOutput)

	// Cross chain token from main chain
	vmOutput, err = esdtLocalMintF.ProcessBuiltinFunction(&mock.AccountWrapMock{}, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("TKNX-abcdef"), big.NewInt(1).Bytes()},
			GasProvided: 500,
		},
	})
	require.Equal(t, errNotAllowedToMint, err)
	require.Nil(t, vmOutput)
}
