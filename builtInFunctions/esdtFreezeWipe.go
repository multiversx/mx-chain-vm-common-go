package builtInFunctions

import (
	"bytes"
	"math/big"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

type esdtFreezeWipe struct {
	baseAlwaysActiveHandler
	esdtStorageHandler vmcommon.ESDTNFTStorageHandler
	marshaller         vmcommon.Marshalizer
	keyPrefix          []byte
	wipe               bool
	freeze             bool
}

// NewESDTFreezeWipeFunc returns the esdt freeze/un-freeze/wipe built-in function component
func NewESDTFreezeWipeFunc(
	esdtStorageHandler vmcommon.ESDTNFTStorageHandler,
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

	e := &esdtFreezeWipe{
		esdtStorageHandler: esdtStorageHandler,
		marshaller:         marshaller,
		keyPrefix:          []byte(baseESDTKeyPrefix),
		freeze:             freeze,
		wipe:               wipe,
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

	tokenIDKey := append(e.keyPrefix, identifier...)
	err = e.esdtStorageHandler.AddToLiquiditySystemAcc(tokenIDKey, nonce, big.NewInt(0).Neg(tokenData.Value))
	if err != nil {
		return nil, err
	}

	wipedAmount := vmcommon.ZeroValueIfNil(tokenData.Value)
	return wipedAmount, nil
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
