package builtInFunctions

import (
	"bytes"
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
	GuardianKeyIdentifier = "guardians"
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

// BlockChainEpochHook is a light-weight blockchain hook,
// which can be queried to provide the current epoch
type BlockChainEpochHook interface {
	CurrentEpoch() uint32
	IsInterfaceNil() bool
}

// SetGuardianArgs is a struct placeholder for all necessary args
// to create a NewSetGuardianFunc
type SetGuardianArgs struct {
	Marshaller      marshal.Marshalizer
	PubKeyConverter core.PubkeyConverter
	EpochNotifier   vmcommon.EpochNotifier
	BlockChainHook  BlockChainEpochHook

	FuncGasCost              uint64
	GuardianActivationEpochs uint32
	SetGuardianEnableEpoch   uint32
}

type setGuardian struct {
	*baseEnabled
	marshaller      marshal.Marshalizer
	blockchainHook  BlockChainEpochHook
	pubKeyConverter core.PubkeyConverter

	mutExecution             sync.RWMutex
	guardianActivationEpochs uint32
	funcGasCost              uint64
	keyPrefix                []byte
}

// NewSetGuardianFunc will instantiate a new set guardian built-in function
func NewSetGuardianFunc(args SetGuardianArgs) (*setGuardian, error) {
	if check.IfNil(args.Marshaller) {
		return nil, ErrNilMarshaller
	}
	if check.IfNil(args.BlockChainHook) {
		return nil, ErrNilBlockChainHook
	}
	if check.IfNil(args.PubKeyConverter) {
		return nil, ErrNilPubKeyConverter
	}
	if check.IfNil(args.EpochNotifier) {
		return nil, ErrNilEpochNotifier
	}

	setGuardianFunc := &setGuardian{
		funcGasCost:              args.FuncGasCost,
		marshaller:               args.Marshaller,
		blockchainHook:           args.BlockChainHook,
		pubKeyConverter:          args.PubKeyConverter,
		guardianActivationEpochs: args.GuardianActivationEpochs,
		mutExecution:             sync.RWMutex{},
		keyPrefix:                []byte(core.ElrondProtectedKeyPrefix + GuardianKeyIdentifier),
	}
	setGuardianFunc.baseEnabled = &baseEnabled{
		function:        BuiltInFunctionSetGuardian,
		activationEpoch: args.SetGuardianEnableEpoch,
		flagActivated:   atomic.Flag{},
	}

	logAccountFreezer.Debug("set guardian enable epoch", args.SetGuardianEnableEpoch)
	args.EpochNotifier.RegisterNotifyHandler(setGuardianFunc)

	return setGuardianFunc, nil
}

// ProcessBuiltinFunction will process the set guardian built-in function call
// Currently, the following cases are treated:
// Case 1. User does NOT have any guardian => set guardian
// Case 2. User has ONE guardian pending => do not set new guardian, wait until first one is set
// Case 3. User has ONE guardian enabled => set guardian
// Case 4. User has TWO guardians. FIRST is enabled, SECOND is pending => does not set, wait until second one is set
// Case 5. User has TWO guardians. FIRST is enabled, SECOND is enabled => replace oldest one + set new one as pending
func (sg *setGuardian) ProcessBuiltinFunction(
	senderAccount, receiverAccount vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	sg.mutExecution.RLock()
	defer sg.mutExecution.RUnlock()

	err := sg.checkArguments(senderAccount, receiverAccount, vmInput)
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
		return &vmcommon.VMOutput{ReturnCode: vmcommon.ExecutionFailed}, nil
	}
}

func (sg *setGuardian) checkArguments(
	senderAccount,
	receiverAccount vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) error {
	if check.IfNil(senderAccount) {
		return fmt.Errorf("%w for sender", ErrNilUserAccount)
	}
	if check.IfNil(receiverAccount) {
		return fmt.Errorf("%w for receiver", ErrNilUserAccount)
	}
	if vmInput == nil {
		return ErrNilVmInput
	}
	if !(bytes.Equal(senderAccount.AddressBytes(), receiverAccount.AddressBytes()) &&
		bytes.Equal(senderAccount.AddressBytes(), vmInput.CallerAddr)) {
		return ErrOperationNotPermitted
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

func (sg *setGuardian) guardians(account vmcommon.UserAccountHandler) (*Guardians, error) {
	marshalledData, err := account.AccountDataHandler().RetrieveValue(sg.keyPrefix)
	if err != nil {
		return nil, err
	}

	// Fine, account has no guardian set
	if len(marshalledData) == 0 {
		return &Guardians{Data: make([]*Guardian, 0)}, nil
	}

	guardians := &Guardians{}
	err = sg.marshaller.Unmarshal(guardians, marshalledData)
	return guardians, err
}

func (sg *setGuardian) contains(guardians *Guardians, guardianAddress []byte) bool {
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
	guardians *Guardians,
	gasProvided uint64,
) (*vmcommon.VMOutput, error) {
	err := sg.addGuardian(account, guardianAddress, guardians)
	if err != nil {
		return nil, err
	}
	return &vmcommon.VMOutput{ReturnCode: vmcommon.Ok, GasRemaining: gasProvided - sg.funcGasCost}, nil
}

func (sg *setGuardian) addGuardian(account vmcommon.UserAccountHandler, guardianAddress []byte, guardians *Guardians) error {
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

// SetNewGasConfig is called whenever gas cost is changed
func (sg *setGuardian) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	sg.mutExecution.Lock()
	sg.funcGasCost = gasCost.BuiltInCost.SetGuardian
	sg.mutExecution.Unlock()
}
