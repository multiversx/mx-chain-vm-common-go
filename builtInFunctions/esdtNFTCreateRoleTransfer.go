package builtInFunctions

import (
	"bytes"
	"encoding/hex"
	"math/big"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data/esdt"
	"github.com/multiversx/mx-chain-vm-common-go"
)

type esdtNFTCreateRoleTransfer struct {
	baseAlwaysActiveHandler
	keyPrefix        []byte
	marshaller       vmcommon.Marshalizer
	accounts         vmcommon.AccountsAdapter
	shardCoordinator vmcommon.Coordinator
}

// NewESDTNFTCreateRoleTransfer returns the esdt NFT create role transfer built-in function component
func NewESDTNFTCreateRoleTransfer(
	marshaller vmcommon.Marshalizer,
	accounts vmcommon.AccountsAdapter,
	shardCoordinator vmcommon.Coordinator,
) (*esdtNFTCreateRoleTransfer, error) {
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if check.IfNil(shardCoordinator) {
		return nil, ErrNilShardCoordinator
	}

	e := &esdtNFTCreateRoleTransfer{
		keyPrefix:        []byte(baseESDTKeyPrefix),
		marshaller:       marshaller,
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
	if bytes.Equal(vmInput.CallerAddr, core.ESDTSCAddress) {
		outAcc, errExec := e.executeTransferNFTCreateChangeAtCurrentOwner(vmOutput, acntDst, vmInput)
		if errExec != nil {
			return nil, errExec
		}
		vmOutput.OutputAccounts = make(map[string]*vmcommon.OutputAccount)
		vmOutput.OutputAccounts[string(outAcc.Address)] = outAcc
	} else {
		err = e.executeTransferNFTCreateChangeAtNextOwner(vmOutput, acntDst, vmInput)
		if err != nil {
			return nil, err
		}
	}

	return vmOutput, nil
}

func (e *esdtNFTCreateRoleTransfer) executeTransferNFTCreateChangeAtCurrentOwner(
	vmOutput *vmcommon.VMOutput,
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

	logData := [][]byte{acntDst.AddressBytes(), boolToSlice(false)}
	addESDTEntryInVMOutput(vmOutput, []byte(vmInput.Function), tokenID, 0, big.NewInt(0), logData...)

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

		logData = [][]byte{destAddress, boolToSlice(true)}
		addESDTEntryInVMOutput(vmOutput, []byte(vmInput.Function), tokenID, 0, big.NewInt(0), logData...)
	}

	outAcc := &vmcommon.OutputAccount{
		Address:         destAddress,
		Balance:         big.NewInt(0),
		BalanceDelta:    big.NewInt(0),
		OutputTransfers: make([]vmcommon.OutputTransfer, 0),
	}
	outTransfer := vmcommon.OutputTransfer{
		Value: big.NewInt(0),
		Data: []byte(core.BuiltInFunctionESDTNFTCreateRoleTransfer + "@" +
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
	roles, _, err := getESDTRolesForAcnt(e.marshaller, acntDst, esdtTokenRoleKey)
	if err != nil {
		return err
	}

	deleteRoles(roles, [][]byte{[]byte(core.ESDTRoleNFTCreate)})
	return saveRolesToAccount(acntDst, esdtTokenRoleKey, roles, e.marshaller)
}

func (e *esdtNFTCreateRoleTransfer) addCreateRoleToAccount(
	acntDst vmcommon.UserAccountHandler,
	esdtTokenRoleKey []byte,
) error {
	roles, _, err := getESDTRolesForAcnt(e.marshaller, acntDst, esdtTokenRoleKey)
	if err != nil {
		return err
	}

	for _, role := range roles.Roles {
		if bytes.Equal(role, []byte(core.ESDTRoleNFTCreate)) {
			return nil
		}
	}

	roles.Roles = append(roles.Roles, []byte(core.ESDTRoleNFTCreate))
	return saveRolesToAccount(acntDst, esdtTokenRoleKey, roles, e.marshaller)
}

func saveRolesToAccount(
	acntDst vmcommon.UserAccountHandler,
	esdtTokenRoleKey []byte,
	roles *esdt.ESDTRoles,
	marshaller vmcommon.Marshalizer,
) error {
	marshaledData, err := marshaller.Marshal(roles)
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
	vmOutput *vmcommon.VMOutput,
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

	logData := [][]byte{acntDst.AddressBytes(), boolToSlice(true)}
	addESDTEntryInVMOutput(vmOutput, []byte(vmInput.Function), tokenID, 0, big.NewInt(0), logData...)

	return nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtNFTCreateRoleTransfer) IsInterfaceNil() bool {
	return e == nil
}
