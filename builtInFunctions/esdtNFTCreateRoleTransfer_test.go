package builtInFunctions

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/data/esdt"
	"github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/mock"
	"github.com/stretchr/testify/assert"
)

func TestEsdtNFTCreateRoleTransfer_Constructor(t *testing.T) {
	t.Parallel()

	e, err := NewESDTNFTCreateRoleTransfer(nil, &mock.AccountsStub{}, mock.NewMultiShardsCoordinatorMock(2))
	assert.Nil(t, e)
	assert.Equal(t, err, ErrNilMarshalizer)

	e, err = NewESDTNFTCreateRoleTransfer(&mock.MarshalizerMock{}, nil, mock.NewMultiShardsCoordinatorMock(2))
	assert.Nil(t, e)
	assert.Equal(t, err, ErrNilAccountsAdapter)

	e, err = NewESDTNFTCreateRoleTransfer(&mock.MarshalizerMock{}, &mock.AccountsStub{}, nil)
	assert.Nil(t, e)
	assert.Equal(t, err, ErrNilShardCoordinator)

	e, err = NewESDTNFTCreateRoleTransfer(&mock.MarshalizerMock{}, &mock.AccountsStub{}, mock.NewMultiShardsCoordinatorMock(2))
	assert.Nil(t, err)
	assert.NotNil(t, e)
	assert.False(t, e.IsInterfaceNil())

	e.SetNewGasConfig(&vmcommon.GasCost{})
}

func TestESDTNFTCreateRoleTransfer_ProcessWithErrors(t *testing.T) {
	t.Parallel()

	e, err := NewESDTNFTCreateRoleTransfer(&mock.MarshalizerMock{}, &mock.AccountsStub{}, mock.NewMultiShardsCoordinatorMock(2))
	assert.Nil(t, err)
	assert.NotNil(t, e)

	vmOutput, err := e.ProcessBuiltinFunction(nil, nil, nil)
	assert.Equal(t, err, ErrNilVmInput)
	assert.Nil(t, vmOutput)

	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, &vmcommon.ContractCallInput{})
	assert.Equal(t, err, ErrNilValue)
	assert.Nil(t, vmOutput)

	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, &vmcommon.ContractCallInput{VMInput: vmcommon.VMInput{CallValue: big.NewInt(10)}})
	assert.Equal(t, err, ErrBuiltInFunctionCalledWithValue)
	assert.Nil(t, vmOutput)

	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, &vmcommon.ContractCallInput{VMInput: vmcommon.VMInput{CallValue: big.NewInt(0)}})
	assert.Equal(t, err, ErrInvalidArguments)
	assert.Nil(t, vmOutput)

	vmInput := &vmcommon.ContractCallInput{VMInput: vmcommon.VMInput{CallValue: big.NewInt(0)}}
	vmInput.Arguments = [][]byte{{1}, {2}}
	vmOutput, err = e.ProcessBuiltinFunction(&mock.UserAccountStub{}, nil, vmInput)
	assert.Equal(t, err, ErrInvalidArguments)
	assert.Nil(t, vmOutput)

	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Equal(t, err, ErrNilUserAccount)
	assert.Nil(t, vmOutput)

	vmInput.CallerAddr = vmcommon.ESDTSCAddress
	vmInput.Arguments = [][]byte{{1}, {2}, {3}}
	vmOutput, err = e.ProcessBuiltinFunction(nil, &mock.UserAccountStub{}, vmInput)
	assert.Equal(t, err, ErrInvalidArguments)
	assert.Nil(t, vmOutput)

	vmInput.Arguments = [][]byte{{1}, {2}}
	vmOutput, err = e.ProcessBuiltinFunction(nil, &mock.UserAccountStub{}, vmInput)
	assert.Equal(t, err, ErrInvalidArguments)
	assert.Nil(t, vmOutput)
}

func createESDTNFTCreateRoleTransferComponent(t *testing.T) *esdtNFTCreateRoleTransfer {
	marshalizer := &mock.MarshalizerMock{}
	shardCoordinator := mock.NewMultiShardsCoordinatorMock(2)
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
	}

	e, err := NewESDTNFTCreateRoleTransfer(marshalizer, accounts, shardCoordinator)
	assert.Nil(t, err)
	assert.NotNil(t, e)
	return e
}

func TestESDTNFTCreateRoleTransfer_ProcessAtCurrentShard(t *testing.T) {
	t.Parallel()

	e := createESDTNFTCreateRoleTransferComponent(t)

	tokenID := []byte("NFT")
	currentOwner := bytes.Repeat([]byte{1}, 32)
	destinationAddr := bytes.Repeat([]byte{2}, 32)
	vmInput := &vmcommon.ContractCallInput{}
	vmInput.CallValue = big.NewInt(0)
	vmInput.CallerAddr = vmcommon.ESDTSCAddress
	vmInput.Arguments = [][]byte{tokenID, destinationAddr}

	destAcc, _ := e.accounts.LoadAccount(currentOwner)
	userAcc := destAcc.(vmcommon.UserAccountHandler)

	esdtTokenRoleKey := append(roleKeyPrefix, tokenID...)
	err := saveRolesToAccount(userAcc, esdtTokenRoleKey, &esdt.ESDTRoles{Roles: [][]byte{[]byte(vmcommon.ESDTRoleNFTCreate), []byte(vmcommon.ESDTRoleNFTAddQuantity)}}, e.marshalizer)
	assert.Nil(t, err)
	_ = saveLatestNonce(userAcc, tokenID, 100)
	_ = e.accounts.SaveAccount(userAcc)
	_, _ = e.accounts.Commit()
	destAcc, _ = e.accounts.LoadAccount(currentOwner)
	userAcc = destAcc.(vmcommon.UserAccountHandler)

	vmOutput, err := e.ProcessBuiltinFunction(nil, userAcc, vmInput)
	assert.Nil(t, err)
	assert.Equal(t, len(vmOutput.OutputAccounts), 1)

	_ = e.accounts.SaveAccount(userAcc)
	_, _ = e.accounts.Commit()
	checkLatestNonce(t, e, currentOwner, tokenID, 0)
	checkNFTCreateRoleExists(t, e, currentOwner, tokenID, -1)

	checkLatestNonce(t, e, destinationAddr, tokenID, 100)
	checkNFTCreateRoleExists(t, e, destinationAddr, tokenID, 0)
}

func TestESDTNFTCreateRoleTransfer_ProcessCrossShard(t *testing.T) {
	t.Parallel()

	e := createESDTNFTCreateRoleTransferComponent(t)

	tokenID := []byte("NFT")
	currentOwner := bytes.Repeat([]byte{1}, 32)
	destinationAddr := bytes.Repeat([]byte{2}, 32)
	vmInput := &vmcommon.ContractCallInput{}
	vmInput.CallValue = big.NewInt(0)
	vmInput.CallerAddr = currentOwner
	nonce := uint64(100)
	vmInput.Arguments = [][]byte{tokenID, big.NewInt(0).SetUint64(nonce).Bytes()}

	destAcc, _ := e.accounts.LoadAccount(destinationAddr)
	userAcc := destAcc.(vmcommon.UserAccountHandler)
	vmOutput, err := e.ProcessBuiltinFunction(nil, userAcc, vmInput)
	assert.Nil(t, err)
	assert.Equal(t, len(vmOutput.OutputAccounts), 0)

	_ = e.accounts.SaveAccount(userAcc)
	_, _ = e.accounts.Commit()
	checkLatestNonce(t, e, destinationAddr, tokenID, 100)
	checkNFTCreateRoleExists(t, e, destinationAddr, tokenID, 0)

	destAcc, _ = e.accounts.LoadAccount(destinationAddr)
	userAcc = destAcc.(vmcommon.UserAccountHandler)
	vmOutput, err = e.ProcessBuiltinFunction(nil, userAcc, vmInput)
	assert.Nil(t, err)
	assert.Equal(t, len(vmOutput.OutputAccounts), 0)

	_ = e.accounts.SaveAccount(userAcc)
	_, _ = e.accounts.Commit()
	checkLatestNonce(t, e, destinationAddr, tokenID, 100)
	checkNFTCreateRoleExists(t, e, destinationAddr, tokenID, 0)

	vmInput.Arguments = append(vmInput.Arguments, []byte{100})
	vmOutput, err = e.ProcessBuiltinFunction(nil, userAcc, vmInput)
	assert.Equal(t, err, ErrInvalidArguments)
	assert.Nil(t, vmOutput)
}

func checkLatestNonce(t *testing.T, e *esdtNFTCreateRoleTransfer, addr []byte, tokenID []byte, expectedNonce uint64) {
	destAcc, _ := e.accounts.LoadAccount(addr)
	userAcc := destAcc.(vmcommon.UserAccountHandler)
	nonce, _ := getLatestNonce(userAcc, tokenID)
	assert.Equal(t, expectedNonce, nonce)
}

func checkNFTCreateRoleExists(t *testing.T, e *esdtNFTCreateRoleTransfer, addr []byte, tokenID []byte, expectedIndex int) {
	destAcc, _ := e.accounts.LoadAccount(addr)
	userAcc := destAcc.(vmcommon.UserAccountHandler)
	esdtTokenRoleKey := append(roleKeyPrefix, tokenID...)
	roles, _, _ := getESDTRolesForAcnt(e.marshalizer, userAcc, esdtTokenRoleKey)
	assert.Equal(t, 1, len(roles.Roles))
	index, _ := doesRoleExist(roles, []byte(vmcommon.ESDTRoleNFTCreate))
	assert.Equal(t, expectedIndex, index)
}
