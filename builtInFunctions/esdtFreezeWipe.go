package builtInFunctions

import (
	"bytes"
	"math/big"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-vm-common"
)

type esdtFreezeWipe struct {
	baseAlwaysActive
	marshalizer vmcommon.Marshalizer
	keyPrefix   []byte
	wipe        bool
	freeze      bool
}

// NewESDTFreezeWipeFunc returns the esdt freeze/un-freeze/wipe built-in function component
func NewESDTFreezeWipeFunc(
	marshalizer vmcommon.Marshalizer,
	freeze bool,
	wipe bool,
) (*esdtFreezeWipe, error) {
	if check.IfNil(marshalizer) {
		return nil, ErrNilMarshalizer
	}

	e := &esdtFreezeWipe{
		marshalizer: marshalizer,
		keyPrefix:   []byte(vmcommon.ElrondProtectedKeyPrefix + vmcommon.ESDTKeyIdentifier),
		freeze:      freeze,
		wipe:        wipe,
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
	if !bytes.Equal(vmInput.CallerAddr, vmcommon.ESDTSCAddress) {
		return nil, ErrAddressIsNotESDTSystemSC
	}
	if check.IfNil(acntDst) {
		return nil, ErrNilUserAccount
	}

	esdtTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)

	if e.wipe {
		err := e.wipeIfApplicable(acntDst, esdtTokenKey)
		if err != nil {
			return nil, err
		}
	} else {
		err := e.toggleFreeze(acntDst, esdtTokenKey)
		if err != nil {
			return nil, err
		}
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}
	if e.wipe {
		addESDTEntryInVMOutput(vmOutput, []byte(vmcommon.BuiltInFunctionESDTWipe), vmInput.Arguments[0], big.NewInt(0), vmInput.CallerAddr, acntDst.AddressBytes())
	}

	return vmOutput, nil
}

func (e *esdtFreezeWipe) wipeIfApplicable(acntDst vmcommon.UserAccountHandler, tokenKey []byte) error {
	tokenData, err := getESDTDataFromKey(acntDst, tokenKey, e.marshalizer)
	if err != nil {
		return err
	}

	esdtUserMetadata := ESDTUserMetadataFromBytes(tokenData.Properties)
	if !esdtUserMetadata.Frozen {
		return ErrCannotWipeAccountNotFrozen
	}

	return acntDst.AccountDataHandler().SaveKeyValue(tokenKey, nil)
}

func (e *esdtFreezeWipe) toggleFreeze(acntDst vmcommon.UserAccountHandler, tokenKey []byte) error {
	tokenData, err := getESDTDataFromKey(acntDst, tokenKey, e.marshalizer)
	if err != nil {
		return err
	}

	esdtUserMetadata := ESDTUserMetadataFromBytes(tokenData.Properties)
	esdtUserMetadata.Frozen = e.freeze
	tokenData.Properties = esdtUserMetadata.ToBytes()

	err = saveESDTData(acntDst, tokenData, tokenKey, e.marshalizer)
	if err != nil {
		return err
	}

	return nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *esdtFreezeWipe) IsInterfaceNil() bool {
	return e == nil
}
