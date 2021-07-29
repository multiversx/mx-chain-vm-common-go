package builtInFunctions

import (
	"fmt"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/container"
)

var _ vmcommon.BuiltInFunctionContainer = (*functionContainer)(nil)

// functionContainer is an interceptors holder organized by type
type functionContainer struct {
	objects *container.MutexMap
}

// NewBuiltInFunctionContainer will create a new instance of a container
func NewBuiltInFunctionContainer() *functionContainer {
	return &functionContainer{
		objects: container.NewMutexMap(),
	}
}

// Get returns the object stored at a certain key.
// Returns an error if the element does not exist
func (f *functionContainer) Get(key string) (vmcommon.BuiltinFunction, error) {
	value, ok := f.objects.Get(key)
	if !ok {
		return nil, fmt.Errorf("%w in function container for key %v", ErrInvalidContainerKey, key)
	}

	function, ok := value.(vmcommon.BuiltinFunction)
	if !ok {
		return nil, ErrWrongTypeInContainer
	}

	return function, nil
}

// Add will add an object at a given key. Returns
// an error if the element already exists
func (f *functionContainer) Add(key string, function vmcommon.BuiltinFunction) error {
	if check.IfNil(function) {
		return ErrNilContainerElement
	}
	if len(key) == 0 {
		return ErrEmptyFunctionName
	}

	ok := f.objects.Insert(key, function)
	if !ok {
		return ErrContainerKeyAlreadyExists
	}

	return nil
}

// Replace will add (or replace if it already exists) an object at a given key
func (f *functionContainer) Replace(key string, function vmcommon.BuiltinFunction) error {
	if check.IfNil(function) {
		return ErrNilContainerElement
	}
	if len(key) == 0 {
		return ErrEmptyFunctionName
	}

	f.objects.Set(key, function)
	return nil
}

// Remove will remove an object at a given key
func (f *functionContainer) Remove(key string) {
	f.objects.Remove(key)
}

// Len returns the length of the added objects
func (f *functionContainer) Len() int {
	return f.objects.Len()
}

// Keys returns all the keys in the containers
func (f *functionContainer) Keys() map[string]struct{} {
	keys := make(map[string]struct{}, f.Len())

	for _, key := range f.objects.Keys() {
		stringKey, ok := key.(string)
		if !ok {
			continue
		}

		keys[stringKey] = struct{}{}
	}

	return keys
}

// IsInterfaceNil returns true if there is no value under the interface
func (f *functionContainer) IsInterfaceNil() bool {
	return f == nil
}
