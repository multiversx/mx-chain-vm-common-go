package builtInFunctions

import (
	"bytes"
	"math/big"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data/esdt"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

const numArgsPerAdd = 3

type esdtDeleteMetaData struct {
	baseActiveHandler
	allowedAddress []byte
	delete         bool
	accounts       vmcommon.AccountsAdapter
	keyPrefix      []byte
	marshaller     vmcommon.Marshalizer
	funcGasCost    uint64
	function       string
}

// ArgsNewESDTDeleteMetadata defines the argument list for new esdt delete metadata built in function
type ArgsNewESDTDeleteMetadata struct {
	FuncGasCost         uint64
	Marshalizer         vmcommon.Marshalizer
	Accounts            vmcommon.AccountsAdapter
	AllowedAddress      []byte
	Delete              bool
	EnableEpochsHandler vmcommon.EnableEpochsHandler
}

// NewESDTDeleteMetadataFunc returns the esdt metadata deletion built-in function component
func NewESDTDeleteMetadataFunc(
	args ArgsNewESDTDeleteMetadata,
) (*esdtDeleteMetaData, error) {
	if check.IfNil(args.Marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(args.Accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if check.IfNil(args.EnableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}

	e := &esdtDeleteMetaData{
		keyPrefix:      []byte(baseESDTKeyPrefix),
		marshaller:     args.Marshalizer,
		funcGasCost:    args.FuncGasCost,
		accounts:       args.Accounts,
		allowedAddress: args.AllowedAddress,
		delete:         args.Delete,
		function:       core.BuiltInFunctionMultiESDTNFTTransfer,
	}

	e.baseActiveHandler.activeHandler = args.EnableEpochsHandler.IsSendAlwaysFlagEnabled

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *esdtDeleteMetaData) SetNewGasConfig(_ *vmcommon.GasCost) {
}

// ProcessBuiltinFunction resolves ESDT delete and add metadata function call
func (e *esdtDeleteMetaData) ProcessBuiltinFunction(
	_, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	if vmInput == nil {
		return nil, ErrNilVmInput
	}
	if vmInput.CallValue.Cmp(zero) != 0 {
		return nil, ErrBuiltInFunctionCalledWithValue
	}
	if !bytes.Equal(vmInput.CallerAddr, e.allowedAddress) {
		return nil, ErrAddressIsNotAllowed
	}
	if !bytes.Equal(vmInput.CallerAddr, vmInput.RecipientAddr) {
		return nil, ErrInvalidRcvAddr
	}

	if e.delete {
		err := e.deleteMetadata(vmInput.Arguments)
		if err != nil {
			return nil, err
		}
	} else {
		err := e.addMetadata(vmInput.Arguments)
		if err != nil {
			return nil, err
		}
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}

	return vmOutput, nil
}

// input is list(tokenID-numIntervals-list(start,end))
func (e *esdtDeleteMetaData) deleteMetadata(args [][]byte) error {
	lenArgs := uint64(len(args))
	if lenArgs < 4 {
		return ErrInvalidNumOfArgs
	}

	systemAcc, err := e.getSystemAccount()
	if err != nil {
		return err
	}

	for i := uint64(0); i+1 < uint64(len(args)); {
		tokenID := args[i]
		numIntervals := big.NewInt(0).SetBytes(args[i+1]).Uint64()
		i += 2

		if !vmcommon.ValidateToken(tokenID) {
			return ErrInvalidTokenID
		}

		if i >= lenArgs {
			return ErrInvalidNumOfArgs
		}

		err = e.deleteMetadataForListIntervals(systemAcc, tokenID, args, i, numIntervals)
		if err != nil {
			return err
		}

		i += numIntervals * 2
	}

	err = e.accounts.SaveAccount(systemAcc)
	if err != nil {
		return err
	}

	return nil
}

func (e *esdtDeleteMetaData) deleteMetadataForListIntervals(
	systemAcc vmcommon.UserAccountHandler,
	tokenID []byte,
	args [][]byte,
	index, numIntervals uint64,
) error {
	lenArgs := uint64(len(args))
	for j := index; j < index+numIntervals*2; j += 2 {
		if j > lenArgs-2 {
			return ErrInvalidNumOfArgs
		}

		startIndex := big.NewInt(0).SetBytes(args[j]).Uint64()
		endIndex := big.NewInt(0).SetBytes(args[j+1]).Uint64()

		err := e.deleteMetadataForInterval(systemAcc, tokenID, startIndex, endIndex)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *esdtDeleteMetaData) deleteMetadataForInterval(
	systemAcc vmcommon.UserAccountHandler,
	tokenID []byte,
	startIndex, endIndex uint64,
) error {
	if endIndex < startIndex {
		return ErrInvalidArguments
	}
	if startIndex == 0 {
		return ErrInvalidNonce
	}

	esdtTokenKey := append(e.keyPrefix, tokenID...)
	for nonce := startIndex; nonce <= endIndex; nonce++ {
		esdtNFTTokenKey := computeESDTNFTTokenKey(esdtTokenKey, nonce)

		err := systemAcc.AccountDataHandler().SaveKeyValue(esdtNFTTokenKey, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

// input is list(tokenID-nonce-metadata)
func (e *esdtDeleteMetaData) addMetadata(args [][]byte) error {
	if len(args)%numArgsPerAdd != 0 || len(args) < numArgsPerAdd {
		return ErrInvalidNumOfArgs
	}

	systemAcc, err := e.getSystemAccount()
	if err != nil {
		return err
	}

	for i := 0; i < len(args); i += numArgsPerAdd {
		tokenID := args[i]
		nonce := big.NewInt(0).SetBytes(args[i+1]).Uint64()
		if nonce == 0 {
			return ErrInvalidNonce
		}

		if !vmcommon.ValidateToken(tokenID) {
			return ErrInvalidTokenID
		}

		esdtTokenKey := append(e.keyPrefix, tokenID...)
		esdtNFTTokenKey := computeESDTNFTTokenKey(esdtTokenKey, nonce)
		metaData := &esdt.MetaData{}
		err = e.marshaller.Unmarshal(metaData, args[i+2])
		if err != nil {
			return err
		}
		if metaData.Nonce != nonce {
			return ErrInvalidMetadata
		}

		var tokenFromSystemSC *esdt.ESDigitalToken
		tokenFromSystemSC, err = e.getESDTDigitalTokenDataFromSystemAccount(systemAcc, esdtNFTTokenKey)
		if err != nil {
			return err
		}

		if tokenFromSystemSC != nil && tokenFromSystemSC.TokenMetaData != nil {
			return ErrTokenHasValidMetadata
		}

		if tokenFromSystemSC == nil {
			tokenFromSystemSC = &esdt.ESDigitalToken{
				Value: big.NewInt(0),
				Type:  uint32(core.NonFungible),
			}
		}
		tokenFromSystemSC.TokenMetaData = metaData
		err = e.marshalAndSaveData(systemAcc, tokenFromSystemSC, esdtNFTTokenKey)
		if err != nil {
			return err
		}
	}

	err = e.accounts.SaveAccount(systemAcc)
	if err != nil {
		return err
	}

	return nil
}

func (e *esdtDeleteMetaData) getESDTDigitalTokenDataFromSystemAccount(
	systemAcc vmcommon.UserAccountHandler,
	esdtNFTTokenKey []byte,
) (*esdt.ESDigitalToken, error) {
	marshaledData, _, err := systemAcc.AccountDataHandler().RetrieveValue(esdtNFTTokenKey)
	if err != nil || len(marshaledData) == 0 {
		return nil, nil
	}

	esdtData := &esdt.ESDigitalToken{}
	err = e.marshaller.Unmarshal(esdtData, marshaledData)
	if err != nil {
		return nil, err
	}

	return esdtData, nil
}

func (e *esdtDeleteMetaData) getSystemAccount() (vmcommon.UserAccountHandler, error) {
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

func (e *esdtDeleteMetaData) marshalAndSaveData(
	systemAcc vmcommon.UserAccountHandler,
	esdtData *esdt.ESDigitalToken,
	esdtNFTTokenKey []byte,
) error {
	marshaledData, err := e.marshaller.Marshal(esdtData)
	if err != nil {
		return err
	}

	err = systemAcc.AccountDataHandler().SaveKeyValue(esdtNFTTokenKey, marshaledData)
	if err != nil {
		return err
	}

	return nil
}

// IsInterfaceNil returns true if underlying object is nil
func (e *esdtDeleteMetaData) IsInterfaceNil() bool {
	return e == nil
}
