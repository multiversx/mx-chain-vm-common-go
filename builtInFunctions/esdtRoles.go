package builtInFunctions

import (
	"bytes"
	"math"
	"math/big"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data/esdt"
	"github.com/multiversx/mx-chain-vm-common-go"
)

var roleKeyPrefix = []byte(core.ProtectedKeyPrefix + core.ESDTRoleIdentifier + core.ESDTKeyIdentifier)

type esdtRoles struct {
	baseAlwaysActiveHandler
	set                    bool
	marshaller             vmcommon.Marshalizer
	crossChainTokenChecker CrossChainTokenCheckerHandler
	crossChainActions      map[string]struct{}
}

// NewESDTRolesFunc returns the esdt change roles built-in function component
func NewESDTRolesFunc(
	marshaller vmcommon.Marshalizer,
	crossChainTokenChecker CrossChainTokenCheckerHandler,
	set bool,
) (*esdtRoles, error) {
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(crossChainTokenChecker) {
		return nil, ErrNilCrossChainTokenChecker
	}

	e := &esdtRoles{
		set:                    set,
		marshaller:             marshaller,
		crossChainActions:      getCrossChainActions(),
		crossChainTokenChecker: crossChainTokenChecker,
	}

	return e, nil
}

func getCrossChainActions() map[string]struct{} {
	actions := make(map[string]struct{})

	actions[core.ESDTRoleLocalMint] = struct{}{}
	actions[core.ESDTRoleNFTAddQuantity] = struct{}{}
	actions[core.ESDTRoleNFTCreate] = struct{}{}
	actions[core.ESDTRoleLocalBurn] = struct{}{}

	return actions
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
	if !bytes.Equal(vmInput.CallerAddr, core.ESDTSCAddress) {
		return nil, ErrAddressIsNotESDTSystemSC
	}
	if check.IfNil(acntDst) {
		return nil, ErrNilUserAccount
	}

	esdtTokenRoleKey := append(roleKeyPrefix, vmInput.Arguments[0]...)

	roles, _, err := getESDTRolesForAcnt(e.marshaller, acntDst, esdtTokenRoleKey)
	if err != nil {
		return nil, err
	}

	if e.set {
		roles.Roles = append(roles.Roles, vmInput.Arguments[1:]...)
	} else {
		deleteRoles(roles, vmInput.Arguments[1:])
	}

	for _, arg := range vmInput.Arguments[1:] {
		if !bytes.Equal(arg, []byte(core.ESDTRoleNFTCreateMultiShard)) {
			continue
		}

		err = saveLatestNonce(acntDst, vmInput.Arguments[0], computeStartNonce(vmInput.RecipientAddr))
		if err != nil {
			return nil, err
		}

		break
	}

	err = saveRolesToAccount(acntDst, esdtTokenRoleKey, roles, e.marshaller)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}

	logData := append([][]byte{acntDst.AddressBytes()}, vmInput.Arguments[1:]...)
	addESDTEntryInVMOutput(vmOutput, []byte(vmInput.Function), vmInput.Arguments[0], 0, big.NewInt(0), logData...)

	return vmOutput, nil
}

// Nonces on multi shard NFT create are from (LastByte * MaxUint64 / 256), this is in order to differentiate them
// even like this, if one contract makes 1000 NFT create on each block, it would need 14 million years to occupy the whole space
// 2 ^ 64 / 256 / 1000 / 14400 / 365 ~= 14 million
func computeStartNonce(destAddress []byte) uint64 {
	lastByteOfAddress := uint64(destAddress[len(destAddress)-1])
	startNonce := (math.MaxUint64 / 256) * lastByteOfAddress
	return startNonce
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
	marshaller vmcommon.Marshalizer,
	acnt vmcommon.UserAccountHandler,
	key []byte,
) (*esdt.ESDTRoles, bool, error) {
	roles := &esdt.ESDTRoles{
		Roles: make([][]byte, 0),
	}

	marshaledData, _, err := acnt.AccountDataHandler().RetrieveValue(key)
	if core.IsGetNodeFromDBError(err) {
		return nil, false, err
	}
	if err != nil || len(marshaledData) == 0 {
		return roles, true, nil
	}

	err = marshaller.Unmarshal(roles, marshaledData)
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

	if e.isAllowedToExecuteCrossChain(account.AddressBytes(), tokenID, action) {
		return nil
	}

	esdtTokenRoleKey := append(roleKeyPrefix, tokenID...)
	roles, isNew, err := getESDTRolesForAcnt(e.marshaller, account, esdtTokenRoleKey)
	if err != nil {
		return err
	}
	if isNew {
		return ErrActionNotAllowed
	}
	_, exist := doesRoleExist(roles, action)
	if !exist {
		return ErrActionNotAllowed
	}

	return nil
}

func (e *esdtRoles) isAllowedToExecuteCrossChain(address []byte, tokenID []byte, action []byte) bool {
	actionStr := string(action)
	if _, isCrossChainAction := e.crossChainActions[actionStr]; !isCrossChainAction {
		return false
	}

	return e.crossChainTokenChecker.IsCrossChainOperationAllowed(address, tokenID)
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtRoles) IsInterfaceNil() bool {
	return e == nil
}
