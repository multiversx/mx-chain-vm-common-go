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
	FuncGasCost              uint64
	Marshaller               marshal.Marshalizer
	BlockChainHook           BlockChainEpochHook
	PubKeyConverter          core.PubkeyConverter
	GuardianActivationEpochs uint32
	SetGuardianEnableEpoch   uint32
	EpochNotifier            vmcommon.EpochNotifier
}

type setGuardian struct {
	funcGasCost              uint64
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
		funcGasCost:              args.FuncGasCost,
		marshaller:               args.Marshaller,
		blockchainHook:           args.BlockChainHook,
		pubKeyConverter:          args.PubKeyConverter,
		guardianActivationEpochs: args.GuardianActivationEpochs,
		setGuardianEnableEpoch:   args.SetGuardianEnableEpoch,
		mutExecution:             sync.RWMutex{},
		keyPrefix:                []byte(core.ElrondProtectedKeyPrefix + SetGuardianKeyIdentifier),
	}
	logAccountFreezer.Debug("set guardian enable epoch", setGuardianFunc.setGuardianEnableEpoch)
	args.EpochNotifier.RegisterNotifyHandler(setGuardianFunc)

	return setGuardianFunc, nil
}

// Case 1. User does NOT have any guardian => set guardian
// Case 2. User has ONE guardian pending => does not set, wait until first one is set
// Case 3. User has ONE guardian enabled => add it
// Case 4. User has TWO guardians. FIRST is enabled, SECOND is pending => does not set, wait until second one is set
// Case 5. User has TWO guardians. FIRST is enabled, SECOND is enabled => replace oldest one + set new one as pending

func (sg *setGuardian) ProcessBuiltinFunction(
	senderAccount, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	sg.mutExecution.RLock()
	defer sg.mutExecution.RUnlock()

	if !sg.flagEnabled.IsSet() {
		return nil, fmt.Errorf("%w, enable epoch is: %d", ErrSetGuardianNotEnabled, sg.setGuardianEnableEpoch)
	}
	err := sg.checkArguments(vmInput)
	if err != nil {
		return nil, err
	}

	guardians, err := sg.guardians(senderAccount)
	if err != nil {
		return nil, err
	}
	if sg.contains(guardians, vmInput.Arguments[0]) {
		return nil, ErrGuardianAlreadyExists
	}

	switch len(guardians.Data) {
	case 0:
		// Case 1
		return sg.tryAddGuardian(senderAccount, vmInput.Arguments[0], guardians, vmInput.GasProvided)
	case 1:
		// Case 2
		if sg.pending(guardians.Data[0]) {
			return nil, fmt.Errorf("%w: %s", ErrOwnerAlreadyHasOneGuardianPending, sg.pubKeyConverter.Encode(guardians.Data[0].Address))
		}
		// Case 3
		return sg.tryAddGuardian(senderAccount, vmInput.Arguments[0], guardians, vmInput.GasProvided)
	case 2:
		// Case 4
		if sg.pending(guardians.Data[1]) {
			return nil, fmt.Errorf("%w: %s", ErrOwnerAlreadyHasOneGuardianPending, sg.pubKeyConverter.Encode(guardians.Data[1].Address))
		}
		// Case 5
		guardians.Data = guardians.Data[1:] // remove oldest guardian
		return sg.tryAddGuardian(senderAccount, vmInput.Arguments[0], guardians, vmInput.GasProvided)
	default:
		return nil, errors.New("invalid")
	}
}

func (sg *setGuardian) checkArguments(vmInput *vmcommon.ContractCallInput) error {
	if vmInput == nil {
		return ErrNilVmInput
	}
	if vmInput.CallValue == nil {
		return ErrNilValue
	}
	if !isZero(vmInput.CallValue) {
		return ErrBuiltInFunctionCalledWithValue
	}
	if len(vmInput.Arguments) != 1 {
		return fmt.Errorf("%w, expected 1, got %d ", ErrInvalidNumberOfArguments, len(vmInput.Arguments))
	}
	if !sg.isAddressValid(vmInput.Arguments[0]) {
		return fmt.Errorf("%w for guardian", ErrInvalidAddress)
	}
	if bytes.Equal(vmInput.CallerAddr, vmInput.Arguments[0]) {
		return ErrCannotOwnAddressAsGuardian
	}
	if vmInput.GasProvided < sg.funcGasCost {
		return ErrNotEnoughGas
	}

	return nil
}

func isZero(n *big.Int) bool {
	return len(n.Bits()) == 0
}

func (sg *setGuardian) isAddressValid(addressBytes []byte) bool {
	isLengthOk := len(addressBytes) == sg.pubKeyConverter.Len()
	if !isLengthOk {
		return false
	}

	encodedAddress := sg.pubKeyConverter.Encode(addressBytes)

	return encodedAddress != ""
}

func (sg *setGuardian) guardians(account vmcommon.UserAccountHandler) (Guardians, error) {
	marshalledData, err := account.AccountDataHandler().RetrieveValue(sg.keyPrefix)
	if err != nil {
		return Guardians{}, err
	}

	// Fine, account has no guardian set
	if len(marshalledData) == 0 {
		return Guardians{}, nil
	}

	guardians := Guardians{}
	err = sg.marshaller.Unmarshal(&guardians, marshalledData)
	return guardians, err
}

func (sg *setGuardian) contains(guardians Guardians, guardianAddress []byte) bool {
	for _, guardian := range guardians.Data {
		if bytes.Equal(guardian.Address, guardianAddress) {
			return true
		}
	}
	return false
}

func (sg *setGuardian) tryAddGuardian(
	account vmcommon.UserAccountHandler,
	guardianAddress []byte,
	guardians Guardians,
	gasProvided uint64,
) (*vmcommon.VMOutput, error) {
	err := sg.addGuardian(account, guardianAddress, guardians)
	if err != nil {
		return nil, err
	}
	return &vmcommon.VMOutput{ReturnCode: vmcommon.Ok, GasRemaining: gasProvided - sg.funcGasCost}, nil
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

func (sg *setGuardian) pending(guardian *Guardian) bool {
	return guardian.ActivationEpoch > sg.blockchainHook.CurrentEpoch()
}

func (sg *setGuardian) EpochConfirmed(epoch uint32, _ uint64) {
	sg.flagEnabled.SetValue(epoch >= sg.setGuardianEnableEpoch)
	log.Debug("set guardian", "enabled", sg.flagEnabled.IsSet())
}

func (sg *setGuardian) CanUseContract() bool {
	return false
}

func (sg *setGuardian) SetNewGasCost(gasCost vmcommon.GasCost) {
	sg.mutExecution.Lock()
	sg.funcGasCost = gasCost.BuiltInCost.SetGuardian
	sg.mutExecution.Unlock()
}

func (sg *setGuardian) IsInterfaceNil() bool {
	return sg == nil
}
