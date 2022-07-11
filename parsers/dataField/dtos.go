package datafield

// ResponseParseData is the response with results after the data field was parsed
type ResponseParseData struct {
	// Operation field is used to store the name of the operation that the transaction will try to do
	// an example of operation is `transfer` or `ESDTTransfer etc
	Operation string
	// Function field is used to store the function name that the transaction will try to call from a smart contract
	Function         string
	ESDTValues       []string
	Tokens           []string
	Receivers        [][]byte
	ReceiversShardID []uint32
	IsRelayed        bool
}

func NewResponseParseDataAsRelayed() *ResponseParseData {
	return &ResponseParseData{
		IsRelayed: true,
	}
}
