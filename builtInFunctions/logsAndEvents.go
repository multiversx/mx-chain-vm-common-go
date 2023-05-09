package builtInFunctions

import (
	"bytes"
	"fmt"
	"math/big"
	"strconv"

	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

const (
	esdtIdentifierSeparator  = "-"
	esdtRandomSequenceLength = 6
)

func addESDTEntryForTransferInVMOutput(
	vmInput *vmcommon.ContractCallInput,
	vmOutput *vmcommon.VMOutput,
	identifier []byte,
	destination []byte,
	tokenID []byte,
	nonce uint64,
	value *big.Int,
) {
	nonceBig := big.NewInt(0).SetUint64(nonce)

	logEntry := &vmcommon.LogEntry{
		Identifier: identifier,
		Address:    destination,
		Topics:     [][]byte{vmInput.CallerAddr, tokenID, nonceBig.Bytes(), value.Bytes()},
		Data:       vmcommon.FormatLogDataForCall("", vmInput.Function, vmInput.Arguments),
	}

	if vmOutput.Logs == nil {
		vmOutput.Logs = make([]*vmcommon.LogEntry, 0, 1)
	}

	vmOutput.Logs = append(vmOutput.Logs, logEntry)
}

func addESDTEntryInVMOutput(vmOutput *vmcommon.VMOutput, identifier []byte, tokenID []byte, nonce uint64, value *big.Int, args ...[]byte) {
	entry := newEntryForESDT(identifier, tokenID, nonce, value, args...)

	if vmOutput.Logs == nil {
		vmOutput.Logs = make([]*vmcommon.LogEntry, 0, 1)
	}

	vmOutput.Logs = append(vmOutput.Logs, entry)
}

func newEntryForESDT(identifier, tokenID []byte, nonce uint64, value *big.Int, args ...[]byte) *vmcommon.LogEntry {
	nonceBig := big.NewInt(0).SetUint64(nonce)

	logEntry := &vmcommon.LogEntry{
		Identifier: identifier,
		Topics:     [][]byte{tokenID, nonceBig.Bytes(), value.Bytes()},
	}

	if len(args) > 0 {
		logEntry.Address = args[0]
	}

	if len(args) > 1 {
		logEntry.Topics = append(logEntry.Topics, args[1:]...)
	}

	return logEntry
}

func extractTokenIdentifierAndNonceESDTWipe(args []byte) ([]byte, uint64) {
	argsSplit := bytes.Split(args, []byte(esdtIdentifierSeparator))
	if len(argsSplit) < 2 {
		return args, 0
	}

	if len(argsSplit[1]) <= esdtRandomSequenceLength {
		return args, 0
	}

	identifier := []byte(fmt.Sprintf("%s-%s", argsSplit[0], argsSplit[1][:esdtRandomSequenceLength]))
	nonce := big.NewInt(0).SetBytes(argsSplit[1][esdtRandomSequenceLength:])

	return identifier, nonce.Uint64()
}

func boolToSlice(b bool) []byte {
	return []byte(strconv.FormatBool(b))
}
