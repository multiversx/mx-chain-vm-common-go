package vmcommon

import "errors"

// ErrStringSplitFailed signals that data splitting into arguments and code failed
var ErrStringSplitFailed = errors.New("data splitting into arguments and code/function failed")

// ErrNilArguments signals that arguments from transactions data is nil
var ErrNilArguments = errors.New("smart contract arguments are nil")

// ErrNilCode signals that code from transaction data is nil
var ErrNilCode = errors.New("smart contract code is nil")

// ErrNilFunction signals that the function name from transaction data is nil
var ErrNilFunction = errors.New("smart contract function is nil")

// ErrInvalidDataString signals that the transaction data string could not be split evenly
var ErrInvalidDataString = errors.New("transaction data string is unevenly split")
