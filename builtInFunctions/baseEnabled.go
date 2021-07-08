package builtInFunctions

import (
	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/ElrondNetwork/elrond-vm-common/atomic"
)

var log = logger.GetOrCreate("vmCommon/builtInFunctions")

type baseEnabled struct {
	function        string
	activationEpoch uint32
	flagActivated   atomic.Flag
}

// IsActive returns true if function is activated
func (b *baseEnabled) IsActive() bool {
	return b.flagActivated.IsSet()
}

// EpochConfirmed is called whenever a new epoch is confirmed
func (b *baseEnabled) EpochConfirmed(epoch uint32, _ uint64) {
	b.flagActivated.Toggle(epoch >= b.activationEpoch)
	log.Debug("built in function", "name: ", b.function, "enabled", b.flagActivated.IsSet())
}

// IsInterfaceNil returns true if there is no value under the interface
func (b *baseEnabled) IsInterfaceNil() bool {
	return b == nil
}

type baseAlwaysActive struct {
}

// IsActive returns true as this built in function was always active
func (b baseAlwaysActive) IsActive() bool {
	return true
}

// IsInterfaceNil returns true if there is no value under the interface
func (b baseAlwaysActive) IsInterfaceNil() bool {
	return false
}
