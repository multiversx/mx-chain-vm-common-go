package builtInFunctions

import (
	"bytes"
	"fmt"
	"math/big"
	"sync"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/atomic"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data/esdt"
	"github.com/ElrondNetwork/elrond-go-core/data/vm"
	"github.com/ElrondNetwork/elrond-vm-common"
)

var noncePrefix = []byte(core.ElrondProtectedKeyPrefix + core.ESDTNFTLatestNonceIdentifier)

type esdtNFTCreate struct {
	baseAlwaysActive
	keyPrefix             []byte
	accounts              vmcommon.AccountsAdapter
	marshalizer           vmcommon.Marshalizer
	globalSettingsHandler vmcommon.ESDTGlobalSettingsHandler
	rolesHandler          vmcommon.ESDTRoleHandler
	funcGasCost           uint64
	gasConfig             vmcommon.BaseOperationCost
	esdtStorageHandler    vmcommon.ESDTNFTStorageHandler
	mutExecution          sync.RWMutex

	valueLengthCheckEnableEpoch uint32
	flagValueLengthCheck        atomic.Flag
}

// NewESDTNFTCreateFunc returns the esdt NFT create built-in function component
func NewESDTNFTCreateFunc(
	funcGasCost uint64,
	gasConfig vmcommon.BaseOperationCost,
	marshalizer vmcommon.Marshalizer,
	globalSettingsHandler vmcommon.ESDTGlobalSettingsHandler,
	rolesHandler vmcommon.ESDTRoleHandler,
	esdtStorageHandler vmcommon.ESDTNFTStorageHandler,
	accounts vmcommon.AccountsAdapter,
	valueLengthCheckEnableEpoch uint32,
	epochNotifier vmcommon.EpochNotifier,
) (*esdtNFTCreate, error) {
	if check.IfNil(marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(globalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
	}
	if check.IfNil(rolesHandler) {
		return nil, ErrNilRolesHandler
	}
	if check.IfNil(esdtStorageHandler) {
		return nil, ErrNilESDTNFTStorageHandler
	}
	if check.IfNil(epochNotifier) {
		return nil, ErrNilEpochHandler
	}
	if check.IfNil(accounts) {
		return nil, ErrNilAccountsAdapter
	}

	e := &esdtNFTCreate{
		keyPrefix:                   []byte(core.ElrondProtectedKeyPrefix + core.ESDTKeyIdentifier),
		marshalizer:                 marshalizer,
		globalSettingsHandler:       globalSettingsHandler,
		rolesHandler:                rolesHandler,
		funcGasCost:                 funcGasCost,
		gasConfig:                   gasConfig,
		esdtStorageHandler:          esdtStorageHandler,
		mutExecution:                sync.RWMutex{},
		valueLengthCheckEnableEpoch: valueLengthCheckEnableEpoch,
		accounts:                    accounts,
	}

	epochNotifier.RegisterNotifyHandler(e)

	return e, nil
}

// EpochConfirmed is called whenever a new epoch is confirmed
func (e *esdtNFTCreate) EpochConfirmed(epoch uint32, _ uint64) {
	e.flagValueLengthCheck.SetValue(epoch >= e.valueLengthCheckEnableEpoch)
	log.Debug("ESDT NFT Create quantity value length check", "enabled", e.flagValueLengthCheck.IsSet())
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtNFTCreate) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.ESDTNFTCreate
	e.gasConfig = gasCost.BaseOperationCost
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves ESDT NFT create function call
// Requires at least 7 arguments:
// arg0 - token identifier
// arg1 - initial quantity
// arg2 - NFT name
// arg3 - Royalties - max 10000
// arg4 - hash
// arg5 - attributes
// arg6+ - multiple entries of URI (minimum 1)
func (e *esdtNFTCreate) ProcessBuiltinFunction(
	acntSnd, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	e.mutExecution.RLock()
	defer e.mutExecution.RUnlock()

	err := checkESDTNFTCreateBurnAddInput(acntSnd, vmInput, e.funcGasCost)
	if err != nil {
		return nil, err
	}

	minNumOfArgs := 7
	if vmInput.CallType == vm.ExecOnDestByCaller {
		minNumOfArgs = 8
	}
	lenArgs := len(vmInput.Arguments)
	if lenArgs < minNumOfArgs {
		return nil, fmt.Errorf("%w, wrong number of arguments", ErrInvalidArguments)
	}

	accountWithRoles := acntSnd
	uris := vmInput.Arguments[6:]
	if vmInput.CallType == vm.ExecOnDestByCaller {
		scAddressWithRoles := vmInput.Arguments[lenArgs-1]
		uris = vmInput.Arguments[6 : lenArgs-1]

		if len(scAddressWithRoles) != len(vmInput.CallerAddr) {
			return nil, ErrInvalidAddressLength
		}
		if bytes.Equal(scAddressWithRoles, vmInput.CallerAddr) {
			return nil, ErrInvalidRcvAddr
		}

		accountWithRoles, err = e.getAccount(scAddressWithRoles)
		if err != nil {
			return nil, err
		}
	}

	tokenID := vmInput.Arguments[0]
	err = e.rolesHandler.CheckAllowedToExecute(accountWithRoles, vmInput.Arguments[0], []byte(core.ESDTRoleNFTCreate))
	if err != nil {
		return nil, err
	}

	nonce, err := getLatestNonce(accountWithRoles, tokenID)
	if err != nil {
		return nil, err
	}

	totalLength := uint64(0)
	for _, arg := range vmInput.Arguments {
		totalLength += uint64(len(arg))
	}
	gasToUse := totalLength*e.gasConfig.StorePerByte + e.funcGasCost
	if vmInput.GasProvided < gasToUse {
		return nil, ErrNotEnoughGas
	}

	royalties := uint32(big.NewInt(0).SetBytes(vmInput.Arguments[3]).Uint64())
	if royalties > core.MaxRoyalty {
		return nil, fmt.Errorf("%w, invalid max royality value", ErrInvalidArguments)
	}

	esdtTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	quantity := big.NewInt(0).SetBytes(vmInput.Arguments[1])
	if quantity.Cmp(zero) <= 0 {
		return nil, fmt.Errorf("%w, invalid quantity", ErrInvalidArguments)
	}
	if quantity.Cmp(big.NewInt(1)) > 0 {
		err = e.rolesHandler.CheckAllowedToExecute(accountWithRoles, vmInput.Arguments[0], []byte(core.ESDTRoleNFTAddQuantity))
		if err != nil {
			return nil, err
		}
	}
	if e.flagValueLengthCheck.IsSet() && len(vmInput.Arguments[1]) > maxLenForAddNFTQuantity {
		return nil, fmt.Errorf("%w max length for quantity in nft create is %d", ErrInvalidArguments, maxLenForAddNFTQuantity)
	}

	nextNonce := nonce + 1
	esdtData := &esdt.ESDigitalToken{
		Type:  uint32(core.NonFungible),
		Value: quantity,
		TokenMetaData: &esdt.MetaData{
			Nonce:      nextNonce,
			Name:       vmInput.Arguments[2],
			Creator:    vmInput.CallerAddr,
			Royalties:  royalties,
			Hash:       vmInput.Arguments[4],
			Attributes: vmInput.Arguments[5],
			URIs:       uris,
		},
	}

	var esdtDataBytes []byte
	esdtDataBytes, err = e.esdtStorageHandler.SaveESDTNFTToken(accountWithRoles.AddressBytes(), accountWithRoles, esdtTokenKey, nextNonce, esdtData, true, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	err = saveLatestNonce(accountWithRoles, tokenID, nextNonce)
	if err != nil {
		return nil, err
	}

	if vmInput.CallType == vm.ExecOnDestByCaller {
		err = e.accounts.SaveAccount(accountWithRoles)
		if err != nil {
			return nil, err
		}
	}

	vmOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: vmInput.GasProvided - gasToUse,
		ReturnData:   [][]byte{big.NewInt(0).SetUint64(nextNonce).Bytes()},
	}

	addESDTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionESDTNFTCreate), vmInput.Arguments[0], nextNonce, quantity, vmInput.CallerAddr, esdtDataBytes)

	return vmOutput, nil
}

func (e *esdtNFTCreate) getAccount(address []byte) (vmcommon.UserAccountHandler, error) {
	account, err := e.accounts.LoadAccount(address)
	if err != nil {
		return nil, err
	}

	userAcc, ok := account.(vmcommon.UserAccountHandler)
	if !ok {
		return nil, ErrWrongTypeAssertion
	}

	return userAcc, nil
}

func getLatestNonce(acnt vmcommon.UserAccountHandler, tokenID []byte) (uint64, error) {
	nonceKey := getNonceKey(tokenID)
	nonceData, err := acnt.AccountDataHandler().RetrieveValue(nonceKey)
	if err != nil {
		return 0, err
	}

	if len(nonceData) == 0 {
		return 0, nil
	}

	return big.NewInt(0).SetBytes(nonceData).Uint64(), nil
}

func saveLatestNonce(acnt vmcommon.UserAccountHandler, tokenID []byte, nonce uint64) error {
	nonceKey := getNonceKey(tokenID)
	return acnt.AccountDataHandler().SaveKeyValue(nonceKey, big.NewInt(0).SetUint64(nonce).Bytes())
}

func computeESDTNFTTokenKey(esdtTokenKey []byte, nonce uint64) []byte {
	return append(esdtTokenKey, big.NewInt(0).SetUint64(nonce).Bytes()...)
}

func checkESDTNFTCreateBurnAddInput(
	account vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
	funcGasCost uint64,
) error {
	err := checkBasicESDTArguments(vmInput)
	if err != nil {
		return err
	}
	if !bytes.Equal(vmInput.CallerAddr, vmInput.RecipientAddr) {
		return ErrInvalidRcvAddr
	}
	if check.IfNil(account) && vmInput.CallType != vm.ExecOnDestByCaller {
		return ErrNilUserAccount
	}
	if vmInput.GasProvided < funcGasCost {
		return ErrNotEnoughGas
	}
	return nil
}

func getNonceKey(tokenID []byte) []byte {
	return append(noncePrefix, tokenID...)
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtNFTCreate) IsInterfaceNil() bool {
	return e == nil
}
