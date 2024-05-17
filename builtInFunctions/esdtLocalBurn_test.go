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

func createESDTLocalMintBurnArgs() ESDTLocalMintBurnFuncArgs {
	ctc, _ := NewCrossChainTokenChecker(nil, getWhiteListedAddress())
	return ESDTLocalMintBurnFuncArgs{
		FuncGasCost:            0,
		Marshaller:             &mock.MarshalizerMock{},
		GlobalSettingsHandler:  &mock.GlobalSettingsHandlerStub{},
		RolesHandler:           &mock.ESDTRoleHandlerStub{},
		EnableEpochsHandler:    &mock.EnableEpochsHandlerStub{},
		CrossChainTokenChecker: ctc,
	}
}

func TestNewESDTLocalBurnFunc(t *testing.T) {
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
			_, err := NewESDTLocalBurnFunc(tt.argsFunc())
			require.Equal(t, err, tt.exError)
		})
	}
}

func TestEsdtLocalBurn_ProcessBuiltinFunction_CalledWithValueShouldErr(t *testing.T) {
	t.Parallel()

	esdtLocalBurnF, _ := NewESDTLocalBurnFunc(createESDTLocalMintBurnArgs())

	_, err := esdtLocalBurnF.ProcessBuiltinFunction(&mock.AccountWrapMock{}, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(1),
		},
	})
	require.Equal(t, ErrBuiltInFunctionCalledWithValue, err)
}

func TestEsdtLocalBurn_ProcessBuiltinFunction_CheckAllowToExecuteShouldErr(t *testing.T) {
	t.Parallel()

	localErr := errors.New("local err")
	args := createESDTLocalMintBurnArgs()
	args.RolesHandler = &mock.ESDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			return localErr
		},
	}
	esdtLocalBurnF, _ := NewESDTLocalBurnFunc(args)

	_, err := esdtLocalBurnF.ProcessBuiltinFunction(&mock.AccountWrapMock{}, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
			Arguments: [][]byte{[]byte("arg1"), []byte("arg2")},
		},
	})
	require.Equal(t, localErr, err)
}

func TestEsdtLocalBurn_ProcessBuiltinFunction_CannotAddToEsdtBalanceShouldErr(t *testing.T) {
	t.Parallel()

	args := createESDTLocalMintBurnArgs()
	args.RolesHandler = &mock.ESDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			return nil
		},
	}
	esdtLocalBurnF, _ := NewESDTLocalBurnFunc(args)

	localErr := errors.New("local err")
	_, err := esdtLocalBurnF.ProcessBuiltinFunction(&mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					return nil, 0, localErr
				},
			}
		},
	}, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
			Arguments: [][]byte{[]byte("arg1"), []byte("arg2")},
		},
	})
	require.Equal(t, ErrInsufficientFunds, err)
}

func TestEsdtLocalBurn_ProcessBuiltinFunction_ValueTooLong(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	args := createESDTLocalMintBurnArgs()
	args.FuncGasCost = 50
	args.RolesHandler = &mock.ESDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			assert.Equal(t, core.ESDTRoleLocalBurn, string(action))
			return nil
		},
	}
	esdtLocalBurnF, _ := NewESDTLocalBurnFunc(args)

	sndAccount := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					esdtData := &esdt.ESDigitalToken{Value: big.NewInt(100)}
					serializedEsdtData, err := marshaller.Marshal(esdtData)
					return serializedEsdtData, 0, err
				},
			}
		},
	}

	bigValueStr := "1" + strings.Repeat("0", 1000)
	bigValue, _ := big.NewInt(0).SetString(bigValueStr, 10)
	vmOutput, err := esdtLocalBurnF.ProcessBuiltinFunction(sndAccount, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("arg1"), bigValue.Bytes()},
			GasProvided: 500,
		},
	})
	require.Equal(t, "insufficient funds", err.Error()) // before the activation of the flag
	require.Empty(t, vmOutput)

	// try again with the flag enabled
	esdtLocalBurnF.enableEpochsHandler = &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ConsistentTokensValuesLengthCheckFlag
		},
	}
	vmOutput, err = esdtLocalBurnF.ProcessBuiltinFunction(sndAccount, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("arg1"), bigValue.Bytes()},
			GasProvided: 500,
		},
	})
	require.Equal(t, "invalid arguments to process built-in function: max length for esdt local burn value is 100", err.Error())
	require.Empty(t, vmOutput)
}

func TestEsdtLocalBurn_ProcessBuiltinFunction_ShouldWork(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	args := createESDTLocalMintBurnArgs()
	args.FuncGasCost = 50
	args.RolesHandler = &mock.ESDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			assert.Equal(t, core.ESDTRoleLocalBurn, string(action))
			return nil
		},
	}
	esdtLocalBurnF, _ := NewESDTLocalBurnFunc(args)

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
					require.Equal(t, big.NewInt(99), esdtData.Value)
					return nil
				},
			}
		},
	}
	vmOutput, err := esdtLocalBurnF.ProcessBuiltinFunction(sndAccout, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
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
				Identifier: []byte("ESDTLocalBurn"),
				Address:    nil,
				Topics:     [][]byte{[]byte("arg1"), big.NewInt(0).Bytes(), big.NewInt(1).Bytes()},
				Data:       nil,
			},
		},
	}
	require.Equal(t, expectedVMOutput, vmOutput)
}

func TestEsdtLocalBurn_ProcessBuiltinFunction_WithGlobalBurn(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	args := createESDTLocalMintBurnArgs()
	args.FuncGasCost = 50
	args.GlobalSettingsHandler = &mock.GlobalSettingsHandlerStub{
		IsBurnForAllCalled: func(token []byte) bool {
			return true
		},
	}
	args.RolesHandler = &mock.ESDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			return errors.New("no role")
		},
	}
	esdtLocalBurnF, _ := NewESDTLocalBurnFunc(args)

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
					require.Equal(t, big.NewInt(99), esdtData.Value)
					return nil
				},
			}
		},
	}
	vmOutput, err := esdtLocalBurnF.ProcessBuiltinFunction(sndAccout, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
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
				Identifier: []byte("ESDTLocalBurn"),
				Address:    nil,
				Topics:     [][]byte{[]byte("arg1"), big.NewInt(0).Bytes(), big.NewInt(1).Bytes()},
				Data:       nil,
			},
		},
	}
	require.Equal(t, expectedVMOutput, vmOutput)
}

func TestEsdtLocalBurn_SetNewGasConfig(t *testing.T) {
	t.Parallel()

	esdtLocalBurnF, _ := NewESDTLocalBurnFunc(createESDTLocalMintBurnArgs())

	esdtLocalBurnF.SetNewGasConfig(&vmcommon.GasCost{BuiltInCost: vmcommon.BuiltInCost{
		ESDTLocalBurn: 500},
	})

	require.Equal(t, uint64(500), esdtLocalBurnF.funcGasCost)
}

func TestCheckInputArgumentsForLocalAction_InvalidRecipientAddr(t *testing.T) {
	t.Parallel()

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			Arguments:  [][]byte{[]byte("arg1"), big.NewInt(1).Bytes()},
			CallerAddr: []byte("caller"),
		},
		RecipientAddr: []byte("rec"),
	}

	err := checkInputArgumentsForLocalAction(&mock.UserAccountStub{}, vmInput, 0)
	require.Equal(t, ErrInvalidRcvAddr, err)
}

func TestCheckInputArgumentsForLocalAction_NilUserAccount(t *testing.T) {
	t.Parallel()

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			Arguments:  [][]byte{[]byte("arg1"), big.NewInt(1).Bytes()},
			CallerAddr: []byte("caller"),
		},
		RecipientAddr: []byte("caller"),
	}

	err := checkInputArgumentsForLocalAction(nil, vmInput, 0)
	require.Equal(t, ErrNilUserAccount, err)
}

func TestCheckInputArgumentsForLocalAction_NotEnoughGas(t *testing.T) {
	t.Parallel()

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("arg1"), big.NewInt(10).Bytes()},
			CallerAddr:  []byte("caller"),
			GasProvided: 1,
		},
		RecipientAddr: []byte("caller"),
	}

	err := checkInputArgumentsForLocalAction(&mock.UserAccountStub{}, vmInput, 500)
	require.Equal(t, ErrNotEnoughGas, err)
}

func TestEsdtLocalBurn_ProcessBuiltinFunction_CrossChainOperations(t *testing.T) {
	t.Parallel()

	testEsdtLocalBurnCrossChainOperations(t, nil, []byte("sov1-TKN-abcdef"))
	testEsdtLocalBurnCrossChainOperations(t, []byte("sov2"), []byte("sov1-TKN-abcdef"))
	testEsdtLocalBurnCrossChainOperations(t, []byte("sov1"), []byte("TKN-abcdef"))
}

func testEsdtLocalBurnCrossChainOperations(t *testing.T, selfPrefix, crossChainToken []byte) {
	args := createESDTLocalMintBurnArgs()
	args.FuncGasCost = 50

	whiteListedAddr := make(map[string]struct{})
	if len(selfPrefix) == 0 {
		whiteListedAddr = getWhiteListedAddress()
	}
	args.CrossChainTokenChecker, _ = NewCrossChainTokenChecker(selfPrefix, whiteListedAddr)

	wasAllowedToExecuteCalled := false
	args.RolesHandler = &mock.ESDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			wasAllowedToExecuteCalled = true
			return nil
		},
	}
	wasBurnForAllCalled := false
	args.GlobalSettingsHandler = &mock.GlobalSettingsHandlerStub{
		IsBurnForAllCalled: func(token []byte) bool {
			wasBurnForAllCalled = true
			return false
		},
	}

	esdtLocalBurnF, _ := NewESDTLocalBurnFunc(args)

	initialBalance := big.NewInt(100)
	burnValue := big.NewInt(44)
	wasNewBalanceUpdated := false
	marshaller := args.Marshaller
	senderAcc := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					esdtData := &esdt.ESDigitalToken{Value: initialBalance}
					serializedEsdtData, err := marshaller.Marshal(esdtData)
					return serializedEsdtData, 0, err
				},
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					esdtData := &esdt.ESDigitalToken{}
					_ = marshaller.Unmarshal(esdtData, value)
					require.Equal(t, big.NewInt(0).Sub(initialBalance, burnValue), esdtData.Value)

					wasNewBalanceUpdated = true
					return nil
				},
			}
		},
	}

	vmOutput, err := esdtLocalBurnF.ProcessBuiltinFunction(senderAcc, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{crossChainToken, burnValue.Bytes()},
			GasProvided: 500,
		},
	})
	require.Nil(t, err)
	expectedVMOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: 450,
		Logs: []*vmcommon.LogEntry{
			{
				Identifier: []byte("ESDTLocalBurn"),
				Address:    nil,
				Topics:     [][]byte{crossChainToken, big.NewInt(0).Bytes(), burnValue.Bytes()},
				Data:       nil,
			},
		},
	}
	require.Equal(t, expectedVMOutput, vmOutput)
	require.True(t, wasNewBalanceUpdated)
	require.False(t, wasAllowedToExecuteCalled)
	require.False(t, wasBurnForAllCalled)
}
