package builtInFunctions

import (
	"encoding/hex"
	"math/big"

	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

func addESDTEntryInVMOutput(vmOutput *vmcommon.VMOutput, identifier []byte, tokenID []byte, nonce uint64, value *big.Int, args ...[]byte) {
	entry := newEntryForESDT(identifier, tokenID, nonce, value, args...)

	if vmOutput.Logs == nil {
		vmOutput.Logs = make([]*vmcommon.LogEntry, 0, 1)
	}

	vmOutput.Logs = append(vmOutput.Logs, entry)
}

func newEntryForESDT(identifier, tokenID []byte, nonce uint64, value *big.Int, args ...[]byte) *vmcommon.LogEntry {
	tokenIdentifier := tokenID
	if nonce != 0 {
		nonceBig := big.NewInt(0).SetUint64(nonce)
		hexEncodedNonce := hex.EncodeToString(nonceBig.Bytes())

		tokenIdentifier = append(tokenIdentifier, []byte("-")...)
		tokenIdentifier = append(tokenIdentifier, []byte(hexEncodedNonce)...)
	}

	logEntry := &vmcommon.LogEntry{
		Identifier: identifier,
		Topics:     [][]byte{tokenIdentifier, value.Bytes()},
	}

	if len(args) > 0 {
		logEntry.Address = args[0]
	}

	if len(args) > 1 {
		logEntry.Topics = append(logEntry.Topics, args[1:]...)
	}

	return logEntry
}
