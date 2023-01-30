package builtInFunctions

import (
	"strings"
	"sync"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

const userNameSep = "."

type migrateUserName struct {
	baseActiveHandler
	delete          bool
	gasCost         uint64
	mapDnsAddresses map[string]struct{}
	mutExecution    sync.RWMutex
}

// NewDeleteUserNameFunc returns a delete username built in function implementation
func NewDeleteUserNameFunc(
	gasCost uint64,
	mapDnsAddresses map[string]struct{},
	enableEpochsHandler vmcommon.EnableEpochsHandler,
) (*migrateUserName, error) {
	m, err := NewMigrateUserNameFunc(gasCost, mapDnsAddresses, enableEpochsHandler)
	if err != nil {
		return nil, err
	}
	m.delete = true
	return m, nil
}

// NewMigrateUserNameFunc returns a migrate username built in function implementation
func NewMigrateUserNameFunc(
	gasCost uint64,
	mapDnsAddresses map[string]struct{},
	enableEpochsHandler vmcommon.EnableEpochsHandler,
) (*migrateUserName, error) {
	if mapDnsAddresses == nil {
		return nil, ErrNilDnsAddresses
	}
	if check.IfNil(enableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}

	m := &migrateUserName{
		gasCost:         gasCost,
		mapDnsAddresses: make(map[string]struct{}, len(mapDnsAddresses)),
	}
	for key := range mapDnsAddresses {
		m.mapDnsAddresses[key] = struct{}{}
	}
	m.activeHandler = enableEpochsHandler.IsMigrateUsernameEnabled

	return m, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (m *migrateUserName) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	m.mutExecution.Lock()
	m.gasCost = gasCost.BuiltInCost.SaveUserName
	m.mutExecution.Unlock()
}

// ProcessBuiltinFunction sets the username to the account if it is allowed
func (m *migrateUserName) ProcessBuiltinFunction(
	_, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	m.mutExecution.RLock()
	defer m.mutExecution.RUnlock()

	err := inputCheckForUserNameCall(vmInput, m.mapDnsAddresses, m.gasCost)
	if err != nil {
		return nil, err
	}

	if check.IfNil(acntDst) {
		return createCrossShardUserNameCall(vmInput, core.BuiltInFunctionSetUserName)
	}

	currentUserName := acntDst.GetUserName()
	if len(currentUserName) == 0 {
		return nil, ErrCannotMigrateNilUserName
	}

	err = checkUsernamesSamePrefix(string(currentUserName), string(vmInput.Arguments[0]))
	if err != nil {
		return nil, err
	}

	acntDst.SetUserName(vmInput.Arguments[0])
	if m.delete {
		acntDst.SetUserName(nil)
	}

	return &vmcommon.VMOutput{GasRemaining: vmInput.GasProvided - m.gasCost, ReturnCode: vmcommon.Ok}, nil
}

func checkUsernamesSamePrefix(oldUserName string, newUserName string) error {
	oldSplitStrings := strings.Split(oldUserName, userNameSep)
	newSplitString := strings.Split(newUserName, userNameSep)

	if len(oldSplitStrings) < 2 || len(newSplitString) < 2 {
		return ErrWrongUserNameSplit
	}

	if oldSplitStrings[0] != newSplitString[0] {
		return ErrUserNamePrefixNotEqual
	}

	return nil
}

// IsInterfaceNil returns true if underlying object in nil
func (m *migrateUserName) IsInterfaceNil() bool {
	return m == nil
}
