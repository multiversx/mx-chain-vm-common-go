package data

import "errors"

// ErrInvalidValue signals that an invalid value has been provided such as NaN to an integer field
var ErrInvalidValue = errors.New("invalid value")
