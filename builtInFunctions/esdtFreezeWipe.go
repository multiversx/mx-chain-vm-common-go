package builtInFunctions

import (
	"bytes"
	"math/big"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

type esdtFreezeWipe struct {
	baseAlwaysActiveHandler
	esdtStorageHandler  vmcommon.ESDTNFTStorageHandler
	enableEpochsHandler vmcommon.EnableEpochsHandler
	marshaller          vmcommon.Marshalizer
	keyPrefix           []byte
	wipe                bool
	freeze              bool
}

// NewESDTFreezeWipeFunc returns the esdt freeze/un-freeze/wipe built-in function component
func NewESDTFreezeWipeFunc(
	esdtStorageHandler vmcommon.ESDTNFTStorageHandler,
	enableEpochsHandler vmcommon.EnableEpochsHandler,
	marshaller vmcommon.Marshalizer,
	freeze bool,
	wipe bool,
) (*esdtFreezeWipe, error) {
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(esdtStorageHandler) {
		return nil, ErrNilESDTNFTStorageHandler
	}
	if check.IfNil(enableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}

	e := &esdtFreezeWipe{
		esdtStorageHandler:  esdtStorageHandler,
		enableEpochsHandler: enableEpochsHandler,
		marshaller:          marshaller,
		keyPrefix:           []byte(baseESDTKeyPrefix),
		freeze:              freeze,
		wipe:                wipe,
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtFreezeWipe) SetNewGasConfig(_ *vmcommon.GasCost) {
}

// ProcessBuiltinFunction resolves ESDT transfer function call
func (e *esdtFreezeWipe) ProcessBuiltinFunction(
	_, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	if vmInput == nil {
		return nil, ErrNilVmInput
	}
	if vmInput.CallValue.Cmp(zero) != 0 {
		return nil, ErrBuiltInFunctionCalledWithValue
	}
	if len(vmInput.Arguments) != 1 {
		return nil, ErrInvalidArguments
	}
	if !bytes.Equal(vmInput.CallerAddr, core.ESDTSCAddress) {
		return nil, ErrAddressIsNotESDTSystemSC
	}
	if check.IfNil(acntDst) {
		return nil, ErrNilUserAccount
	}

	esdtTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	identifier, nonce := extractTokenIdentifierAndNonceESDTWipe(vmInput.Arguments[0])

	var amount *big.Int
	var err error

	if e.wipe {
		amount, err = e.wipeIfApplicable(acntDst, esdtTokenKey, identifier, nonce)
		if err != nil {
			return nil, err
		}

	} else {
		amount, err = e.toggleFreeze(acntDst, esdtTokenKey)
		if err != nil {
			return nil, err
		}
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}
	addESDTEntryInVMOutput(vmOutput, []byte(vmInput.Function), identifier, nonce, amount, vmInput.CallerAddr, acntDst.AddressBytes())

	return vmOutput, nil
}

func (e *esdtFreezeWipe) wipeIfApplicable(acntDst vmcommon.UserAccountHandler, tokenKey []byte, identifier []byte, nonce uint64) (*big.Int, error) {
	tokenData, err := getESDTDataFromKey(acntDst, tokenKey, e.marshaller)
	if err != nil {
		return nil, err
	}

	esdtUserMetadata := ESDTUserMetadataFromBytes(tokenData.Properties)
	if !esdtUserMetadata.Frozen {
		return nil, ErrCannotWipeAccountNotFrozen
	}

	err = acntDst.AccountDataHandler().SaveKeyValue(tokenKey, nil)
	if err != nil {
		return nil, err
	}

	err = e.removeLiquidity(identifier, nonce, tokenData.Value)
	if err != nil {
		return nil, err
	}

	wipedAmount := vmcommon.ZeroValueIfNil(tokenData.Value)
	return wipedAmount, nil
}

func (e *esdtFreezeWipe) removeLiquidity(tokenIdentifier []byte, nonce uint64, value *big.Int) error {
	if !e.enableEpochsHandler.IsFlagEnabled(WipeSingleNFTLiquidityDecreaseFlag) {
		return nil
	}

	tokenIDKey := append(e.keyPrefix, tokenIdentifier...)
	return e.esdtStorageHandler.AddToLiquiditySystemAcc(tokenIDKey, nonce, big.NewInt(0).Neg(value))
}

func (e *esdtFreezeWipe) toggleFreeze(acntDst vmcommon.UserAccountHandler, tokenKey []byte) (*big.Int, error) {
	tokenData, err := getESDTDataFromKey(acntDst, tokenKey, e.marshaller)
	if err != nil {
		return nil, err
	}

	esdtUserMetadata := ESDTUserMetadataFromBytes(tokenData.Properties)
	esdtUserMetadata.Frozen = e.freeze
	tokenData.Properties = esdtUserMetadata.ToBytes()

	err = saveESDTData(acntDst, tokenData, tokenKey, e.marshaller)
	if err != nil {
		return nil, err
	}

	frozenAmount := vmcommon.ZeroValueIfNil(tokenData.Value)
	return frozenAmount, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtFreezeWipe) IsInterfaceNil() bool {
	return e == nil
}
