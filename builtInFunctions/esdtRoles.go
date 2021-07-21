package builtInFunctions

import (
	"bytes"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data/esdt"
	"github.com/ElrondNetwork/elrond-vm-common"
)

var roleKeyPrefix = []byte(vmcommon.ElrondProtectedKeyPrefix + vmcommon.ESDTRoleIdentifier + vmcommon.ESDTKeyIdentifier)

type esdtRoles struct {
	baseAlwaysActive
	set         bool
	marshalizer vmcommon.Marshalizer
}

// NewESDTRolesFunc returns the esdt change roles built-in function component
func NewESDTRolesFunc(
	marshalizer vmcommon.Marshalizer,
	set bool,
) (*esdtRoles, error) {
	if check.IfNil(marshalizer) {
		return nil, ErrNilMarshalizer
	}

	e := &esdtRoles{
		set:         set,
		marshalizer: marshalizer,
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtRoles) SetNewGasConfig(_ *vmcommon.GasCost) {
}

// ProcessBuiltinFunction resolves ESDT change roles function call
func (e *esdtRoles) ProcessBuiltinFunction(
	_, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	err := checkBasicESDTArguments(vmInput)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(vmInput.CallerAddr, vmcommon.ESDTSCAddress) {
		return nil, ErrAddressIsNotESDTSystemSC
	}
	if check.IfNil(acntDst) {
		return nil, ErrNilUserAccount
	}

	esdtTokenRoleKey := append(roleKeyPrefix, vmInput.Arguments[0]...)

	roles, _, err := getESDTRolesForAcnt(e.marshalizer, acntDst, esdtTokenRoleKey)
	if err != nil {
		return nil, err
	}

	if e.set {
		roles.Roles = append(roles.Roles, vmInput.Arguments[1:]...)
	} else {
		deleteRoles(roles, vmInput.Arguments[1:])
	}

	err = saveRolesToAccount(acntDst, esdtTokenRoleKey, roles, e.marshalizer)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}
	return vmOutput, nil
}

func deleteRoles(roles *esdt.ESDTRoles, deleteRoles [][]byte) {
	for _, deleteRole := range deleteRoles {
		index, exist := doesRoleExist(roles, deleteRole)
		if !exist {
			continue
		}

		copy(roles.Roles[index:], roles.Roles[index+1:])
		roles.Roles[len(roles.Roles)-1] = nil
		roles.Roles = roles.Roles[:len(roles.Roles)-1]
	}
}

func doesRoleExist(roles *esdt.ESDTRoles, role []byte) (int, bool) {
	for i, currentRole := range roles.Roles {
		if bytes.Equal(currentRole, role) {
			return i, true
		}
	}
	return -1, false
}

func getESDTRolesForAcnt(
	marshalizer vmcommon.Marshalizer,
	acnt vmcommon.UserAccountHandler,
	key []byte,
) (*esdt.ESDTRoles, bool, error) {
	roles := &esdt.ESDTRoles{
		Roles: make([][]byte, 0),
	}

	marshaledData, err := acnt.AccountDataHandler().RetrieveValue(key)
	if err != nil || len(marshaledData) == 0 {
		return roles, true, nil
	}

	err = marshalizer.Unmarshal(roles, marshaledData)
	if err != nil {
		return nil, false, err
	}

	return roles, false, nil
}

// CheckAllowedToExecute returns error if the account is not allowed to execute the given action
func (e *esdtRoles) CheckAllowedToExecute(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
	if check.IfNil(account) {
		return ErrNilUserAccount
	}

	esdtTokenRoleKey := append(roleKeyPrefix, tokenID...)
	roles, isNew, err := getESDTRolesForAcnt(e.marshalizer, account, esdtTokenRoleKey)
	if err != nil {
		return err
	}
	if isNew {
		return ErrActionNotAllowed
	}

	for _, role := range roles.Roles {
		if bytes.Equal(role, action) {
			return nil
		}
	}

	return ErrActionNotAllowed
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtRoles) IsInterfaceNil() bool {
	return e == nil
}
