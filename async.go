package vmcommon

// AsyncCallStatus represents the different status an async call can have
type AsyncCallStatus uint8

const (
	AsyncCallPending AsyncCallStatus = iota
	AsyncCallResolved
	AsyncCallRejected
)

// AsyncGeneratedCall holds the information abount an async call
type AsyncGeneratedCall struct {
	Status          AsyncCallStatus
	Destination     []byte
	Data            []byte
	GasLimit        uint64
	ValueBytes      []byte
	SuccessCallback string
	ErrorCallback   string
	GasPercentage   int32
}

// AsyncContext is a structure containing a group of async calls and a callback
//  that should be called when all these async calls are resolved
type AsyncContext struct {
	Callback   string
	AsyncCalls []*AsyncGeneratedCall
}

// AsyncInitiator will keep the data about the initiator of an async call
type AsyncInitiator struct {
	CallerAddr []byte
	ReturnData []byte
}

// AsyncContextInfo is the structure resulting after a smart contract call that has initiated
// one or more async calls. It will
type AsyncContextInfo struct {
	AsyncInitiator
	AsyncContextMap map[string]*AsyncContext
}

// GetDestination returns the destination of an async call
func (ac *AsyncGeneratedCall) GetDestination() []byte {
	return ac.Destination
}

// GetData returns the transaction data of the async call
func (ac *AsyncGeneratedCall) GetData() []byte {
	return ac.Data
}

// GetGasLimit returns the gas limit of the current async call
func (ac *AsyncGeneratedCall) GetGasLimit() uint64 {
	return ac.GasLimit
}

// GetValueBytes returns the byte representation of the value of the async call
func (ac *AsyncGeneratedCall) GetValueBytes() []byte {
	return ac.ValueBytes
}

// IsInterfaceNil returns true if there is no value under the interface
func (ac *AsyncGeneratedCall) IsInterfaceNil() bool {
	return ac == nil
}

