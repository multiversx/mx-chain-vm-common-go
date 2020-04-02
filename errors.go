package vmcommon

import "errors"

// ErrTokenizeFailed signals that data splitting into arguments and code failed
var ErrTokenizeFailed = errors.New("data splitting into arguments and code/function failed")

// ErrInvalidDeployArguments signals invalid deploy arguments
var ErrInvalidDeployArguments = errors.New("invalid deploy arguments")

// ErrNilFunction signals that the function name from transaction data is nil
var ErrNilFunction = errors.New("smart contract function is nil")

// ErrNilArguments signals that arguments from transactions data is nil
var ErrNilArguments = errors.New("smart contract arguments are nil")

// ErrInvalidDataString signals that the transaction data string could not be split evenly
var ErrInvalidDataString = errors.New("transaction data string is unevenly split")

// ErrInvalidVMType signals an invalid VMType
var ErrInvalidVMType = errors.New("invalid vm type")
