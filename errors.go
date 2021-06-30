package vmcommon

import "errors"

// ErrSubtractionOverflow signals that uint64 subtraction overflowed
var ErrSubtractionOverflow = errors.New("uint64 subtraction overflowed")
