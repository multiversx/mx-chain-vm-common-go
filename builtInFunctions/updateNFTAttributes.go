package builtInFunctions

import (
	"math/big"
	"sync"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/atomic"
)

type esdtNFTupdate struct {
	*baseEnabled
	keyPrefix    []byte
	marshalizer  vmcommon.Marshalizer
	pauseHandler vmcommon.ESDTPauseHandler
	rolesHandler vmcommon.ESDTRoleHandler
	gasConfig    vmcommon.BaseOperationCost
	funcGasCost  uint64
	mutExecution sync.RWMutex
}

// NewESDTNFTAddUriFunc returns the esdt NFT update attribute built-in function component
func NewESDTNFTUpdateAttributesFunc(
	funcGasCost uint64,
	gasConfig vmcommon.BaseOperationCost,
	marshalizer vmcommon.Marshalizer,
	pauseHandler vmcommon.ESDTPauseHandler,
	rolesHandler vmcommon.ESDTRoleHandler,
	activationEpoch uint32,
	epochNotifier vmcommon.EpochNotifier,
) (*esdtNFTupdate, error) {
	if check.IfNil(marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(pauseHandler) {
		return nil, ErrNilPauseHandler
	}
	if check.IfNil(rolesHandler) {
		return nil, ErrNilRolesHandler
	}
	if check.IfNil(epochNotifier) {
		return nil, ErrNilEpochHandler
	}

	e := &esdtNFTupdate{
		keyPrefix:    []byte(vmcommon.ElrondProtectedKeyPrefix + vmcommon.ESDTKeyIdentifier),
		marshalizer:  marshalizer,
		funcGasCost:  funcGasCost,
		mutExecution: sync.RWMutex{},
		pauseHandler: pauseHandler,
		gasConfig:    gasConfig,
		rolesHandler: rolesHandler,
	}

	e.baseEnabled = &baseEnabled{
		function:        vmcommon.BuiltInFunctionESDTNFTUpdateAttributes,
		activationEpoch: activationEpoch,
		flagActivated:   atomic.Flag{},
	}

	epochNotifier.RegisterNotifyHandler(e)

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtNFTupdate) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.ESDTNFTUpdateAttributes
	e.gasConfig = gasCost.BaseOperationCost
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves ESDT NFT add quantity function call
// Requires 3 arguments:
// arg0 - token identifier
// arg1 - nonce
// arg2 - new attributes
func (e *esdtNFTupdate) ProcessBuiltinFunction(
	acntSnd, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	e.mutExecution.RLock()
	defer e.mutExecution.RUnlock()

	err := checkESDTNFTCreateBurnAddInput(acntSnd, vmInput, e.funcGasCost)
	if err != nil {
		return nil, err
	}
	if len(vmInput.Arguments) != 3 {
		return nil, ErrInvalidArguments
	}

	err = e.rolesHandler.CheckAllowedToExecute(acntSnd, vmInput.Arguments[0], []byte(vmcommon.ESDTRoleNFTUpdateAttributes))
	if err != nil {
		return nil, err
	}

	gasCostForStore := uint64(len(vmInput.Arguments[2])) * e.gasConfig.StorePerByte
	if vmInput.GasProvided < e.funcGasCost+gasCostForStore {
		return nil, ErrNotEnoughGas
	}

	esdtTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	nonce := big.NewInt(0).SetBytes(vmInput.Arguments[1]).Uint64()
	if nonce == 0 {
		return nil, ErrNFTDoesNotHaveMetadata
	}
	esdtData, err := getESDTNFTTokenOnSender(acntSnd, esdtTokenKey, nonce, e.marshalizer)
	if err != nil {
		return nil, err
	}

	esdtData.TokenMetaData.Attributes = vmInput.Arguments[2]

	_, err = saveESDTNFTToken(acntSnd, esdtTokenKey, esdtData, e.marshalizer, e.pauseHandler, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	logEntry := newEntryForNFT(vmcommon.BuiltInFunctionESDTNFTUpdateAttributes, vmInput.CallerAddr, vmInput.Arguments[0], nonce)
	vmOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: vmInput.GasProvided - e.funcGasCost - gasCostForStore,
		Logs:         []*vmcommon.LogEntry{logEntry},
	}
	return vmOutput, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtNFTupdate) IsInterfaceNil() bool {
	return e == nil
}
