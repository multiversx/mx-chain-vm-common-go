package builtInFunctions

import (
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

type accountFreezerBase struct {
	marshaller     marshal.Marshalizer
	blockchainHook BlockChainEpochHook
	keyPrefix      []byte
}

func (afb *accountFreezerBase) guardians(account vmcommon.UserAccountHandler) (*Guardians, error) {
	marshalledData, err := account.AccountDataHandler().RetrieveValue(afb.keyPrefix)
	if err != nil {
		return nil, err
	}

	// Fine, account has no guardian set
	if len(marshalledData) == 0 {
		return &Guardians{Data: make([]*Guardian, 0)}, nil
	}

	guardians := &Guardians{}
	err = afb.marshaller.Unmarshal(guardians, marshalledData)
	return guardians, err
}

func (afb *accountFreezerBase) pending(guardian *Guardian) bool {
	return guardian.ActivationEpoch > afb.blockchainHook.CurrentEpoch()
}

func (afb *accountFreezerBase) enabled(guardian *Guardian) bool {
	return !afb.pending(guardian)
}
