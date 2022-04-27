package mock

import (
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

type GuardedAccountHandlerStub struct {
	GetActiveGuardianCalled func(handler vmcommon.UserAccountHandler) ([]byte, error)
	SetGuardianCalled       func(uah vmcommon.UserAccountHandler, guardianAddress []byte) error
}

func (gahs *GuardedAccountHandlerStub) GetActiveGuardian(handler vmcommon.UserAccountHandler) ([]byte, error) {
	if gahs.GetActiveGuardianCalled != nil {
		return gahs.GetActiveGuardianCalled(handler)
	}
	return nil, nil
}

func (gahs *GuardedAccountHandlerStub) SetGuardian(uah vmcommon.UserAccountHandler, guardianAddress []byte) error {
	if gahs.SetGuardianCalled != nil {
		return gahs.SetGuardianCalled(uah, guardianAddress)
	}
	return nil
}

func (gahs *GuardedAccountHandlerStub) IsInterfaceNil() bool {
	return gahs == nil
}
