package builtInFunctions

import (
	"bytes"
	"math/big"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/atomic"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

const transfer = "transfer"

var transferAddressesKeyPrefix = []byte(core.ElrondProtectedKeyPrefix + transfer + core.ESDTKeyIdentifier)

type esdtTransferAddress struct {
	*baseEnabled
	set             bool
	marshalizer     vmcommon.Marshalizer
	accounts        vmcommon.AccountsAdapter
	maxNumAddresses uint32
}

// NewESDTTransferRoleAddressFunc returns the esdt transfer role address handler built-in function component
func NewESDTTransferRoleAddressFunc(
	accounts vmcommon.AccountsAdapter,
	marshalizer marshal.Marshalizer,
	activationEpoch uint32,
	epochNotifier vmcommon.EpochNotifier,
	maxNumAddresses uint32,
	set bool,
) (*esdtTransferAddress, error) {
	if check.IfNil(marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(epochNotifier) {
		return nil, ErrNilEpochHandler
	}
	if check.IfNil(accounts) {
		return nil, ErrNilAccountsAdapter
	}

	e := &esdtTransferAddress{
		accounts:        accounts,
		marshalizer:     marshalizer,
		maxNumAddresses: maxNumAddresses,
		set:             set,
	}

	e.baseEnabled = &baseEnabled{
		function:        vmcommon.BuiltInFunctionESDTTransferRoleAddAddress,
		activationEpoch: activationEpoch,
		flagActivated:   atomic.Flag{},
	}
	if !set {
		e.function = vmcommon.BuiltInFunctionESDTTransferRoleDeleteAddress
	}

	epochNotifier.RegisterNotifyHandler(e)

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtTransferAddress) SetNewGasConfig(_ *vmcommon.GasCost) {
}

// ProcessBuiltinFunction resolves ESDT change roles function call
func (e *esdtTransferAddress) ProcessBuiltinFunction(
	_, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	err := checkBasicESDTArguments(vmInput)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(vmInput.CallerAddr, core.ESDTSCAddress) {
		return nil, ErrAddressIsNotESDTSystemSC
	}

	systemAcc, err := e.getSystemAccount()
	if err != nil {
		return nil, err
	}

	esdtTokenTransferRoleKey := append(transferAddressesKeyPrefix, vmInput.Arguments[0]...)
	addresses, _, err := getESDTRolesForAcnt(e.marshalizer, systemAcc, esdtTokenTransferRoleKey)
	if err != nil {
		return nil, err
	}

	if e.set {
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
			return nil, ErrTooManyTransferAddresses
		}
	} else {
		deleteRoles(addresses, vmInput.Arguments[1:])
	}

	err = saveRolesToAccount(systemAcc, esdtTokenTransferRoleKey, addresses, e.marshalizer)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}

	logData := append([][]byte{systemAcc.AddressBytes()}, vmInput.Arguments[1:]...)
	addESDTEntryInVMOutput(vmOutput, []byte(vmInput.Function), vmInput.Arguments[0], 0, big.NewInt(0), logData...)

	return vmOutput, nil
}

func (e *esdtTransferAddress) getSystemAccount() (vmcommon.UserAccountHandler, error) {
	systemSCAccount, err := e.accounts.LoadAccount(vmcommon.SystemAccountAddress)
	if err != nil {
		return nil, err
	}

	userAcc, ok := systemSCAccount.(vmcommon.UserAccountHandler)
	if !ok {
		return nil, ErrWrongTypeAssertion
	}

	return userAcc, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtTransferAddress) IsInterfaceNil() bool {
	return e == nil
}
