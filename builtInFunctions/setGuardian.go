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

var logAccountFreezer = logger.GetOrCreate("systemSmartContracts/setGuardian")

// TODO:
// 1. Add builtin function
// 2. Move Guardian structs to elrond-go-core

// Key prefixes
const (
	SetGuardianKeyIdentifier = "guardians"
)

// Functions
const (
	BuiltInFunctionSetGuardian = "BuiltInFunctionSetGuardian"
)

type Guardian struct {
	Address         []byte
	ActivationEpoch uint32
}

type Guardians struct {
	Data []*Guardian
}

type BlockChainEpochHook interface {
	CurrentEpoch() uint32
	IsInterfaceNil() bool
}

type SetGuardianArgs struct {
	GasCost                  vmcommon.GasCost
	Marshaller               marshal.Marshalizer
	BlockChainHook           BlockChainEpochHook
	PubKeyConverter          core.PubkeyConverter
	GuardianActivationEpochs uint32
	SetGuardianEnableEpoch   uint32
	EpochNotifier            vmcommon.EpochNotifier
}

type setGuardian struct {
	gasCost                  vmcommon.GasCost
	marshaller               marshal.Marshalizer
	blockchainHook           BlockChainEpochHook
	pubKeyConverter          core.PubkeyConverter
	guardianActivationEpochs uint32
	mutExecution             sync.RWMutex

	setGuardianEnableEpoch uint32
	flagEnabled            atomic.Flag
	keyPrefix              []byte
}

func NewSetGuardianFunc(args SetGuardianArgs) (*setGuardian, error) {
	if check.IfNil(args.Marshaller) {
		return nil, core.ErrNilMarshalizer
	}
	if check.IfNil(args.BlockChainHook) {
		return nil, ErrNilBlockHeader // TODO: NEW ERROR
	}
	if check.IfNil(args.PubKeyConverter) {
		return nil, nil // TODO: Error
	}

	setGuardianFunc := &setGuardian{
		gasCost:                  args.GasCost,
		marshaller:               args.Marshaller,
		blockchainHook:           args.BlockChainHook,
		pubKeyConverter:          args.PubKeyConverter,
		guardianActivationEpochs: args.GuardianActivationEpochs,
		setGuardianEnableEpoch:   args.SetGuardianEnableEpoch,
		mutExecution:             sync.RWMutex{},
		keyPrefix:                []byte(core.ElrondProtectedKeyPrefix + SetGuardianKeyIdentifier), // TODO: use this instead of func
	}
	logAccountFreezer.Debug("set guardian enable epoch", setGuardianFunc.setGuardianEnableEpoch)
	args.EpochNotifier.RegisterNotifyHandler(setGuardianFunc)

	return setGuardianFunc, nil
}

// todo; check if guardian is already stored?

// 1. User does NOT have any guardian => set guardian
// 2. User has ONE guardian pending => does not set, wait until first one is set
// 3. User has ONE guardian enabled => add it
// 4. User has TWO guardians. FIRST is enabled, SECOND is pending => change pending with current one / does nothing until it is set
// 5. User has TWO guardians. FIRST is enabled, SECOND is enabled => replace oldest one

func (sg *setGuardian) ProcessBuiltinFunction(
	senderAccount, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	sg.mutExecution.RLock()
	defer sg.mutExecution.RUnlock()

	if vmInput == nil {
		return nil, errors.New("nil arguments")
	}
	if !sg.flagEnabled.IsSet() {
		return nil, errors.New(fmt.Sprintf("account freezer not enabled yet, enable epoch: %d", sg.setGuardianEnableEpoch))
	}

	if !isZero(vmInput.CallValue) {
		return nil, ErrBuiltInFunctionCalledWithValue
	}
	if len(vmInput.Arguments) != 1 {
		return nil, errors.New(fmt.Sprintf("invalid number of arguments, expected 1, got %d ", len(vmInput.Arguments)))
	}
	if vmInput.GasProvided < sg.gasCost.BuiltInCost.SetGuardian {
		return nil, ErrNotEnoughGas
	}
	if !sg.isAddressValid(vmInput.Arguments[0]) {
		return nil, errors.New("invalid address")
	}
	if bytes.Equal(vmInput.CallerAddr, vmInput.Arguments[0]) {
		return nil, errors.New("cannot set own address as guardian")
	}

	guardians, err := sg.guardians(senderAccount)
	if err != nil {
		return nil, err
	}

	switch len(guardians.Data) {
	case 0:
		// Case 1
		return sg.tryAddGuardian(senderAccount, vmInput.Arguments[0], guardians)
	case 1:
		// Case 2
		if sg.pending(guardians.Data[0]) {
			return nil, errors.New(fmt.Sprintf("owner already has one guardian pending: %s",
				sg.pubKeyConverter.Encode(guardians.Data[0].Address)))
		}
		// Case 3
		return sg.tryAddGuardian(senderAccount, vmInput.Arguments[0], guardians)
	case 2:
		// Case 4
		if sg.pending(guardians.Data[1]) {
			return nil, errors.New(fmt.Sprintf("owner already has one guardian pending: %s",
				sg.pubKeyConverter.Encode(guardians.Data[1].Address)))
		}
		// Case 5
		guardians.Data = guardians.Data[1:] // remove oldest guardian
		return sg.tryAddGuardian(senderAccount, vmInput.Arguments[0], guardians)
	default:
		return nil, errors.New("invalid")
	}
}

func (sg *setGuardian) tryAddGuardian(account vmcommon.UserAccountHandler, guardianAddress []byte, guardians Guardians) (*vmcommon.VMOutput, error) {
	err := sg.addGuardian(account, guardianAddress, guardians)
	if err != nil {
		return nil, err
	}
	return &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}, nil
}

func (sg *setGuardian) pending(guardian *Guardian) bool {
	currEpoch := sg.blockchainHook.CurrentEpoch()
	remaining := absDiff(currEpoch, guardian.ActivationEpoch) // any edge case here for which we should use abs?
	return remaining < sg.guardianActivationEpochs
}

func absDiff(a, b uint32) uint32 {
	if a < b {
		return b - a
	}
	return a - b
}

func (sg *setGuardian) addGuardian(account vmcommon.UserAccountHandler, guardianAddress []byte, guardians Guardians) error {
	guardian := &Guardian{
		Address:         guardianAddress,
		ActivationEpoch: sg.blockchainHook.CurrentEpoch() + sg.guardianActivationEpochs,
	}

	guardians.Data = append(guardians.Data, guardian)
	marshalledData, err := sg.marshaller.Marshal(guardians)
	if err != nil {
		return err
	}

	return account.AccountDataHandler().SaveKeyValue(sg.keyPrefix, marshalledData)
}

func (sg *setGuardian) guardians(account vmcommon.UserAccountHandler) (Guardians, error) {
	marshalledData, err := account.AccountDataHandler().RetrieveValue(sg.keyPrefix)
	if err != nil {
		return Guardians{}, err
	}

	// Fine, address has no guardian set
	if len(marshalledData) == 0 {
		return Guardians{}, nil
	}

	guardians := Guardians{}
	err = sg.marshaller.Unmarshal(&guardians, marshalledData)
	if err != nil {
		return Guardians{}, err
	}

	return guardians, nil
}

func isZero(n *big.Int) bool {
	return len(n.Bits()) == 0
}

// TODO: Move this to common  + remove from esdt.go
func (sg *setGuardian) isAddressValid(addressBytes []byte) bool {
	isLengthOk := len(addressBytes) == sg.pubKeyConverter.Len()
	if !isLengthOk {
		return false
	}

	encodedAddress := sg.pubKeyConverter.Encode(addressBytes)

	return encodedAddress != ""
}

func (sg *setGuardian) EpochConfirmed(epoch uint32, _ uint64) {
	sg.flagEnabled.SetValue(epoch >= sg.setGuardianEnableEpoch)
	log.Debug("account freezer", "enabled", sg.flagEnabled.IsSet())
}

func (sg *setGuardian) CanUseContract() bool {
	return false
}

func (sg *setGuardian) SetNewGasCost(gasCost vmcommon.GasCost) {
	sg.mutExecution.Lock()
	sg.gasCost = gasCost
	sg.mutExecution.Unlock()
}

func (sg *setGuardian) IsInterfaceNil() bool {
	return sg == nil
}
