package builtInFunctions

import (
	"bytes"
	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/atomic"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data/vm"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

type payableCheck struct {
	checkFunctionArgumentsEnableEpoch uint32
	flagCheckFunctionArgument         atomic.Flag

	fixAsyncCallbackCheckEnableEpoch uint32
	flagAsyncCallbackCheck           atomic.Flag

	payableHandler vmcommon.PayableHandler
}

// NewPayableCheckFunc returns a new component which checks if destination is payableCheck when needed
func NewPayableCheckFunc(
	payable vmcommon.PayableHandler,
	checkFunctionArgumentsEnableEpoch uint32,
	fixAsyncCallbackCheckEnableEpoch uint32,
	epochNotifier vmcommon.EpochNotifier,
) (*payableCheck, error) {
	if check.IfNil(epochNotifier) {
		return nil, ErrNilEpochHandler
	}
	if check.IfNil(payable) {
		return nil, ErrNilPayableHandler
	}

	p := &payableCheck{
		payableHandler:                    payable,
		checkFunctionArgumentsEnableEpoch: checkFunctionArgumentsEnableEpoch,
		fixAsyncCallbackCheckEnableEpoch:  fixAsyncCallbackCheckEnableEpoch,
	}

	epochNotifier.RegisterNotifyHandler(p)

	return p, nil
}

func (p *payableCheck) mustVerifyPayable(vmInput *vmcommon.ContractCallInput, minLenArguments int) bool {
	typeToVerify := vm.AsynchronousCall
	if p.flagAsyncCallbackCheck.IsSet() {
		typeToVerify = vm.AsynchronousCallBack
		if vmInput.ReturnCallAfterError {
			return false
		}
	}
	if vmInput.CallType == typeToVerify || vmInput.CallType == vm.ESDTTransferAndExecute {
		return false
	}
	if bytes.Equal(vmInput.CallerAddr, core.ESDTSCAddress) {
		return false
	}
	if len(vmInput.Arguments) > minLenArguments {
		if p.flagCheckFunctionArgument.IsSet() {
			if len(vmInput.Arguments[minLenArguments]) > 0 {
				return false
			}
		} else {
			return false
		}
	}

	return true
}

// CheckPayable returns error if the destination account a non-payable smart contract and there is no sc call after transfer
func (p *payableCheck) CheckPayable(vmInput *vmcommon.ContractCallInput, dstAddress []byte, minLenArguments int) error {
	if !p.mustVerifyPayable(vmInput, minLenArguments) {
		return nil
	}

	isPayable, errIsPayable := p.payableHandler.IsPayable(vmInput.CallerAddr, dstAddress)
	if errIsPayable != nil {
		return errIsPayable
	}
	if !isPayable {
		return ErrAccountNotPayable
	}

	return nil
}

// DetermineIsSCCallAfter returns true if there is a smart contract call after execution
func (p *payableCheck) DetermineIsSCCallAfter(vmInput *vmcommon.ContractCallInput, destAddress []byte, minLenArguments int) bool {
	if len(vmInput.Arguments) <= minLenArguments {
		return false
	}
	if vmInput.ReturnCallAfterError && vmInput.CallType != vm.AsynchronousCallBack {
		return false
	}
	if !vmcommon.IsSmartContractAddress(destAddress) {
		return false
	}
	if p.flagCheckFunctionArgument.IsSet() {
		if len(vmInput.Arguments[minLenArguments]) == 0 {
			return false
		}
	}

	return true
}

// EpochConfirmed is called whenever a new epoch is confirmed
func (p *payableCheck) EpochConfirmed(epoch uint32, _ uint64) {
	p.flagCheckFunctionArgument.SetValue(epoch >= p.checkFunctionArgumentsEnableEpoch)
	log.Debug("ESDTPayable func check function argument", "enabled", p.flagCheckFunctionArgument.IsSet())
	p.flagAsyncCallbackCheck.SetValue(epoch >= p.fixAsyncCallbackCheckEnableEpoch)
	log.Debug("ESDTPayable fix asyncCallBack check", "enabled", p.flagAsyncCallbackCheck.IsSet())

}

// IsInterfaceNil returns true if underlying object is nil
func (p *payableCheck) IsInterfaceNil() bool {
	return p == nil
}
