package builtInFunctions

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/data/esdt"
	"github.com/ElrondNetwork/elrond-vm-common"
	"github.com/ElrondNetwork/elrond-vm-common/mock"
	"github.com/stretchr/testify/require"
)

func TestNewESDTRolesFunc_NilMarshalizerShouldErr(t *testing.T) {
	t.Parallel()

	esdtRolesF, err := NewESDTRolesFunc(nil, false)

	require.Equal(t, ErrNilMarshalizer, err)
	require.Nil(t, esdtRolesF)
}

func TestEsdtRoles_ProcessBuiltinFunction_NilVMInputShouldErr(t *testing.T) {
	t.Parallel()

	esdtRolesF, _ := NewESDTRolesFunc(nil, false)

	_, err := esdtRolesF.ProcessBuiltinFunction(nil, &mock.UserAccountStub{}, nil)
	require.Equal(t, ErrNilVmInput, err)
}

func TestEsdtRoles_ProcessBuiltinFunction_WrongCalledShouldErr(t *testing.T) {
	t.Parallel()

	esdtRolesF, _ := NewESDTRolesFunc(nil, false)

	_, err := esdtRolesF.ProcessBuiltinFunction(nil, &mock.UserAccountStub{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: []byte{},
			Arguments:  [][]byte{[]byte("1"), []byte("2")},
		},
	})
	require.Equal(t, ErrAddressIsNotESDTSystemSC, err)
}

func TestEsdtRoles_ProcessBuiltinFunction_NilAccountDestShouldErr(t *testing.T) {
	t.Parallel()

	esdtRolesF, _ := NewESDTRolesFunc(nil, false)

	_, err := esdtRolesF.ProcessBuiltinFunction(nil, nil, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: vmcommon.ESDTSCAddress,
			Arguments:  [][]byte{[]byte("1"), []byte("2")},
		},
	})
	require.Equal(t, ErrNilUserAccount, err)
}

func TestEsdtRoles_ProcessBuiltinFunction_GetRolesFailShouldErr(t *testing.T) {
	t.Parallel()

	esdtRolesF, _ := NewESDTRolesFunc(&mock.MarshalizerMock{Fail: true}, false)

	_, err := esdtRolesF.ProcessBuiltinFunction(nil, &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(key []byte) ([]byte, error) {
					return nil, nil
				},
			}
		},
	}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: vmcommon.ESDTSCAddress,
			Arguments:  [][]byte{[]byte("1"), []byte("2")},
		},
	})
	require.Error(t, err)
}

func TestEsdtRoles_ProcessBuiltinFunction_GetRolesFailShouldWorkEvenIfAccntTrieIsNil(t *testing.T) {
	t.Parallel()

	saveKeyWasCalled := false
	esdtRolesF, _ := NewESDTRolesFunc(&mock.MarshalizerMock{}, false)

	_, err := esdtRolesF.ProcessBuiltinFunction(nil, &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, error) {
					return nil, nil
				},
				SaveKeyValueCalled: func(_ []byte, _ []byte) error {
					saveKeyWasCalled = true
					return nil
				},
			}
		},
	}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: vmcommon.ESDTSCAddress,
			Arguments:  [][]byte{[]byte("1"), []byte("2")},
		},
	})
	require.NoError(t, err)
	require.True(t, saveKeyWasCalled)
}

func TestEsdtRoles_ProcessBuiltinFunction_SetRolesShouldWork(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.MarshalizerMock{}
	esdtRolesF, _ := NewESDTRolesFunc(marshalizer, true)

	acc := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(key []byte) ([]byte, error) {
					roles := &esdt.ESDTRoles{}
					return marshalizer.Marshal(roles)
				},
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					roles := &esdt.ESDTRoles{}
					_ = marshalizer.Unmarshal(roles, value)
					require.Equal(t, roles.Roles, [][]byte{[]byte(vmcommon.ESDTRoleLocalMint)})
					return nil
				},
			}
		},
	}
	_, err := esdtRolesF.ProcessBuiltinFunction(nil, acc, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: vmcommon.ESDTSCAddress,
			Arguments:  [][]byte{[]byte("1"), []byte(vmcommon.ESDTRoleLocalMint)},
		},
	})
	require.Nil(t, err)
}

func TestEsdtRoles_ProcessBuiltinFunction_SaveFailedShouldErr(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.MarshalizerMock{}
	esdtRolesF, _ := NewESDTRolesFunc(marshalizer, true)

	localErr := errors.New("local err")
	acc := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(key []byte) ([]byte, error) {
					roles := &esdt.ESDTRoles{}
					return marshalizer.Marshal(roles)
				},
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					return localErr
				},
			}
		},
	}
	_, err := esdtRolesF.ProcessBuiltinFunction(nil, acc, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: vmcommon.ESDTSCAddress,
			Arguments:  [][]byte{[]byte("1"), []byte(vmcommon.ESDTRoleLocalMint)},
		},
	})
	require.Equal(t, localErr, err)
}

func TestEsdtRoles_ProcessBuiltinFunction_UnsetRolesDoesNotExistsShouldWork(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.MarshalizerMock{}
	esdtRolesF, _ := NewESDTRolesFunc(marshalizer, false)

	acc := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(key []byte) ([]byte, error) {
					roles := &esdt.ESDTRoles{}
					return marshalizer.Marshal(roles)
				},
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					roles := &esdt.ESDTRoles{}
					_ = marshalizer.Unmarshal(roles, value)
					require.Len(t, roles.Roles, 0)
					return nil
				},
			}
		},
	}
	_, err := esdtRolesF.ProcessBuiltinFunction(nil, acc, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: vmcommon.ESDTSCAddress,
			Arguments:  [][]byte{[]byte("1"), []byte(vmcommon.ESDTRoleLocalMint)},
		},
	})
	require.Nil(t, err)
}

func TestEsdtRoles_ProcessBuiltinFunction_UnsetRolesShouldWork(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.MarshalizerMock{}
	esdtRolesF, _ := NewESDTRolesFunc(marshalizer, false)

	acc := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(key []byte) ([]byte, error) {
					roles := &esdt.ESDTRoles{
						Roles: [][]byte{[]byte(vmcommon.ESDTRoleLocalMint)},
					}
					return marshalizer.Marshal(roles)
				},
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					roles := &esdt.ESDTRoles{}
					_ = marshalizer.Unmarshal(roles, value)
					require.Len(t, roles.Roles, 0)
					return nil
				},
			}
		},
	}
	_, err := esdtRolesF.ProcessBuiltinFunction(nil, acc, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: vmcommon.ESDTSCAddress,
			Arguments:  [][]byte{[]byte("1"), []byte(vmcommon.ESDTRoleLocalMint)},
		},
	})
	require.Nil(t, err)
}

func TestEsdtRoles_CheckAllowedToExecuteNilAccountShouldErr(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.MarshalizerMock{}
	esdtRolesF, _ := NewESDTRolesFunc(marshalizer, false)

	err := esdtRolesF.CheckAllowedToExecute(nil, []byte("ID"), []byte(vmcommon.ESDTRoleLocalBurn))
	require.Equal(t, ErrNilUserAccount, err)
}

func TestEsdtRoles_CheckAllowedToExecuteCannotGetESDTRole(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.MarshalizerMock{Fail: true}
	esdtRolesF, _ := NewESDTRolesFunc(marshalizer, false)

	err := esdtRolesF.CheckAllowedToExecute(&mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(key []byte) ([]byte, error) {
					return nil, nil
				},
			}
		},
	}, []byte("ID"), []byte(vmcommon.ESDTRoleLocalBurn))
	require.Error(t, err)
}

func TestEsdtRoles_CheckAllowedToExecuteIsNewNotAllowed(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.MarshalizerMock{}
	esdtRolesF, _ := NewESDTRolesFunc(marshalizer, false)

	err := esdtRolesF.CheckAllowedToExecute(&mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(key []byte) ([]byte, error) {
					return nil, nil
				},
			}
		},
	}, []byte("ID"), []byte(vmcommon.ESDTRoleLocalBurn))
	require.Equal(t, ErrActionNotAllowed, err)
}

func TestEsdtRoles_CheckAllowed_ShouldWork(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.MarshalizerMock{}
	esdtRolesF, _ := NewESDTRolesFunc(marshalizer, false)

	err := esdtRolesF.CheckAllowedToExecute(&mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(key []byte) ([]byte, error) {
					roles := &esdt.ESDTRoles{
						Roles: [][]byte{[]byte(vmcommon.ESDTRoleLocalMint)},
					}
					return marshalizer.Marshal(roles)
				},
			}
		},
	}, []byte("ID"), []byte(vmcommon.ESDTRoleLocalMint))
	require.Nil(t, err)
}

func TestEsdtRoles_CheckAllowedToExecuteRoleNotFind(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.MarshalizerMock{}
	esdtRolesF, _ := NewESDTRolesFunc(marshalizer, false)

	err := esdtRolesF.CheckAllowedToExecute(&mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(key []byte) ([]byte, error) {
					roles := &esdt.ESDTRoles{
						Roles: [][]byte{[]byte(vmcommon.ESDTRoleLocalBurn)},
					}
					return marshalizer.Marshal(roles)
				},
			}
		},
	}, []byte("ID"), []byte(vmcommon.ESDTRoleLocalMint))
	require.Equal(t, ErrActionNotAllowed, err)
}
