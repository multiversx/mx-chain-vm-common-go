package builtInFunctions

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/atomic"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	logger "github.com/ElrondNetwork/elrond-go-logger"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

var logAccountFreezer = logger.GetOrCreate("systemSmartContracts/freezeAccount")

// TODO:
// 1. Add builtin function
// 2. Move Guardian structs to elrond-go-core

// Key prefixes
const (
	GuardiansKey = "guardians"
)

// Functions
const (
	setGuardian = "setGuardian"
)

type Guardian struct {
	Address         []byte
	ActivationEpoch uint32
}

type Guardians struct {
	Data []*Guardian
}

type ArgsFreezeAccountSC struct {
	GasCost                  vmcommon.GasCost
	Marshaller               marshal.Marshalizer
	BlockChainHook           vmcommon.BlockchainHook
	PubKeyConverter          core.PubkeyConverter
	GuardianActivationEpochs uint32
	SetGuardianEnableEpoch   uint32
	EpochNotifier            vmcommon.EpochNotifier
}

type freezeAccount struct {
	gasCost                  vmcommon.GasCost
	marshaller               marshal.Marshalizer
	blockchainHook           vmcommon.BlockchainHook
	pubKeyConverter          core.PubkeyConverter
	guardianActivationEpochs uint32
	mutExecution             sync.RWMutex

	setGuardianEnableEpoch uint32
	flagEnabled            atomic.Flag
}

func NewFreezeAccountSmartContract(args ArgsFreezeAccountSC) (*freezeAccount, error) {
	if check.IfNil(args.Marshaller) {
		return nil, core.ErrNilMarshalizer
	}
	if check.IfNil(args.BlockChainHook) {
		return nil, ErrNilBlockHeader // TODO: NEW ERROR
	}
	if check.IfNil(args.PubKeyConverter) {
		return nil, nil // TODO: Error
	}

	accountFreezer := &freezeAccount{
		gasCost:                  args.GasCost,
		marshaller:               args.Marshaller,
		blockchainHook:           args.BlockChainHook,
		pubKeyConverter:          args.PubKeyConverter,
		guardianActivationEpochs: args.GuardianActivationEpochs,
		setGuardianEnableEpoch:   args.SetGuardianEnableEpoch,
		mutExecution:             sync.RWMutex{},
	}
	logAccountFreezer.Debug("account freezer enable epoch", accountFreezer.setGuardianEnableEpoch)
	args.EpochNotifier.RegisterNotifyHandler(accountFreezer)

	return accountFreezer, nil
}

// todo; check if guardian is already stored?

// 1. User does NOT have any guardian => set guardian
// 2. User has ONE guardian pending => does not set, wait until first one is set
// 3. User has ONE guardian enabled => add it
// 4. User has TWO guardians. FIRST is enabled, SECOND is pending => change pending with current one / does nothing until it is set
// 5. User has TWO guardians. FIRST is enabled, SECOND is enabled => replace oldest one

func (fa *freezeAccount) ProcessBuiltinFunction(
	acntSnd, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	fa.mutExecution.RLock()
	defer fa.mutExecution.RUnlock()

	if vmInput == nil {
		return nil, errors.New("nil arguments")
	}
	if !fa.flagEnabled.IsSet() {
		return nil, errors.New(fmt.Sprintf("account freezer not enabled yet, enable epoch: %d", fa.setGuardianEnableEpoch))
	}

	if !isZero(vmInput.CallValue) {
		return nil, errors.New(fmt.Sprintf("expected value must be zero, got %v", vmInput.CallValue))
	}
	if len(vmInput.Arguments) != 1 {
		return nil, errors.New(fmt.Sprintf("invalid number of arguments, expected 1, got %d ", len(vmInput.Arguments)))
	}
	if vmInput.GasProvided < fa.gasCost.BuiltInCost.SetGuardian {
		return nil, ErrNotEnoughGas
	}
	if !fa.isAddressValid(vmInput.Arguments[0]) {
		return nil, errors.New("invalid address")
	}
	if bytes.Equal(vmInput.CallerAddr, vmInput.Arguments[0]) {
		return nil, errors.New("cannot set own address as guardian")
	}

	guardians, err := fa.guardians(vmInput.CallerAddr)
	if err != nil {
		return nil, err
	}

	switch len(guardians.Data) {
	case 0:
		// Case 1
		return fa.tryAddGuardian(vmInput.CallerAddr, vmInput.Arguments[0], guardians)
	case 1:
		// Case 2
		if fa.pending(guardians.Data[0]) {
			return nil, errors.New(fmt.Sprintf("owner already has one guardian pending: %s",
				fa.pubKeyConverter.Encode(guardians.Data[0].Address)))
		}
		// Case 3
		return fa.tryAddGuardian(vmInput.CallerAddr, vmInput.Arguments[0], guardians)
	case 2:
		// Case 4
		if fa.pending(guardians.Data[1]) {
			return nil, errors.New(fmt.Sprintf("owner already has one guardian pending: %s",
				fa.pubKeyConverter.Encode(guardians.Data[1].Address)))
		}
		// Case 5
		guardians.Data = guardians.Data[1:] // remove oldest guardian
		return fa.tryAddGuardian(vmInput.CallerAddr, vmInput.Arguments[0], guardians)
	default:
		return nil, errors.New("invalid")
	}
}

func (fa *freezeAccount) tryAddGuardian(address []byte, guardianAddress []byte, guardians Guardians) (*vmcommon.VMOutput, error) {
	err := fa.addGuardian(address, guardianAddress, guardians)
	if err != nil {
		return nil, err
	}
	return &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}, nil
}

func (fa *freezeAccount) pending(guardian *Guardian) bool {
	currEpoch := fa.blockchainHook.CurrentEpoch()
	remaining := absDiff(currEpoch, guardian.ActivationEpoch) // any edge case here for which we should use abs?
	return remaining < fa.guardianActivationEpochs
}

func absDiff(a, b uint32) uint32 {
	if a < b {
		return b - a
	}
	return a - b
}

func (fa *freezeAccount) addGuardian(address []byte, guardianAddress []byte, guardians Guardians) error {
	guardian := &Guardian{
		Address:         guardianAddress,
		ActivationEpoch: fa.blockchainHook.CurrentEpoch() + fa.guardianActivationEpochs,
	}

	guardians.Data = append(guardians.Data, guardian)
	marshalledData, err := fa.marshaller.Marshal(guardians)
	if err != nil {
		return err
	}

	account, err := fa.blockchainHook.GetUserAccount(address)
	if err != nil {
		return err
	}

	key := calcProtectedPrefixedKey(GuardiansKey)
	return account.AccountDataHandler().SaveKeyValue(key, marshalledData)
}

func (fa *freezeAccount) guardians(address []byte) (Guardians, error) {
	account, err := fa.blockchainHook.GetUserAccount(address)
	if err != nil {
		return Guardians{}, err
	}

	key := calcProtectedPrefixedKey(GuardiansKey)
	marshalledData, err := account.AccountDataHandler().RetrieveValue(key)
	if err != nil {
		return Guardians{}, err
	}

	// Fine, address has no guardian set
	if len(marshalledData) == 0 {
		return Guardians{}, nil
	}

	guardians := Guardians{}
	err = fa.marshaller.Unmarshal(&guardians, marshalledData)
	if err != nil {
		return Guardians{}, err
	}

	return guardians, nil
}

func calcProtectedPrefixedKey(key string) []byte {
	return append([]byte(core.ElrondProtectedKeyPrefix), []byte(key)...)
}

func isZero(n *big.Int) bool {
	return len(n.Bits()) == 0
}

// TODO: Move this to common  + remove from esdt.go
func (fa *freezeAccount) isAddressValid(addressBytes []byte) bool {
	isLengthOk := len(addressBytes) == fa.pubKeyConverter.Len()
	if !isLengthOk {
		return false
	}

	encodedAddress := fa.pubKeyConverter.Encode(addressBytes)

	return encodedAddress != ""
}

func (fa *freezeAccount) EpochConfirmed(epoch uint32, _ uint64) {
	fa.flagEnabled.SetValue(epoch >= fa.setGuardianEnableEpoch)
	log.Debug("account freezer", "enabled", fa.flagEnabled.IsSet())
}

func (fa *freezeAccount) CanUseContract() bool {
	return false
}

func (fa *freezeAccount) SetNewGasCost(gasCost vmcommon.GasCost) {
	fa.mutExecution.Lock()
	fa.gasCost = gasCost
	fa.mutExecution.Unlock()
}

func (fa *freezeAccount) IsInterfaceNil() bool {
	return fa == nil
}
