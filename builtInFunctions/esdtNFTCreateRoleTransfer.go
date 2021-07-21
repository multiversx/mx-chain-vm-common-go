package builtInFunctions

import (
	"bytes"
	"encoding/hex"
	"math/big"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data/esdt"
	"github.com/ElrondNetwork/elrond-vm-common"
)

type esdtNFTCreateRoleTransfer struct {
	baseAlwaysActive
	keyPrefix        []byte
	marshalizer      vmcommon.Marshalizer
	accounts         vmcommon.AccountsAdapter
	shardCoordinator vmcommon.Coordinator
}

// NewESDTNFTCreateRoleTransfer returns the esdt NFT create role transfer built-in function component
func NewESDTNFTCreateRoleTransfer(
	marshalizer vmcommon.Marshalizer,
	accounts vmcommon.AccountsAdapter,
	shardCoordinator vmcommon.Coordinator,
) (*esdtNFTCreateRoleTransfer, error) {
	if check.IfNil(marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if check.IfNil(shardCoordinator) {
		return nil, ErrNilShardCoordinator
	}

	e := &esdtNFTCreateRoleTransfer{
		keyPrefix:        []byte(vmcommon.ElrondProtectedKeyPrefix + vmcommon.ESDTKeyIdentifier),
		marshalizer:      marshalizer,
		accounts:         accounts,
		shardCoordinator: shardCoordinator,
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtNFTCreateRoleTransfer) SetNewGasConfig(_ *vmcommon.GasCost) {
}

// ProcessBuiltinFunction resolves ESDT create role transfer function call
func (e *esdtNFTCreateRoleTransfer) ProcessBuiltinFunction(
	acntSnd, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {

	err := checkBasicESDTArguments(vmInput)
	if err != nil {
		return nil, err
	}
	if !check.IfNil(acntSnd) {
		return nil, ErrInvalidArguments
	}
	if check.IfNil(acntDst) {
		return nil, ErrNilUserAccount
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}
	if bytes.Equal(vmInput.CallerAddr, vmcommon.ESDTSCAddress) {
		outAcc, errExec := e.executeTransferNFTCreateChangeAtCurrentOwner(acntDst, vmInput)
		if errExec != nil {
			return nil, errExec
		}
		vmOutput.OutputAccounts = make(map[string]*vmcommon.OutputAccount)
		vmOutput.OutputAccounts[string(outAcc.Address)] = outAcc
	} else {
		err = e.executeTransferNFTCreateChangeAtNextOwner(acntDst, vmInput)
		if err != nil {
			return nil, err
		}
	}

	return vmOutput, nil
}

func (e *esdtNFTCreateRoleTransfer) executeTransferNFTCreateChangeAtCurrentOwner(
	acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.OutputAccount, error) {
	if len(vmInput.Arguments) != 2 {
		return nil, ErrInvalidArguments
	}
	if len(vmInput.Arguments[1]) != len(vmInput.CallerAddr) {
		return nil, ErrInvalidArguments
	}

	tokenID := vmInput.Arguments[0]
	nonce, err := getLatestNonce(acntDst, tokenID)
	if err != nil {
		return nil, err
	}

	err = saveLatestNonce(acntDst, tokenID, 0)
	if err != nil {
		return nil, err
	}

	esdtTokenRoleKey := append(roleKeyPrefix, tokenID...)
	err = e.deleteCreateRoleFromAccount(acntDst, esdtTokenRoleKey)
	if err != nil {
		return nil, err
	}

	destAddress := vmInput.Arguments[1]
	if e.shardCoordinator.ComputeId(destAddress) == e.shardCoordinator.SelfId() {
		newDestinationAcc, errLoad := e.accounts.LoadAccount(destAddress)
		if errLoad != nil {
			return nil, errLoad
		}
		newDestUserAcc, ok := newDestinationAcc.(vmcommon.UserAccountHandler)
		if !ok {
			return nil, ErrWrongTypeAssertion
		}

		err = saveLatestNonce(newDestUserAcc, tokenID, nonce)
		if err != nil {
			return nil, err
		}

		err = e.addCreateRoleToAccount(newDestUserAcc, esdtTokenRoleKey)
		if err != nil {
			return nil, err
		}

		err = e.accounts.SaveAccount(newDestUserAcc)
		if err != nil {
			return nil, err
		}
	}

	outAcc := &vmcommon.OutputAccount{
		Address:         destAddress,
		Balance:         big.NewInt(0),
		BalanceDelta:    big.NewInt(0),
		OutputTransfers: make([]vmcommon.OutputTransfer, 0),
	}
	outTransfer := vmcommon.OutputTransfer{
		Value: big.NewInt(0),
		Data: []byte(vmcommon.BuiltInFunctionESDTNFTCreateRoleTransfer + "@" +
			hex.EncodeToString(tokenID) + "@" + hex.EncodeToString(big.NewInt(0).SetUint64(nonce).Bytes())),
		SenderAddress: vmInput.CallerAddr,
	}
	outAcc.OutputTransfers = append(outAcc.OutputTransfers, outTransfer)

	return outAcc, nil
}

func (e *esdtNFTCreateRoleTransfer) deleteCreateRoleFromAccount(
	acntDst vmcommon.UserAccountHandler,
	esdtTokenRoleKey []byte,
) error {
	roles, _, err := getESDTRolesForAcnt(e.marshalizer, acntDst, esdtTokenRoleKey)
	if err != nil {
		return err
	}

	deleteRoles(roles, [][]byte{[]byte(vmcommon.ESDTRoleNFTCreate)})
	return saveRolesToAccount(acntDst, esdtTokenRoleKey, roles, e.marshalizer)
}

func (e *esdtNFTCreateRoleTransfer) addCreateRoleToAccount(
	acntDst vmcommon.UserAccountHandler,
	esdtTokenRoleKey []byte,
) error {
	roles, _, err := getESDTRolesForAcnt(e.marshalizer, acntDst, esdtTokenRoleKey)
	if err != nil {
		return err
	}

	for _, role := range roles.Roles {
		if bytes.Equal(role, []byte(vmcommon.ESDTRoleNFTCreate)) {
			return nil
		}
	}

	roles.Roles = append(roles.Roles, []byte(vmcommon.ESDTRoleNFTCreate))
	return saveRolesToAccount(acntDst, esdtTokenRoleKey, roles, e.marshalizer)
}

func saveRolesToAccount(
	acntDst vmcommon.UserAccountHandler,
	esdtTokenRoleKey []byte,
	roles *esdt.ESDTRoles,
	marshalizer vmcommon.Marshalizer,
) error {
	marshaledData, err := marshalizer.Marshal(roles)
	if err != nil {
		return err
	}
	err = acntDst.AccountDataHandler().SaveKeyValue(esdtTokenRoleKey, marshaledData)
	if err != nil {
		return err
	}

	return nil
}

func (e *esdtNFTCreateRoleTransfer) executeTransferNFTCreateChangeAtNextOwner(
	acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) error {
	if len(vmInput.Arguments) != 2 {
		return ErrInvalidArguments
	}

	tokenID := vmInput.Arguments[0]
	nonce := big.NewInt(0).SetBytes(vmInput.Arguments[1]).Uint64()

	err := saveLatestNonce(acntDst, tokenID, nonce)
	if err != nil {
		return err
	}

	esdtTokenRoleKey := append(roleKeyPrefix, tokenID...)
	err = e.addCreateRoleToAccount(acntDst, esdtTokenRoleKey)
	if err != nil {
		return err
	}

	return nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtNFTCreateRoleTransfer) IsInterfaceNil() bool {
	return e == nil
}
