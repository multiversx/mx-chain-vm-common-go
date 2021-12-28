package builtInFunctions

import (
	"math/big"
	"sync"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/atomic"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-vm-common"
)

type esdtNFTAddUri struct {
	*baseEnabled
	keyPrefix             []byte
	esdtStorageHandler    vmcommon.ESDTNFTStorageHandler
	globalSettingsHandler vmcommon.ESDTGlobalSettingsHandler
	rolesHandler          vmcommon.ESDTRoleHandler
	gasConfig             vmcommon.BaseOperationCost
	funcGasCost           uint64
	mutExecution          sync.RWMutex
}

// NewESDTNFTAddUriFunc returns the esdt NFT add URI built-in function component
func NewESDTNFTAddUriFunc(
	funcGasCost uint64,
	gasConfig vmcommon.BaseOperationCost,
	esdtStorageHandler vmcommon.ESDTNFTStorageHandler,
	globalSettingsHandler vmcommon.ESDTGlobalSettingsHandler,
	rolesHandler vmcommon.ESDTRoleHandler,
	activationEpoch uint32,
	epochNotifier vmcommon.EpochNotifier,
) (*esdtNFTAddUri, error) {
	if check.IfNil(esdtStorageHandler) {
		return nil, ErrNilESDTNFTStorageHandler
	}
	if check.IfNil(globalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
	}
	if check.IfNil(rolesHandler) {
		return nil, ErrNilRolesHandler
	}
	if check.IfNil(epochNotifier) {
		return nil, ErrNilEpochHandler
	}

	e := &esdtNFTAddUri{
		keyPrefix:             []byte(core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier),
		esdtStorageHandler:    esdtStorageHandler,
		funcGasCost:           funcGasCost,
		mutExecution:          sync.RWMutex{},
		globalSettingsHandler: globalSettingsHandler,
		gasConfig:             gasConfig,
		rolesHandler:          rolesHandler,
	}

	e.baseEnabled = &baseEnabled{
		function:        core.BuiltInFunctionESDTNFTAddURI,
		activationEpoch: activationEpoch,
		flagActivated:   atomic.Flag{},
	}

	epochNotifier.RegisterNotifyHandler(e)

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtNFTAddUri) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.ESDTNFTAddURI
	e.gasConfig = gasCost.BaseOperationCost
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves ESDT NFT add quantity function call
// Requires 3 arguments:
// arg0 - token identifier
// arg1 - nonce
// arg2 - attributes to add
func (e *esdtNFTAddUri) ProcessBuiltinFunction(
	acntSnd, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	e.mutExecution.RLock()
	defer e.mutExecution.RUnlock()

	err := checkESDTNFTCreateBurnAddInput(acntSnd, vmInput, e.funcGasCost)
	if err != nil {
		return nil, err
	}
	if len(vmInput.Arguments) < 3 {
		return nil, ErrInvalidArguments
	}

	err = e.rolesHandler.CheckAllowedToExecute(acntSnd, vmInput.Arguments[0], []byte(core.ESDTRoleNFTAddURI))
	if err != nil {
		return nil, err
	}

	gasCostForStore := e.getGasCostForURIStore(vmInput)
	if vmInput.GasProvided < e.funcGasCost+gasCostForStore {
		return nil, ErrNotEnoughGas
	}

	esdtTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	nonce := big.NewInt(0).SetBytes(vmInput.Arguments[1]).Uint64()
	if nonce == 0 {
		return nil, ErrNFTDoesNotHaveMetadata
	}
	esdtData, err := e.esdtStorageHandler.GetESDTNFTTokenOnSender(acntSnd, esdtTokenKey, nonce)
	if err != nil {
		return nil, err
	}

	esdtData.TokenMetaData.URIs = append(esdtData.TokenMetaData.URIs, vmInput.Arguments[2:]...)

	_, err = e.esdtStorageHandler.SaveESDTNFTToken(acntSnd.AddressBytes(), acntSnd, esdtTokenKey, nonce, esdtData, true, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: vmInput.GasProvided - e.funcGasCost - gasCostForStore,
	}

	addESDTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionESDTNFTAddURI), vmInput.Arguments[0], nonce, big.NewInt(0), vmInput.CallerAddr)

	return vmOutput, nil
}

func (e *esdtNFTAddUri) getGasCostForURIStore(vmInput *vmcommon.ContractCallInput) uint64 {
	lenURIs := 0
	for _, uri := range vmInput.Arguments[2:] {
		lenURIs += len(uri)
	}
	return uint64(lenURIs) * e.gasConfig.StorePerByte
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtNFTAddUri) IsInterfaceNil() bool {
	return e == nil
}
