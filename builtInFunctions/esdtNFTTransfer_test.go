package builtInFunctions

import (
	"bytes"
	"encoding/hex"
	"io/ioutil"
	"math/big"
	"strings"
	"testing"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/pubkeyConverter"
	"github.com/ElrondNetwork/elrond-go/core/vmcommon"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/esdt"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/state/factory"
	"github.com/ElrondNetwork/elrond-go/data/trie"
	"github.com/ElrondNetwork/elrond-go/data/trie/evictionWaitingList"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/mock"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/memorydb"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
	"github.com/ElrondNetwork/elrond-go/vm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var keyPrefix = []byte(vmcommon.ElrondProtectedKeyPrefix + vmcommon.ESDTKeyIdentifier)

func createNftTransferWithStubArguments() *esdtNFTTransfer {
	nftTransfer, _ := NewESDTNFTTransferFunc(
		0,
		&mock.MarshalizerStub{},
		&mock.PauseHandlerStub{},
		&mock.AccountsStub{},
		&mock.ShardCoordinatorStub{},
		vmcommon.BaseOperationCost{},
	)

	return nftTransfer
}

func createMemUnit() storage.Storer {
	capacity := uint32(10)
	shards := uint32(1)
	sizeInBytes := uint64(0)
	cache, _ := storageUnit.NewCache(storageUnit.CacheConfig{Type: storageUnit.LRUCache, Capacity: capacity, Shards: shards, SizeInBytes: sizeInBytes})
	persist, _ := memorydb.NewlruDB(100000)
	unit, _ := storageUnit.NewStorageUnit(cache, persist)

	return unit
}

func createTrieStorageManager(store storage.Storer, marshalizer vmcommon.Marshalizer, hasher hashing.Hasher) data.StorageManager {
	ewl, _ := evictionWaitingList.NewEvictionWaitingList(100, memorydb.New(), marshalizer)
	tempDir, _ := ioutil.TempDir("", "process")
	cfg := config.DBConfig{
		FilePath:          tempDir,
		Type:              string(storageUnit.LvlDBSerial),
		BatchDelaySeconds: 4,
		MaxBatchSize:      10000,
		MaxOpenFiles:      10,
	}
	generalCfg := config.TrieStorageManagerConfig{
		PruningBufferLen:   1000,
		SnapshotsBufferLen: 10,
		MaxSnapshots:       2,
	}
	trieStorageManager, _ := trie.NewTrieStorageManager(store, marshalizer, hasher, cfg, ewl, generalCfg)

	return trieStorageManager
}

func createNftTransferWithMockArguments(shardID uint32, numShards uint32, pauseHandler vmcommon.ESDTPauseHandler) *esdtNFTTransfer {
	marshalizer := &mock.MarshalizerMock{}
	hasher := &mock.HasherMock{}
	shardCoordinator, _ := vmcommon.NewMultiShardCoordinator(numShards, shardID)
	trieStoreManager := createTrieStorageManager(createMemUnit(), marshalizer, hasher)
	tr, _ := trie.NewTrie(trieStoreManager, marshalizer, hasher, 6)
	accounts, _ := vmcommon.NewAccountsDB(tr, hasher, marshalizer, factory.NewAccountCreator())

	nftTransfer, _ := NewESDTNFTTransferFunc(
		1,
		marshalizer,
		pauseHandler,
		accounts,
		shardCoordinator,
		vmcommon.BaseOperationCost{},
	)

	return nftTransfer
}

func createMockGasCost() vmcommon.GasCost {
	return vmcommon.GasCost{
		BaseOperationCost: vmcommon.BaseOperationCost{
			StorePerByte:      10,
			ReleasePerByte:    20,
			DataCopyPerByte:   30,
			PersistPerByte:    40,
			CompilePerByte:    50,
			AoTPreparePerByte: 60,
		},
		BuiltInCost: vmcommon.BuiltInCost{
			ChangeOwnerAddress:       70,
			ClaimDeveloperRewards:    80,
			SaveUserName:             90,
			SaveKeyValue:             100,
			ESDTTransfer:             110,
			ESDTBurn:                 120,
			ESDTLocalMint:            130,
			ESDTLocalBurn:            140,
			ESDTNFTCreate:            150,
			ESDTNFTAddQuantity:       160,
			ESDTNFTBurn:              170,
			ESDTNFTTransfer:          180,
			ESDTNFTChangeCreateOwner: 190,
		},
	}
}

func createESDTToken(
	tokenName []byte,
	nftType vmcommon.ESDTType,
	nonce uint64,
	value *big.Int,
	marshalizer vmcommon.Marshalizer,
	account vmcommon.UserAccountHandler,
) {
	tokenId := append(keyPrefix, tokenName...)
	esdtNFTTokenKey := computeESDTNFTTokenKey(tokenId, nonce)
	esdtData := &esdt.ESDigitalToken{
		Type:  uint32(nftType),
		Value: value,
		TokenMetaData: &esdt.MetaData{
			URIs:  [][]byte{[]byte("uri")},
			Nonce: nonce,
			Hash:  []byte("NFT hash"),
		},
	}
	buff, _ := marshalizer.Marshal(esdtData)

	_ = account.DataTrieTracker().SaveKeyValue(esdtNFTTokenKey, buff)
}

func testNFTTokenShouldExist(
	tb testing.TB,
	marshalizer vmcommon.Marshalizer,
	account vmcommon.AccountHandler,
	tokenName []byte,
	nonce uint64,
	expectedValue *big.Int,
) {
	tokenId := append(keyPrefix, tokenName...)
	esdtData, err := getESDTNFTTokenOnSender(account.(vmcommon.UserAccountHandler), tokenId, nonce, marshalizer)
	require.Nil(tb, err)
	assert.Equal(tb, expectedValue, esdtData.Value)
}

func TestNewESDTNFTTransferFunc_NilArgumentsShouldErr(t *testing.T) {
	t.Parallel()

	nftTransfer, err := NewESDTNFTTransferFunc(
		0,
		nil,
		&mock.PauseHandlerStub{},
		&mock.AccountsStub{},
		&mock.ShardCoordinatorStub{},
		vmcommon.BaseOperationCost{},
	)
	assert.True(t, check.IfNil(nftTransfer))
	assert.Equal(t, ErrNilMarshalizer, err)

	nftTransfer, err = NewESDTNFTTransferFunc(
		0,
		&mock.MarshalizerStub{},
		nil,
		&mock.AccountsStub{},
		&mock.ShardCoordinatorStub{},
		vmcommon.BaseOperationCost{},
	)
	assert.True(t, check.IfNil(nftTransfer))
	assert.Equal(t, ErrNilPauseHandler, err)

	nftTransfer, err = NewESDTNFTTransferFunc(
		0,
		&mock.MarshalizerStub{},
		&mock.PauseHandlerStub{},
		nil,
		&mock.ShardCoordinatorStub{},
		vmcommon.BaseOperationCost{},
	)
	assert.True(t, check.IfNil(nftTransfer))
	assert.Equal(t, ErrNilAccountsAdapter, err)

	nftTransfer, err = NewESDTNFTTransferFunc(
		0,
		&mock.MarshalizerStub{},
		&mock.PauseHandlerStub{},
		&mock.AccountsStub{},
		nil,
		vmcommon.BaseOperationCost{},
	)
	assert.True(t, check.IfNil(nftTransfer))
	assert.Equal(t, ErrNilShardCoordinator, err)
}

func TestNewESDTNFTTransferFunc(t *testing.T) {
	t.Parallel()

	nftTransfer, err := NewESDTNFTTransferFunc(
		0,
		&mock.MarshalizerStub{},
		&mock.PauseHandlerStub{},
		&mock.AccountsStub{},
		&mock.ShardCoordinatorStub{},
		vmcommon.BaseOperationCost{},
	)
	assert.False(t, check.IfNil(nftTransfer))
	assert.Nil(t, err)
}

func TestEsdtNFTTransfer_SetPayable(t *testing.T) {
	t.Parallel()

	nftTransfer := createNftTransferWithStubArguments()
	err := nftTransfer.setPayableHandler(nil)
	assert.Equal(t, ErrNilPayableHandler, err)

	handler := &mock.PayableHandlerStub{}
	err = nftTransfer.setPayableHandler(handler)
	assert.Nil(t, err)
	assert.True(t, handler == nftTransfer.payableHandler) //pointer testing
}

func TestEsdtNFTTransfer_SetNewGasConfig(t *testing.T) {
	t.Parallel()

	nftTransfer := createNftTransferWithStubArguments()
	nftTransfer.SetNewGasConfig(nil)
	assert.Equal(t, uint64(0), nftTransfer.funcGasCost)
	assert.Equal(t, vmcommon.BaseOperationCost{}, nftTransfer.gasConfig)

	gasCost := createMockGasCost()
	nftTransfer.SetNewGasConfig(&gasCost)
	assert.Equal(t, gasCost.BuiltInCost.ESDTNFTTransfer, nftTransfer.funcGasCost)
	assert.Equal(t, gasCost.BaseOperationCost, nftTransfer.gasConfig)
}

func TestEsdtNFTTransfer_ProcessBuiltinFunctionInvalidArgumentsShouldErr(t *testing.T) {
	t.Parallel()

	nftTransfer := createNftTransferWithStubArguments()
	vmOutput, err := nftTransfer.ProcessBuiltinFunction(&mock.UserAccountStub{}, &mock.UserAccountStub{}, nil)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrNilVmInput, err)

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
			Arguments: [][]byte{[]byte("arg1"), []byte("arg2")},
		},
	}
	vmOutput, err = nftTransfer.ProcessBuiltinFunction(&mock.UserAccountStub{}, &mock.UserAccountStub{}, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrInvalidArguments, err)

	nftTransfer.shardCoordinator = &mock.ShardCoordinatorStub{ComputeIdCalled: func(address []byte) uint32 {
		return vmcommon.MetachainShardId
	}}

	tokenName := []byte("token")
	senderAddress := bytes.Repeat([]byte{2}, 32)
	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{tokenName, big.NewInt(1).Bytes(), big.NewInt(1).Bytes(), vmcommon.ESDTSCAddress},
			GasProvided: 1,
		},
		RecipientAddr: senderAddress,
	}
	vmOutput, err = nftTransfer.ProcessBuiltinFunction(&mock.UserAccountStub{}, &mock.UserAccountStub{}, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrInvalidRcvAddr, err)
}

func TestEsdtNFTTransfer_ProcessBuiltinFunctionOnSameShardWithScCall(t *testing.T) {
	t.Parallel()

	nftTransfer := createNftTransferWithMockArguments(0, 1, &mock.PauseHandlerStub{})
	_ = nftTransfer.setPayableHandler(
		&mock.PayableHandlerStub{
			IsPayableCalled: func(address []byte) (bool, error) {
				return true, nil
			},
		})
	senderAddress := bytes.Repeat([]byte{2}, 32)
	pkConv, _ := pubkeyConverter.NewBech32PubkeyConverter(32)
	destinationAddress, _ := pkConv.Decode("erd1qqqqqqqqqqqqqpgqrchxzx5uu8sv3ceg8nx8cxc0gesezure5awqn46gtd")
	sender, err := nftTransfer.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)
	destination, err := nftTransfer.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	tokenName := []byte("token")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createESDTToken(tokenName, vmcommon.NonFungible, tokenNonce, initialTokens, nftTransfer.marshalizer, sender.(vmcommon.UserAccountHandler))
	_ = nftTransfer.accounts.SaveAccount(sender)
	_ = nftTransfer.accounts.SaveAccount(destination)
	_, _ = nftTransfer.accounts.Commit()

	//reload accounts
	sender, err = nftTransfer.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)
	destination, err = nftTransfer.accounts.LoadAccount(destinationAddress)
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
			Arguments:   [][]byte{tokenName, nonceBytes, quantityBytes, destinationAddress},
			GasProvided: 1,
		},
		RecipientAddr: senderAddress,
	}
	vmInput.Arguments = append(vmInput.Arguments, scCallArgs...)

	vmOutput, err := nftTransfer.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), destination.(vmcommon.UserAccountHandler), vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)

	_ = nftTransfer.accounts.SaveAccount(sender)
	_, _ = nftTransfer.accounts.Commit()

	//reload accounts
	sender, err = nftTransfer.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)
	destination, err = nftTransfer.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	testNFTTokenShouldExist(t, nftTransfer.marshalizer, sender, tokenName, tokenNonce, big.NewInt(2)) //3 initial - 1 transferred
	testNFTTokenShouldExist(t, nftTransfer.marshalizer, destination, tokenName, tokenNonce, big.NewInt(1))
	funcName, args := extractScResultsFromVmOutput(t, vmOutput)
	assert.Equal(t, scCallFunctionAsHex, funcName)
	require.Equal(t, 1, len(args))
	require.Equal(t, []byte(scCallArg), args[0])
}

func TestEsdtNFTTransfer_ProcessBuiltinFunctionOnCrossShardsDestinationDoesNotHoldingNFTWithSCCall(t *testing.T) {
	t.Parallel()

	payableHandler := &mock.PayableHandlerStub{
		IsPayableCalled: func(address []byte) (bool, error) {
			return true, nil
		},
	}

	nftTransferSenderShard := createNftTransferWithMockArguments(1, 2, &mock.PauseHandlerStub{})
	_ = nftTransferSenderShard.setPayableHandler(payableHandler)

	nftTransferDestinationShard := createNftTransferWithMockArguments(0, 2, &mock.PauseHandlerStub{})
	_ = nftTransferDestinationShard.setPayableHandler(payableHandler)

	senderAddress := bytes.Repeat([]byte{1}, 32)
	pkConv, _ := pubkeyConverter.NewBech32PubkeyConverter(32)
	destinationAddress, _ := pkConv.Decode("erd1qqqqqqqqqqqqqpgqrchxzx5uu8sv3ceg8nx8cxc0gesezure5awqn46gtd")
	sender, err := nftTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	tokenName := []byte("token")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createESDTToken(tokenName, vmcommon.NonFungible, tokenNonce, initialTokens, nftTransferSenderShard.marshalizer, sender.(vmcommon.UserAccountHandler))
	_ = nftTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = nftTransferSenderShard.accounts.Commit()

	//reload sender account
	sender, err = nftTransferSenderShard.accounts.LoadAccount(senderAddress)
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
			Arguments:   [][]byte{tokenName, nonceBytes, quantityBytes, destinationAddress},
			GasProvided: 1,
		},
		RecipientAddr: senderAddress,
	}
	vmInput.Arguments = append(vmInput.Arguments, scCallArgs...)

	vmOutput, err := nftTransferSenderShard.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), nil, vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)

	_ = nftTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = nftTransferSenderShard.accounts.Commit()

	//reload sender account
	sender, err = nftTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	testNFTTokenShouldExist(t, nftTransferSenderShard.marshalizer, sender, tokenName, tokenNonce, big.NewInt(2)) //3 initial - 1 transferred

	funcName, args := extractScResultsFromVmOutput(t, vmOutput)
	log.Info("executing on destination shard", "function", funcName, "args", args)

	destination, err := nftTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: senderAddress,
			Arguments:  args,
		},
		RecipientAddr: destinationAddress,
	}

	vmOutput, err = nftTransferDestinationShard.ProcessBuiltinFunction(nil, destination.(vmcommon.UserAccountHandler), vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)
	_ = nftTransferDestinationShard.accounts.SaveAccount(destination)
	_, _ = nftTransferDestinationShard.accounts.Commit()

	destination, err = nftTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	testNFTTokenShouldExist(t, nftTransferDestinationShard.marshalizer, destination, tokenName, tokenNonce, big.NewInt(1))
	funcName, args = extractScResultsFromVmOutput(t, vmOutput)
	assert.Equal(t, scCallFunctionAsHex, funcName)
	require.Equal(t, 1, len(args))
	require.Equal(t, []byte(scCallArg), args[0])
}

func TestEsdtNFTTransfer_ProcessBuiltinFunctionOnCrossShardsDestinationHoldsNFT(t *testing.T) {
	t.Parallel()

	payableHandler := &mock.PayableHandlerStub{
		IsPayableCalled: func(address []byte) (bool, error) {
			return true, nil
		},
	}

	nftTransferSenderShard := createNftTransferWithMockArguments(0, 2, &mock.PauseHandlerStub{})
	_ = nftTransferSenderShard.setPayableHandler(payableHandler)

	nftTransferDestinationShard := createNftTransferWithMockArguments(1, 2, &mock.PauseHandlerStub{})
	_ = nftTransferDestinationShard.setPayableHandler(payableHandler)

	senderAddress := bytes.Repeat([]byte{2}, 32) // sender is in the same shard
	destinationAddress := bytes.Repeat([]byte{1}, 32)
	sender, err := nftTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	tokenName := []byte("token")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createESDTToken(tokenName, vmcommon.NonFungible, tokenNonce, initialTokens, nftTransferSenderShard.marshalizer, sender.(vmcommon.UserAccountHandler))
	_ = nftTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = nftTransferSenderShard.accounts.Commit()

	//reload sender account
	sender, err = nftTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{tokenName, nonceBytes, quantityBytes, destinationAddress},
			GasProvided: 1,
		},
		RecipientAddr: senderAddress,
	}

	vmOutput, err := nftTransferSenderShard.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), nil, vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)

	_ = nftTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = nftTransferSenderShard.accounts.Commit()

	//reload sender account
	sender, err = nftTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	testNFTTokenShouldExist(t, nftTransferSenderShard.marshalizer, sender, tokenName, tokenNonce, big.NewInt(2)) //3 initial - 1 transferred

	funcName, args := extractScResultsFromVmOutput(t, vmOutput)
	log.Info("executing on destination shard", "function", funcName, "args", args)

	destinationNumTokens := big.NewInt(1000)
	destination, err := nftTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)
	createESDTToken(tokenName, vmcommon.NonFungible, tokenNonce, destinationNumTokens, nftTransferDestinationShard.marshalizer, destination.(vmcommon.UserAccountHandler))
	_ = nftTransferDestinationShard.accounts.SaveAccount(destination)
	_, _ = nftTransferDestinationShard.accounts.Commit()

	destination, err = nftTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: senderAddress,
			Arguments:  args,
		},
		RecipientAddr: destinationAddress,
	}

	vmOutput, err = nftTransferDestinationShard.ProcessBuiltinFunction(nil, destination.(vmcommon.UserAccountHandler), vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)
	_ = nftTransferDestinationShard.accounts.SaveAccount(destination)
	_, _ = nftTransferDestinationShard.accounts.Commit()

	destination, err = nftTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	expected := big.NewInt(0).Add(destinationNumTokens, big.NewInt(1))
	testNFTTokenShouldExist(t, nftTransferDestinationShard.marshalizer, destination, tokenName, tokenNonce, expected)
}

func TestESDTNFTTransfer_SndDstFrozen(t *testing.T) {
	t.Parallel()

	transferFunc := createNftTransferWithMockArguments(0, 1, &mock.PauseHandlerStub{})
	_ = transferFunc.setPayableHandler(&mock.PayableHandlerStub{})

	senderAddress := bytes.Repeat([]byte{2}, 32) // sender is in the same shard
	destinationAddress := bytes.Repeat([]byte{1}, 32)
	sender, err := transferFunc.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	tokenName := []byte("token")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createESDTToken(tokenName, vmcommon.NonFungible, tokenNonce, initialTokens, transferFunc.marshalizer, sender.(vmcommon.UserAccountHandler))
	esdtFrozen := ESDTUserMetadata{Frozen: true}

	_ = transferFunc.accounts.SaveAccount(sender)
	_, _ = transferFunc.accounts.Commit()
	//reload sender account
	sender, err = transferFunc.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{tokenName, nonceBytes, quantityBytes, destinationAddress},
			GasProvided: 1,
		},
		RecipientAddr: senderAddress,
	}

	destination, _ := transferFunc.accounts.LoadAccount(destinationAddress)
	tokenId := append(keyPrefix, tokenName...)
	esdtKey := computeESDTNFTTokenKey(tokenId, tokenNonce)
	esdtToken := &esdt.ESDigitalToken{Value: big.NewInt(0), Properties: esdtFrozen.ToBytes()}
	marshaledData, _ := transferFunc.marshalizer.Marshal(esdtToken)
	_ = destination.(vmcommon.UserAccountHandler).DataTrieTracker().SaveKeyValue(esdtKey, marshaledData)
	_ = transferFunc.accounts.SaveAccount(destination)
	_, _ = transferFunc.accounts.Commit()

	_, err = transferFunc.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), sender.(vmcommon.UserAccountHandler), vmInput)
	assert.Equal(t, err, ErrESDTIsFrozenForAccount)
}

func TestESDTNFTTransfer_NotEnoughGas(t *testing.T) {
	t.Parallel()

	transferFunc := createNftTransferWithMockArguments(0, 1, &mock.PauseHandlerStub{})
	_ = transferFunc.setPayableHandler(&mock.PayableHandlerStub{})

	senderAddress := bytes.Repeat([]byte{2}, 32) // sender is in the same shard
	destinationAddress := bytes.Repeat([]byte{1}, 32)
	sender, err := transferFunc.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	tokenName := []byte("token")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createESDTToken(tokenName, vmcommon.NonFungible, tokenNonce, initialTokens, transferFunc.marshalizer, sender.(vmcommon.UserAccountHandler))
	_ = transferFunc.accounts.SaveAccount(sender)
	_, _ = transferFunc.accounts.Commit()
	//reload sender account
	sender, err = transferFunc.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{tokenName, nonceBytes, quantityBytes, destinationAddress},
			GasProvided: 0,
		},
		RecipientAddr: senderAddress,
	}

	_, err = transferFunc.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), sender.(vmcommon.UserAccountHandler), vmInput)
	assert.Equal(t, err, ErrNotEnoughGas)
}

func extractScResultsFromVmOutput(t testing.TB, vmOutput *vmcommon.VMOutput) (string, [][]byte) {
	require.NotNil(t, vmOutput)
	require.Equal(t, 1, len(vmOutput.OutputAccounts))
	var outputAccount *vmcommon.OutputAccount
	for _, account := range vmOutput.OutputAccounts {
		outputAccount = account
		break
	}
	require.NotNil(t, outputAccount)
	if outputAccount == nil {
		//supress next warnings, goland does not know about require.NotNil
		return "", nil
	}
	require.Equal(t, 1, len(outputAccount.OutputTransfers))
	outputTransfer := outputAccount.OutputTransfers[0]
	split := strings.Split(string(outputTransfer.Data), "@")

	args := make([][]byte, len(split)-1)
	var err error
	for i, splitArg := range split[1:] {
		args[i], err = hex.DecodeString(splitArg)
		require.Nil(t, err)
	}

	return split[0], args
}
