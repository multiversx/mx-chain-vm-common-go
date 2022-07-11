package builtInFunctions

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data/esdt"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createESDTNFTMultiTransferWithStubArguments() *esdtNFTMultiTransfer {
	multiTransfer, _ := NewESDTNFTMultiTransferFunc(
		0,
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.AccountsStub{},
		&mock.ShardCoordinatorStub{},
		vmcommon.BaseOperationCost{},
		0,
		&mock.EpochNotifierStub{},
		&mock.ESDTRoleHandlerStub{},
		1000,
		0,
		0,
		createNewESDTDataStorageHandler(),
	)

	return multiTransfer
}

func createAccountsAdapterWithMap() vmcommon.AccountsAdapter {
	mapAccounts := make(map[string]vmcommon.UserAccountHandler)
	accounts := &mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			_, ok := mapAccounts[string(address)]
			if !ok {
				mapAccounts[string(address)] = mock.NewUserAccount(address)
			}
			return mapAccounts[string(address)], nil
		},
		GetExistingAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			_, ok := mapAccounts[string(address)]
			if !ok {
				mapAccounts[string(address)] = mock.NewUserAccount(address)
			}
			return mapAccounts[string(address)], nil
		},
		SaveAccountCalled: func(account vmcommon.AccountHandler) error {
			mapAccounts[string(account.AddressBytes())] = account.(vmcommon.UserAccountHandler)
			return nil
		},
	}
	return accounts
}

func createESDTNFTMultiTransferWithMockArguments(selfShard uint32, numShards uint32, globalSettingsHandler vmcommon.ESDTGlobalSettingsHandler) *esdtNFTMultiTransfer {
	marshalizer := &mock.MarshalizerMock{}
	shardCoordinator := mock.NewMultiShardsCoordinatorMock(numShards)
	shardCoordinator.CurrentShard = selfShard
	shardCoordinator.ComputeIdCalled = func(address []byte) uint32 {
		lastByte := uint32(address[len(address)-1])
		return lastByte
	}
	accounts := createAccountsAdapterWithMap()

	multiTransfer, _ := NewESDTNFTMultiTransferFunc(
		1,
		marshalizer,
		globalSettingsHandler,
		accounts,
		shardCoordinator,
		vmcommon.BaseOperationCost{},
		0,
		&mock.EpochNotifierStub{},
		&mock.ESDTRoleHandlerStub{
			CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
				if bytes.Equal(action, []byte(core.ESDTRoleTransfer)) {
					return ErrActionNotAllowed
				}
				return nil
			},
		},
		1000,
		0,
		0,
		createNewESDTDataStorageHandlerWithArgs(globalSettingsHandler, accounts),
	)

	return multiTransfer
}

func TestNewESDTNFTMultiTransferFunc_NilArgumentsShouldErr(t *testing.T) {
	t.Parallel()

	multiTransfer, err := NewESDTNFTMultiTransferFunc(
		0,
		nil,
		&mock.GlobalSettingsHandlerStub{},
		&mock.AccountsStub{},
		&mock.ShardCoordinatorStub{},
		vmcommon.BaseOperationCost{},
		0,
		&mock.EpochNotifierStub{},
		&mock.ESDTRoleHandlerStub{},
		1000,
		0,
		0,
		createNewESDTDataStorageHandler(),
	)
	assert.True(t, check.IfNil(multiTransfer))
	assert.Equal(t, ErrNilMarshalizer, err)

	multiTransfer, err = NewESDTNFTMultiTransferFunc(
		0,
		&mock.MarshalizerMock{},
		nil,
		&mock.AccountsStub{},
		&mock.ShardCoordinatorStub{},
		vmcommon.BaseOperationCost{},
		0,
		&mock.EpochNotifierStub{},
		&mock.ESDTRoleHandlerStub{},
		1000,
		0,
		0,
		createNewESDTDataStorageHandler(),
	)
	assert.True(t, check.IfNil(multiTransfer))
	assert.Equal(t, ErrNilGlobalSettingsHandler, err)

	multiTransfer, err = NewESDTNFTMultiTransferFunc(
		0,
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		nil,
		&mock.ShardCoordinatorStub{},
		vmcommon.BaseOperationCost{},
		0,
		&mock.EpochNotifierStub{},
		&mock.ESDTRoleHandlerStub{},
		1000,
		0,
		0,
		createNewESDTDataStorageHandler(),
	)
	assert.True(t, check.IfNil(multiTransfer))
	assert.Equal(t, ErrNilAccountsAdapter, err)

	multiTransfer, err = NewESDTNFTMultiTransferFunc(
		0,
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.AccountsStub{},
		nil,
		vmcommon.BaseOperationCost{},
		0,
		&mock.EpochNotifierStub{},
		&mock.ESDTRoleHandlerStub{},
		1000,
		0,
		0,
		createNewESDTDataStorageHandler(),
	)
	assert.True(t, check.IfNil(multiTransfer))
	assert.Equal(t, ErrNilShardCoordinator, err)

	multiTransfer, err = NewESDTNFTMultiTransferFunc(
		0,
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.AccountsStub{},
		&mock.ShardCoordinatorStub{},
		vmcommon.BaseOperationCost{},
		0,
		nil,
		&mock.ESDTRoleHandlerStub{},
		1000,
		0,
		0,
		createNewESDTDataStorageHandler(),
	)
	assert.True(t, check.IfNil(multiTransfer))
	assert.Equal(t, ErrNilEpochHandler, err)

	multiTransfer, err = NewESDTNFTMultiTransferFunc(
		0,
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.AccountsStub{},
		&mock.ShardCoordinatorStub{},
		vmcommon.BaseOperationCost{},
		0,
		&mock.EpochNotifierStub{},
		nil,
		1000,
		0,
		0,
		createNewESDTDataStorageHandler(),
	)
	assert.True(t, check.IfNil(multiTransfer))
	assert.Equal(t, ErrNilRolesHandler, err)

	multiTransfer, err = NewESDTNFTMultiTransferFunc(
		0,
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.AccountsStub{},
		&mock.ShardCoordinatorStub{},
		vmcommon.BaseOperationCost{},
		0,
		&mock.EpochNotifierStub{},
		&mock.ESDTRoleHandlerStub{},
		1000,
		0,
		0,
		nil,
	)
	assert.True(t, check.IfNil(multiTransfer))
	assert.Equal(t, ErrNilESDTNFTStorageHandler, err)
}

func TestNewESDTNFTMultiTransferFunc(t *testing.T) {
	t.Parallel()

	multiTransfer, err := NewESDTNFTMultiTransferFunc(
		0,
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.AccountsStub{},
		&mock.ShardCoordinatorStub{},
		vmcommon.BaseOperationCost{},
		0,
		&mock.EpochNotifierStub{},
		&mock.ESDTRoleHandlerStub{},
		1000,
		0,
		0,
		createNewESDTDataStorageHandler(),
	)
	assert.False(t, check.IfNil(multiTransfer))
	assert.Nil(t, err)
}

func TestESDTNFTMultiTransfer_SetPayable(t *testing.T) {
	t.Parallel()

	multiTransfer := createESDTNFTMultiTransferWithStubArguments()
	err := multiTransfer.SetPayableHandler(nil)
	assert.Equal(t, ErrNilPayableHandler, err)

	handler := &mock.PayableHandlerStub{}
	err = multiTransfer.SetPayableHandler(handler)
	assert.Nil(t, err)
	assert.True(t, handler == multiTransfer.payableHandler) // pointer testing
}

func TestESDTNFTMultiTransfer_SetNewGasConfig(t *testing.T) {
	t.Parallel()

	multiTransfer := createESDTNFTMultiTransferWithStubArguments()
	multiTransfer.SetNewGasConfig(nil)
	assert.Equal(t, uint64(0), multiTransfer.funcGasCost)
	assert.Equal(t, vmcommon.BaseOperationCost{}, multiTransfer.gasConfig)

	gasCost := createMockGasCost()
	multiTransfer.SetNewGasConfig(&gasCost)
	assert.Equal(t, gasCost.BuiltInCost.ESDTNFTMultiTransfer, multiTransfer.funcGasCost)
	assert.Equal(t, gasCost.BaseOperationCost, multiTransfer.gasConfig)
}

func TestESDTNFTMultiTransfer_ProcessBuiltinFunctionInvalidArgumentsShouldErr(t *testing.T) {
	t.Parallel()

	multiTransfer := createESDTNFTMultiTransferWithStubArguments()
	vmOutput, err := multiTransfer.ProcessBuiltinFunction(&mock.UserAccountStub{}, &mock.UserAccountStub{}, nil)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrNilVmInput, err)

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
			Arguments: [][]byte{[]byte("arg1"), []byte("arg2")},
		},
	}
	vmOutput, err = multiTransfer.ProcessBuiltinFunction(&mock.UserAccountStub{}, &mock.UserAccountStub{}, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrInvalidArguments, err)

	multiTransfer.shardCoordinator = &mock.ShardCoordinatorStub{ComputeIdCalled: func(address []byte) uint32 {
		return core.MetachainShardId
	}}

	token1 := []byte("token")
	senderAddress := bytes.Repeat([]byte{2}, 32)
	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{core.ESDTSCAddress, big.NewInt(1).Bytes(), token1, big.NewInt(1).Bytes(), big.NewInt(1).Bytes()},
			GasProvided: 1,
		},
		RecipientAddr: senderAddress,
	}
	vmOutput, err = multiTransfer.ProcessBuiltinFunction(&mock.UserAccountStub{}, &mock.UserAccountStub{}, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrInvalidRcvAddr, err)
}

func TestESDTNFTMultiTransfer_ProcessBuiltinFunctionOnSameShardWithScCall(t *testing.T) {
	t.Parallel()

	multiTransfer := createESDTNFTMultiTransferWithMockArguments(0, 1, &mock.GlobalSettingsHandlerStub{})
	_ = multiTransfer.SetPayableHandler(
		&mock.PayableHandlerStub{
			IsPayableCalled: func(address []byte) (bool, error) {
				return true, nil
			},
		})
	senderAddress := bytes.Repeat([]byte{2}, 32)
	destinationAddress := bytes.Repeat([]byte{0}, 32)
	destinationAddress[25] = 1
	sender, err := multiTransfer.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)
	destination, err := multiTransfer.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	token1 := []byte("token1")
	token2 := []byte("token2")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, initialTokens, multiTransfer.marshalizer, sender.(vmcommon.UserAccountHandler))
	createESDTNFTToken(token2, core.Fungible, 0, initialTokens, multiTransfer.marshalizer, sender.(vmcommon.UserAccountHandler))

	createESDTNFTToken(token1, core.NonFungible, tokenNonce, initialTokens, multiTransfer.marshalizer, destination.(vmcommon.UserAccountHandler))
	createESDTNFTToken(token2, core.Fungible, 0, initialTokens, multiTransfer.marshalizer, destination.(vmcommon.UserAccountHandler))

	_ = multiTransfer.accounts.SaveAccount(sender)
	_ = multiTransfer.accounts.SaveAccount(destination)
	_, _ = multiTransfer.accounts.Commit()

	// reload accounts
	sender, err = multiTransfer.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)
	destination, err = multiTransfer.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	scCallFunctionAsHex := hex.EncodeToString([]byte("functionToCall"))
	scCallArg := hex.EncodeToString([]byte("arg"))
	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	scCallArgs := [][]byte{[]byte(scCallFunctionAsHex), []byte(scCallArg)}
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{destinationAddress, big.NewInt(2).Bytes(), token1, nonceBytes, quantityBytes, token2, big.NewInt(0).Bytes(), quantityBytes},
			GasProvided: 100000,
		},
		RecipientAddr: senderAddress,
	}
	vmInput.Arguments = append(vmInput.Arguments, scCallArgs...)

	vmOutput, err := multiTransfer.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), destination.(vmcommon.UserAccountHandler), vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)

	_ = multiTransfer.accounts.SaveAccount(sender)
	_, _ = multiTransfer.accounts.Commit()

	// reload accounts
	sender, err = multiTransfer.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)
	destination, err = multiTransfer.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	testNFTTokenShouldExist(t, multiTransfer.marshalizer, sender, token1, tokenNonce, big.NewInt(2)) // 3 initial - 1 transferred
	testNFTTokenShouldExist(t, multiTransfer.marshalizer, sender, token2, 0, big.NewInt(2))
	testNFTTokenShouldExist(t, multiTransfer.marshalizer, destination, token1, tokenNonce, big.NewInt(4))
	testNFTTokenShouldExist(t, multiTransfer.marshalizer, destination, token2, 0, big.NewInt(4))
	funcName, args := extractScResultsFromVmOutput(t, vmOutput)
	assert.Equal(t, scCallFunctionAsHex, funcName)
	require.Equal(t, 1, len(args))
	require.Equal(t, []byte(scCallArg), args[0])
}

func TestESDTNFTMultiTransfer_ProcessBuiltinFunctionOnCrossShardsDestinationDoesNotHoldingNFTWithSCCall(t *testing.T) {
	t.Parallel()

	payableHandler := &mock.PayableHandlerStub{
		IsPayableCalled: func(address []byte) (bool, error) {
			return true, nil
		},
	}

	multiTransferSenderShard := createESDTNFTMultiTransferWithMockArguments(1, 2, &mock.GlobalSettingsHandlerStub{})
	_ = multiTransferSenderShard.SetPayableHandler(payableHandler)

	multiTransferDestinationShard := createESDTNFTMultiTransferWithMockArguments(0, 2, &mock.GlobalSettingsHandlerStub{})
	_ = multiTransferDestinationShard.SetPayableHandler(payableHandler)

	senderAddress := bytes.Repeat([]byte{1}, 32)
	destinationAddress := bytes.Repeat([]byte{0}, 32)
	destinationAddress[25] = 1
	sender, err := multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	token1 := []byte("token1")
	token2 := []byte("token2")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, initialTokens, multiTransferSenderShard.marshalizer, sender.(vmcommon.UserAccountHandler))
	createESDTNFTToken(token2, core.Fungible, 0, initialTokens, multiTransferSenderShard.marshalizer, sender.(vmcommon.UserAccountHandler))
	_ = multiTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = multiTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	scCallFunctionAsHex := hex.EncodeToString([]byte("functionToCall"))
	scCallArg := hex.EncodeToString([]byte("arg"))
	scCallArgs := [][]byte{[]byte(scCallFunctionAsHex), []byte(scCallArg)}
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{destinationAddress, big.NewInt(2).Bytes(), token1, nonceBytes, quantityBytes, token2, big.NewInt(0).Bytes(), quantityBytes},
			GasProvided: 1000000,
		},
		RecipientAddr: senderAddress,
	}
	vmInput.Arguments = append(vmInput.Arguments, scCallArgs...)

	vmOutput, err := multiTransferSenderShard.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), nil, vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)

	_ = multiTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = multiTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	testNFTTokenShouldExist(t, multiTransferSenderShard.marshalizer, sender, token1, tokenNonce, big.NewInt(2)) // 3 initial - 1 transferred

	funcName, args := extractScResultsFromVmOutput(t, vmOutput)

	destination, err := multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: senderAddress,
			Arguments:  args,
		},
		RecipientAddr: destinationAddress,
	}

	vmOutput, err = multiTransferDestinationShard.ProcessBuiltinFunction(nil, destination.(vmcommon.UserAccountHandler), vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)
	_ = multiTransferDestinationShard.accounts.SaveAccount(destination)
	_, _ = multiTransferDestinationShard.accounts.Commit()

	destination, err = multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	testNFTTokenShouldExist(t, multiTransferDestinationShard.marshalizer, destination, token1, tokenNonce, big.NewInt(1))
	testNFTTokenShouldExist(t, multiTransferDestinationShard.marshalizer, destination, token2, 0, big.NewInt(1))
	funcName, args = extractScResultsFromVmOutput(t, vmOutput)
	assert.Equal(t, scCallFunctionAsHex, funcName)
	require.Equal(t, 1, len(args))
	require.Equal(t, []byte(scCallArg), args[0])
}

func TestESDTNFTMultiTransfer_ProcessBuiltinFunctionOnCrossShardsDestinationAddToEsdtBalanceShouldErr(t *testing.T) {
	t.Parallel()

	payableHandler := &mock.PayableHandlerStub{
		IsPayableCalled: func(address []byte) (bool, error) {
			return true, nil
		},
	}

	multiTransferSenderShard := createESDTNFTMultiTransferWithMockArguments(1, 2, &mock.GlobalSettingsHandlerStub{})
	_ = multiTransferSenderShard.SetPayableHandler(payableHandler)

	multiTransferDestinationShard := createESDTNFTMultiTransferWithMockArguments(0, 2, &mock.GlobalSettingsHandlerStub{})
	_ = multiTransferDestinationShard.SetPayableHandler(payableHandler)

	senderAddress := bytes.Repeat([]byte{1}, 32)
	destinationAddress := bytes.Repeat([]byte{0}, 32)
	destinationAddress[25] = 1
	sender, err := multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	token1 := []byte("token1")
	token2 := []byte("token2")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, initialTokens, multiTransferSenderShard.marshalizer, sender.(vmcommon.UserAccountHandler))
	createESDTNFTToken(token2, core.Fungible, 0, initialTokens, multiTransferSenderShard.marshalizer, sender.(vmcommon.UserAccountHandler))
	_ = multiTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = multiTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	scCallFunctionAsHex := hex.EncodeToString([]byte("functionToCall"))
	scCallArg := hex.EncodeToString([]byte("arg"))
	scCallArgs := [][]byte{[]byte(scCallFunctionAsHex), []byte(scCallArg)}
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{destinationAddress, big.NewInt(2).Bytes(), token1, nonceBytes, quantityBytes, token2, big.NewInt(0).Bytes(), quantityBytes},
			GasProvided: 1000000,
		},
		RecipientAddr: senderAddress,
	}
	vmInput.Arguments = append(vmInput.Arguments, scCallArgs...)

	vmOutput, err := multiTransferSenderShard.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), nil, vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)

	_ = multiTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = multiTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	testNFTTokenShouldExist(t, multiTransferSenderShard.marshalizer, sender, token1, tokenNonce, big.NewInt(2)) // 3 initial - 1 transferred

	_, args := extractScResultsFromVmOutput(t, vmOutput)

	destination, err := multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: senderAddress,
			Arguments:  args,
		},
		RecipientAddr: destinationAddress,
	}

	multiTransferDestinationShard.globalSettingsHandler = &mock.GlobalSettingsHandlerStub{
		IsPausedCalled: func(tokenKey []byte) bool {
			esdtTokenKey := []byte(core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier)
			esdtTokenKey = append(esdtTokenKey, token2...)
			if bytes.Equal(tokenKey, esdtTokenKey) {
				return true
			}

			return false
		},
	}
	vmOutput, err = multiTransferDestinationShard.ProcessBuiltinFunction(nil, destination.(vmcommon.UserAccountHandler), vmInput)
	require.Error(t, err)
	require.Equal(t, "esdt token is paused for token token2", err.Error())
}

func TestESDTNFTMultiTransfer_ProcessBuiltinFunctionOnCrossShardsOneTransfer(t *testing.T) {
	t.Parallel()

	payableHandler := &mock.PayableHandlerStub{
		IsPayableCalled: func(address []byte) (bool, error) {
			return true, nil
		},
	}

	multiTransferSenderShard := createESDTNFTMultiTransferWithMockArguments(0, 2, &mock.GlobalSettingsHandlerStub{})
	_ = multiTransferSenderShard.SetPayableHandler(payableHandler)

	multiTransferDestinationShard := createESDTNFTMultiTransferWithMockArguments(1, 2, &mock.GlobalSettingsHandlerStub{})
	_ = multiTransferDestinationShard.SetPayableHandler(payableHandler)

	senderAddress := bytes.Repeat([]byte{2}, 32) // sender is in the same shard
	destinationAddress := bytes.Repeat([]byte{1}, 32)
	sender, err := multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	token1 := []byte("token1")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, initialTokens, multiTransferSenderShard.marshalizer, sender.(vmcommon.UserAccountHandler))
	_ = multiTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = multiTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{destinationAddress, big.NewInt(1).Bytes(), token1, nonceBytes, quantityBytes},
			GasProvided: 100000,
		},
		RecipientAddr: senderAddress,
	}

	vmOutput, err := multiTransferSenderShard.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), nil, vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)

	_ = multiTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = multiTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	testNFTTokenShouldExist(t, multiTransferSenderShard.marshalizer, sender, token1, tokenNonce, big.NewInt(2)) // 3 initial - 1 transferred
	_, args := extractScResultsFromVmOutput(t, vmOutput)

	destinationNumTokens1 := big.NewInt(1000)
	destination, err := multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, destinationNumTokens1, multiTransferDestinationShard.marshalizer, destination.(vmcommon.UserAccountHandler))
	_ = multiTransferDestinationShard.accounts.SaveAccount(destination)
	_, _ = multiTransferDestinationShard.accounts.Commit()

	destination, err = multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: senderAddress,
			Arguments:  args,
		},
		RecipientAddr: destinationAddress,
	}

	vmOutput, err = multiTransferDestinationShard.ProcessBuiltinFunction(nil, destination.(vmcommon.UserAccountHandler), vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)
	_ = multiTransferDestinationShard.accounts.SaveAccount(destination)
	_, _ = multiTransferDestinationShard.accounts.Commit()

	destination, err = multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	expectedTokens1 := big.NewInt(0).Add(destinationNumTokens1, big.NewInt(1))
	testNFTTokenShouldExist(t, multiTransferDestinationShard.marshalizer, destination, token1, tokenNonce, expectedTokens1)
}

func TestESDTNFTMultiTransfer_ProcessBuiltinFunctionOnCrossShardsDestinationHoldsNFT(t *testing.T) {
	t.Parallel()

	payableHandler := &mock.PayableHandlerStub{
		IsPayableCalled: func(address []byte) (bool, error) {
			return true, nil
		},
	}

	multiTransferSenderShard := createESDTNFTMultiTransferWithMockArguments(0, 2, &mock.GlobalSettingsHandlerStub{})
	_ = multiTransferSenderShard.SetPayableHandler(payableHandler)

	multiTransferDestinationShard := createESDTNFTMultiTransferWithMockArguments(1, 2, &mock.GlobalSettingsHandlerStub{})
	_ = multiTransferDestinationShard.SetPayableHandler(payableHandler)

	senderAddress := bytes.Repeat([]byte{2}, 32) // sender is in the same shard
	destinationAddress := bytes.Repeat([]byte{1}, 32)
	sender, err := multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	token1 := []byte("token1")
	token2 := []byte("token2")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, initialTokens, multiTransferSenderShard.marshalizer, sender.(vmcommon.UserAccountHandler))
	createESDTNFTToken(token2, core.Fungible, 0, initialTokens, multiTransferSenderShard.marshalizer, sender.(vmcommon.UserAccountHandler))
	_ = multiTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = multiTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{destinationAddress, big.NewInt(2).Bytes(), token1, nonceBytes, quantityBytes, token2, big.NewInt(0).Bytes(), quantityBytes},
			GasProvided: 100000,
		},
		RecipientAddr: senderAddress,
	}

	vmOutput, err := multiTransferSenderShard.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), nil, vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)

	_ = multiTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = multiTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	testNFTTokenShouldExist(t, multiTransferSenderShard.marshalizer, sender, token1, tokenNonce, big.NewInt(2)) // 3 initial - 1 transferred
	testNFTTokenShouldExist(t, multiTransferSenderShard.marshalizer, sender, token2, 0, big.NewInt(2))          // 3 initial - 1 transferred
	_, args := extractScResultsFromVmOutput(t, vmOutput)

	destinationNumTokens1 := big.NewInt(1000)
	destinationNumTokens2 := big.NewInt(1000)
	destination, err := multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, destinationNumTokens1, multiTransferDestinationShard.marshalizer, destination.(vmcommon.UserAccountHandler))
	createESDTNFTToken(token2, core.Fungible, 0, destinationNumTokens2, multiTransferDestinationShard.marshalizer, destination.(vmcommon.UserAccountHandler))
	_ = multiTransferDestinationShard.accounts.SaveAccount(destination)
	_, _ = multiTransferDestinationShard.accounts.Commit()

	destination, err = multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: senderAddress,
			Arguments:  args,
		},
		RecipientAddr: destinationAddress,
	}

	vmOutput, err = multiTransferDestinationShard.ProcessBuiltinFunction(nil, destination.(vmcommon.UserAccountHandler), vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)
	_ = multiTransferDestinationShard.accounts.SaveAccount(destination)
	_, _ = multiTransferDestinationShard.accounts.Commit()

	destination, err = multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	expectedTokens1 := big.NewInt(0).Add(destinationNumTokens1, big.NewInt(1))
	expectedTokens2 := big.NewInt(0).Add(destinationNumTokens2, big.NewInt(1))
	testNFTTokenShouldExist(t, multiTransferDestinationShard.marshalizer, destination, token1, tokenNonce, expectedTokens1)
	testNFTTokenShouldExist(t, multiTransferDestinationShard.marshalizer, destination, token2, 0, expectedTokens2)
}

func TestESDTNFTMultiTransfer_ProcessBuiltinFunctionOnCrossShardsShouldErr(t *testing.T) {
	t.Parallel()

	payableHandler := &mock.PayableHandlerStub{
		IsPayableCalled: func(address []byte) (bool, error) {
			return true, nil
		},
	}

	multiTransferSenderShard := createESDTNFTMultiTransferWithMockArguments(0, 2, &mock.GlobalSettingsHandlerStub{})
	_ = multiTransferSenderShard.SetPayableHandler(payableHandler)

	multiTransferDestinationShard := createESDTNFTMultiTransferWithMockArguments(1, 2, &mock.GlobalSettingsHandlerStub{})
	_ = multiTransferDestinationShard.SetPayableHandler(payableHandler)

	senderAddress := bytes.Repeat([]byte{2}, 32) // sender is in the same shard
	destinationAddress := bytes.Repeat([]byte{1}, 32)
	sender, err := multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	token1 := []byte("token1")
	token2 := []byte("token2")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, initialTokens, multiTransferSenderShard.marshalizer, sender.(vmcommon.UserAccountHandler))
	createESDTNFTToken(token2, core.Fungible, 0, initialTokens, multiTransferSenderShard.marshalizer, sender.(vmcommon.UserAccountHandler))
	_ = multiTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = multiTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{destinationAddress, big.NewInt(2).Bytes(), token1, nonceBytes, quantityBytes, token2, big.NewInt(0).Bytes(), quantityBytes},
			GasProvided: 100000,
		},
		RecipientAddr: senderAddress,
	}

	vmOutput, err := multiTransferSenderShard.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), nil, vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)

	_ = multiTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = multiTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	testNFTTokenShouldExist(t, multiTransferSenderShard.marshalizer, sender, token1, tokenNonce, big.NewInt(2)) // 3 initial - 1 transferred
	testNFTTokenShouldExist(t, multiTransferSenderShard.marshalizer, sender, token2, 0, big.NewInt(2))
	_, args := extractScResultsFromVmOutput(t, vmOutput)

	destinationNumTokens := big.NewInt(1000)
	destination, err := multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, destinationNumTokens, multiTransferDestinationShard.marshalizer, destination.(vmcommon.UserAccountHandler))
	_ = multiTransferDestinationShard.accounts.SaveAccount(destination)
	_, _ = multiTransferDestinationShard.accounts.Commit()

	destination, err = multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: senderAddress,
			Arguments:  args,
		},
		RecipientAddr: destinationAddress,
	}

	payableHandler = &mock.PayableHandlerStub{
		IsPayableCalled: func(address []byte) (bool, error) {
			return false, nil
		},
	}
	_ = multiTransferDestinationShard.SetPayableHandler(payableHandler)
	vmOutput, err = multiTransferDestinationShard.ProcessBuiltinFunction(nil, destination.(vmcommon.UserAccountHandler), vmInput)
	require.Error(t, err)
	require.Equal(t, "sending value to non payable contract", err.Error())

	// check the multi transfer for fungible ESDT transfers as well
	vmInput.Arguments = [][]byte{big.NewInt(2).Bytes(), token1, big.NewInt(0).Bytes(), quantityBytes, token2, big.NewInt(0).Bytes(), quantityBytes}
	vmOutput, err = multiTransferDestinationShard.ProcessBuiltinFunction(nil, destination.(vmcommon.UserAccountHandler), vmInput)
	require.Error(t, err)
	require.Equal(t, "sending value to non payable contract", err.Error())
}

func TestESDTNFTMultiTransfer_SndDstFrozen(t *testing.T) {
	t.Parallel()

	globalSettings := &mock.GlobalSettingsHandlerStub{}
	transferFunc := createESDTNFTMultiTransferWithMockArguments(0, 1, globalSettings)
	_ = transferFunc.SetPayableHandler(&mock.PayableHandlerStub{})

	senderAddress := bytes.Repeat([]byte{2}, 32) // sender is in the same shard
	destinationAddress := bytes.Repeat([]byte{1}, 32)
	destinationAddress[31] = 0
	sender, err := transferFunc.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	token1 := []byte("token1")
	token2 := []byte("token2")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, initialTokens, transferFunc.marshalizer, sender.(vmcommon.UserAccountHandler))
	createESDTNFTToken(token2, core.Fungible, 0, initialTokens, transferFunc.marshalizer, sender.(vmcommon.UserAccountHandler))
	esdtFrozen := ESDTUserMetadata{Frozen: true}

	_ = transferFunc.accounts.SaveAccount(sender)
	_, _ = transferFunc.accounts.Commit()
	// reload sender account
	sender, err = transferFunc.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{destinationAddress, big.NewInt(2).Bytes(), token1, nonceBytes, quantityBytes, token2, big.NewInt(0).Bytes(), quantityBytes},
			GasProvided: 100000,
		},
		RecipientAddr: senderAddress,
	}

	destination, _ := transferFunc.accounts.LoadAccount(destinationAddress)
	tokenId := append(keyPrefix, token1...)
	esdtKey := computeESDTNFTTokenKey(tokenId, tokenNonce)
	esdtToken := &esdt.ESDigitalToken{Value: big.NewInt(0), Properties: esdtFrozen.ToBytes()}
	marshaledData, _ := transferFunc.marshalizer.Marshal(esdtToken)
	_ = destination.(vmcommon.UserAccountHandler).AccountDataHandler().SaveKeyValue(esdtKey, marshaledData)
	_ = transferFunc.accounts.SaveAccount(destination)
	_, _ = transferFunc.accounts.Commit()

	_, err = transferFunc.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), destination.(vmcommon.UserAccountHandler), vmInput)
	assert.Error(t, err)
	assert.Equal(t, fmt.Sprintf("%s for token %s", ErrESDTIsFrozenForAccount, string(token1)), err.Error())

	globalSettings.IsLimiterTransferCalled = func(token []byte) bool {
		return true
	}
	_, err = transferFunc.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), destination.(vmcommon.UserAccountHandler), vmInput)
	assert.Error(t, err)
	assert.Equal(t, fmt.Sprintf("%s for token %s", ErrActionNotAllowed, string(token1)), err.Error())

	globalSettings.IsLimiterTransferCalled = func(token []byte) bool {
		return false
	}
	vmInput.ReturnCallAfterError = true
	_, err = transferFunc.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), destination.(vmcommon.UserAccountHandler), vmInput)
	assert.Nil(t, err)
}

func TestESDTNFTMultiTransfer_NotEnoughGas(t *testing.T) {
	t.Parallel()

	transferFunc := createESDTNFTMultiTransferWithMockArguments(0, 1, &mock.GlobalSettingsHandlerStub{})
	_ = transferFunc.SetPayableHandler(&mock.PayableHandlerStub{})

	senderAddress := bytes.Repeat([]byte{2}, 32) // sender is in the same shard
	destinationAddress := bytes.Repeat([]byte{1}, 32)
	sender, err := transferFunc.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	token1 := []byte("token1")
	token2 := []byte("token2")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, initialTokens, transferFunc.marshalizer, sender.(vmcommon.UserAccountHandler))
	createESDTNFTToken(token2, core.Fungible, 0, initialTokens, transferFunc.marshalizer, sender.(vmcommon.UserAccountHandler))
	_ = transferFunc.accounts.SaveAccount(sender)
	_, _ = transferFunc.accounts.Commit()
	// reload sender account
	sender, err = transferFunc.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{destinationAddress, big.NewInt(2).Bytes(), token1, nonceBytes, quantityBytes, token2, big.NewInt(0).Bytes(), quantityBytes},
			GasProvided: 1,
		},
		RecipientAddr: senderAddress,
	}

	_, err = transferFunc.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), sender.(vmcommon.UserAccountHandler), vmInput)
	assert.Equal(t, err, ErrNotEnoughGas)
}

func TestESDTNFTMultiTransfer_WithEgldValue(t *testing.T) {
	t.Parallel()

	transferFunc := createESDTNFTMultiTransferWithMockArguments(0, 1, &mock.GlobalSettingsHandlerStub{})
	_ = transferFunc.SetPayableHandler(&mock.PayableHandlerStub{})

	senderAddress := bytes.Repeat([]byte{2}, 32)
	destinationAddress := bytes.Repeat([]byte{1}, 32)
	sender, err := transferFunc.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	token1 := []byte("token1")
	token2 := []byte("token2")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createESDTNFTToken(token1, core.NonFungible, tokenNonce, initialTokens, transferFunc.marshalizer, sender.(vmcommon.UserAccountHandler))
	createESDTNFTToken(token2, core.Fungible, 0, initialTokens, transferFunc.marshalizer, sender.(vmcommon.UserAccountHandler))
	_ = transferFunc.accounts.SaveAccount(sender)
	_, _ = transferFunc.accounts.Commit()

	sender, err = transferFunc.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(1),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{destinationAddress, big.NewInt(2).Bytes(), token1, nonceBytes, quantityBytes, token2, big.NewInt(0).Bytes(), quantityBytes},
			GasProvided: 100000,
		},
		RecipientAddr: senderAddress,
	}

	output, err := transferFunc.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), sender.(vmcommon.UserAccountHandler), vmInput)
	require.Nil(t, output)
	require.Equal(t, ErrBuiltInFunctionCalledWithValue, err)
}

func TestComputeInsufficientQuantityESDTError(t *testing.T) {
	t.Parallel()

	resErr := computeInsufficientQuantityESDTError([]byte("my-token"), 0)
	require.NotNil(t, resErr)
	require.Equal(t, errors.New("insufficient quantity for token: my-token").Error(), resErr.Error())

	resErr = computeInsufficientQuantityESDTError([]byte("my-token-2"), 5)
	require.NotNil(t, resErr)
	require.Equal(t, errors.New("insufficient quantity for token: my-token-2 nonce 5").Error(), resErr.Error())
}

func TestESDTNFTMultiTransfer_EpochChange(t *testing.T) {
	t.Parallel()

	var functionHandler vmcommon.EpochSubscriberHandler
	notifier := &mock.EpochNotifierStub{
		RegisterNotifyHandlerCalled: func(handler vmcommon.EpochSubscriberHandler) {
			functionHandler = handler
		},
	}
	transferFunc, _ := NewESDTNFTMultiTransferFunc(
		0,
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.AccountsStub{},
		&mock.ShardCoordinatorStub{},
		vmcommon.BaseOperationCost{},
		0,
		notifier,
		&mock.ESDTRoleHandlerStub{},
		1,
		2,
		3,
		createNewESDTDataStorageHandler(),
	)

	functionHandler.EpochConfirmed(0, 0)
	assert.False(t, transferFunc.flagTransferToMeta.IsSet())
	assert.False(t, transferFunc.flagCheckCorrectTokenID.IsSet())
	assert.False(t, transferFunc.flagCheckFunctionArgument.IsSet())

	functionHandler.EpochConfirmed(1, 0)
	assert.True(t, transferFunc.flagTransferToMeta.IsSet())
	assert.False(t, transferFunc.flagCheckCorrectTokenID.IsSet())
	assert.False(t, transferFunc.flagCheckFunctionArgument.IsSet())

	functionHandler.EpochConfirmed(2, 0)
	assert.True(t, transferFunc.flagTransferToMeta.IsSet())
	assert.True(t, transferFunc.flagCheckCorrectTokenID.IsSet())
	assert.False(t, transferFunc.flagCheckFunctionArgument.IsSet())

	functionHandler.EpochConfirmed(3, 0)
	assert.True(t, transferFunc.flagTransferToMeta.IsSet())
	assert.True(t, transferFunc.flagCheckCorrectTokenID.IsSet())
	assert.True(t, transferFunc.flagCheckFunctionArgument.IsSet())

	functionHandler.EpochConfirmed(4, 0)
	assert.True(t, transferFunc.flagTransferToMeta.IsSet())
	assert.True(t, transferFunc.flagCheckCorrectTokenID.IsSet())
	assert.True(t, transferFunc.flagCheckFunctionArgument.IsSet())
}
