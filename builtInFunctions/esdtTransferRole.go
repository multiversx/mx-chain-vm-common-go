package builtInFunctions

import (
	"bytes"
	"math/big"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data/esdt"
	"github.com/multiversx/mx-chain-core-go/marshal"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

const transfer = "transfer"

var transferAddressesKeyPrefix = []byte(core.ProtectedKeyPrefix + transfer + core.ESDTKeyIdentifier)

type esdtTransferAddress struct {
	baseActiveHandler
	set             bool
	marshaller      vmcommon.Marshalizer
	accounts        vmcommon.AccountsAdapter
	maxNumAddresses uint32
}

// NewESDTTransferRoleAddressFunc returns the esdt transfer role address handler built-in function component
func NewESDTTransferRoleAddressFunc(
	accounts vmcommon.AccountsAdapter,
	marshaller marshal.Marshalizer,
	maxNumAddresses uint32,
	set bool,
	enableEpochsHandler vmcommon.EnableEpochsHandler,
) (*esdtTransferAddress, error) {
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if maxNumAddresses < 1 {
		return nil, ErrInvalidMaxNumAddresses
	}
	if check.IfNil(enableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}

	e := &esdtTransferAddress{
		accounts:        accounts,
		marshaller:      marshaller,
		maxNumAddresses: maxNumAddresses,
		set:             set,
	}

	e.baseActiveHandler.activeHandler = func() bool {
		return enableEpochsHandler.IsFlagEnabled(SendAlwaysFlag)
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtTransferAddress) SetNewGasConfig(_ *vmcommon.GasCost) {
}

// ProcessBuiltinFunction resolves ESDT change roles function call
func (e *esdtTransferAddress) ProcessBuiltinFunction(
	_, dstAccount vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	err := checkBasicESDTArguments(vmInput)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(vmInput.CallerAddr, core.ESDTSCAddress) {
		return nil, ErrAddressIsNotESDTSystemSC
	}
	if !vmcommon.IsSystemAccountAddress(vmInput.RecipientAddr) {
		return nil, ErrOnlySystemAccountAccepted
	}

	systemAcc, err := getSystemAccountIfNeeded(vmInput, dstAccount, e.accounts)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	esdtTokenTransferRoleKey := append(transferAddressesKeyPrefix, vmInput.Arguments[0]...)
	addresses, _, err := getESDTRolesForAcnt(e.marshaller, systemAcc, esdtTokenTransferRoleKey)
	if err != nil {
		return nil, err
	}

	if e.set {
		err = e.addNewAddresses(vmInput, addresses)
		if err != nil {
			return nil, err
		}
	} else {
		deleteRoles(addresses, vmInput.Arguments[1:])
	}

	err = saveRolesToAccount(systemAcc, esdtTokenTransferRoleKey, addresses, e.marshaller)
	if err != nil {
		return nil, err
	}

	err = e.accounts.SaveAccount(systemAcc)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}

	logData := append([][]byte{systemAcc.AddressBytes()}, vmInput.Arguments[1:]...)
	addESDTEntryInVMOutput(vmOutput, []byte(vmInput.Function), vmInput.Arguments[0], 0, big.NewInt(0), logData...)

	return vmOutput, nil
}

func (e *esdtTransferAddress) addNewAddresses(vmInput *vmcommon.ContractCallInput, addresses *esdt.ESDTRoles) error {
	for _, newAddress := range vmInput.Arguments[1:] {
		isNew := true
		for _, address := range addresses.Roles {
			if bytes.Equal(newAddress, address) {
				isNew = false
				break
			}
		}
		if isNew {
			addresses.Roles = append(addresses.Roles, newAddress)
		}
	}

	if uint32(len(addresses.Roles)) > e.maxNumAddresses {
		return ErrTooManyTransferAddresses
	}

	return nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtTransferAddress) IsInterfaceNil() bool {
	return e == nil
}
