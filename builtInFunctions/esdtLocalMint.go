package builtInFunctions

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-vm-common-go"
)

type esdtLocalMint struct {
	baseAlwaysActiveHandler
	keyPrefix              []byte
	marshaller             vmcommon.Marshalizer
	globalSettingsHandler  vmcommon.ESDTGlobalSettingsHandler
	rolesHandler           vmcommon.ESDTRoleHandler
	enableEpochsHandler    vmcommon.EnableEpochsHandler
	crossChainTokenChecker CrossChainTokenCheckerHandler
	funcGasCost            uint64
	mutExecution           sync.RWMutex
}

// NewESDTLocalMintFunc returns the esdt local mint built-in function component
func NewESDTLocalMintFunc(args ESDTLocalMintBurnFuncArgs) (*esdtLocalMint, error) {
	if check.IfNil(args.Marshaller) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(args.GlobalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
	}
	if check.IfNil(args.RolesHandler) {
		return nil, ErrNilRolesHandler
	}
	if check.IfNil(args.EnableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}
	if check.IfNil(args.CrossChainTokenChecker) {
		return nil, ErrNilCrossChainTokenChecker
	}

	e := &esdtLocalMint{
		keyPrefix:              []byte(baseESDTKeyPrefix),
		marshaller:             args.Marshaller,
		globalSettingsHandler:  args.GlobalSettingsHandler,
		rolesHandler:           args.RolesHandler,
		funcGasCost:            args.FuncGasCost,
		enableEpochsHandler:    args.EnableEpochsHandler,
		mutExecution:           sync.RWMutex{},
		crossChainTokenChecker: args.CrossChainTokenChecker,
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtLocalMint) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.ESDTLocalMint
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves ESDT local mint function call
func (e *esdtLocalMint) ProcessBuiltinFunction(
	acntSnd, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	e.mutExecution.RLock()
	defer e.mutExecution.RUnlock()

	err := checkInputArgumentsForLocalAction(acntSnd, vmInput, e.funcGasCost)
	if err != nil {
		return nil, err
	}

	tokenID := vmInput.Arguments[0]
	err = e.isAllowedToMint(acntSnd, tokenID)
	if err != nil {
		return nil, err
	}

	if len(vmInput.Arguments[1]) > core.MaxLenForESDTIssueMint {
		if e.enableEpochsHandler.IsFlagEnabled(ConsistentTokensValuesLengthCheckFlag) {
			return nil, fmt.Errorf("%w: max length for esdt local mint value is %d", ErrInvalidArguments, core.MaxLenForESDTIssueMint)
		}
		// backward compatibility - return old error
		return nil, fmt.Errorf("%w max length for esdt issue is %d", ErrInvalidArguments, core.MaxLenForESDTIssueMint)
	}

	value := big.NewInt(0).SetBytes(vmInput.Arguments[1])
	esdtTokenKey := append(e.keyPrefix, tokenID...)
	err = addToESDTBalance(acntSnd, esdtTokenKey, big.NewInt(0).Set(value), e.marshaller, e.globalSettingsHandler, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok, GasRemaining: vmInput.GasProvided - e.funcGasCost}

	addESDTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionESDTLocalMint), vmInput.Arguments[0], 0, value, vmInput.CallerAddr)

	return vmOutput, nil
}

func (e *esdtLocalMint) isAllowedToMint(acntSnd vmcommon.UserAccountHandler, tokenID []byte) error {
	if e.crossChainTokenChecker.IsCrossChainOperation(tokenID) && e.crossChainTokenChecker.IsWhiteListed(acntSnd.AddressBytes()) {
		return nil
	}

	return e.rolesHandler.CheckAllowedToExecute(acntSnd, tokenID, []byte(core.ESDTRoleLocalMint))
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtLocalMint) IsInterfaceNil() bool {
	return e == nil
}
