package vmcommon

type AsyncCallStatus uint8

const (
	AsyncCallPending AsyncCallStatus = iota
	AsyncCallResolved
	AsyncCallRejected
)

type AsyncCall struct {
	Status          AsyncCallStatus
	Destination     []byte
	Data            []byte
	GasLimit        uint64
	ValueBytes      []byte
	SuccessCallback string
	ErrorCallback   string
	GasPercentage   int32
}

type AsyncContext struct {
	Callback   int32
	AsyncCalls []*AsyncCall
}

type AsyncInitiator struct {
	CallerAddr []byte
	ReturnData []byte
}

type AsyncContextInfo struct {
	AsyncInitiator
	AsyncContextMap map[string]*AsyncContext
}

func (ac *AsyncCall) GetDestination() []byte {
	return ac.Destination
}

func (ac *AsyncCall) GetData() []byte {
	return ac.Data
}

func (ac *AsyncCall) GetGasLimit() uint64 {
	return ac.GasLimit
}

func (ac *AsyncCall) GetValueBytes() []byte {
	return ac.ValueBytes
}

