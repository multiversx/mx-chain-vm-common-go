package vmcommon

import "errors"

// ErrSubtractionOverflow signals that uint64 subtraction overflowed
var ErrSubtractionOverflow = errors.New("uint64 subtraction overflowed")

// ErrAsyncParams signals that there was an error with the async parameters
var ErrAsyncParams = errors.New("async parameters error")
