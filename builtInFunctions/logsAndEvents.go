package builtInFunctions

import (
	"math/big"

	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

func newEntryForNFT(identifier string, caller []byte, tokenID []byte, nonce uint64) *vmcommon.LogEntry {
	nonceBig := big.NewInt(0).SetUint64(nonce)

	logEntry := &vmcommon.LogEntry{
		Identifier: []byte(identifier),
		Address:    caller,
		Topics:     [][]byte{tokenID, nonceBig.Bytes()},
	}

	return logEntry
}

func addESDTEntryInVMOutput(vmOutput *vmcommon.VMOutput, identifier []byte, tokenID []byte, value *big.Int, args ...[]byte) {
	entry := newEntryForESDT(identifier, tokenID, value, args...)

	if vmOutput.Logs == nil {
		vmOutput.Logs = make([]*vmcommon.LogEntry, 0, 1)
	}

	vmOutput.Logs = append(vmOutput.Logs, entry)
}

func newEntryForESDT(identifier, tokenID []byte, value *big.Int, addresses ...[]byte) *vmcommon.LogEntry {
	logEntry := &vmcommon.LogEntry{
		Identifier: identifier,
		Topics:     [][]byte{tokenID, value.Bytes()},
	}

	if len(addresses) > 0 {
		logEntry.Address = addresses[0]
	}

	if len(addresses) > 1 {
		logEntry.Topics = append(logEntry.Topics, addresses[1])
	}

	return logEntry
}
