package builtInFunctions

import (
	"bytes"
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/data/esdt"
	"github.com/multiversx/mx-chain-vm-common-go"
	"github.com/multiversx/mx-chain-vm-common-go/mock"
	"github.com/stretchr/testify/require"
)

func TestNewESDTRolesFunc_NilMarshalizerShouldErr(t *testing.T) {
	t.Parallel()

	esdtRolesF, err := NewESDTRolesFunc(nil, &mock.CrossChainTokenCheckerMock{}, false)

	require.Equal(t, ErrNilMarshalizer, err)
	require.Nil(t, esdtRolesF)
}

func TestEsdtRoles_ProcessBuiltinFunction_NilVMInputShouldErr(t *testing.T) {
	t.Parallel()

	esdtRolesF, _ := NewESDTRolesFunc(nil, &mock.CrossChainTokenCheckerMock{}, false)

	_, err := esdtRolesF.ProcessBuiltinFunction(nil, &mock.UserAccountStub{}, nil)
	require.Equal(t, ErrNilVmInput, err)
}

func TestEsdtRoles_ProcessBuiltinFunction_WrongCalledShouldErr(t *testing.T) {
	t.Parallel()

	esdtRolesF, _ := NewESDTRolesFunc(nil, &mock.CrossChainTokenCheckerMock{}, false)

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

	esdtRolesF, _ := NewESDTRolesFunc(nil, &mock.CrossChainTokenCheckerMock{}, false)

	_, err := esdtRolesF.ProcessBuiltinFunction(nil, nil, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: core.ESDTSCAddress,
			Arguments:  [][]byte{[]byte("1"), []byte("2")},
		},
	})
	require.Equal(t, ErrNilUserAccount, err)
}

func TestEsdtRoles_ProcessBuiltinFunction_GetRolesFailShouldErr(t *testing.T) {
	t.Parallel()

	esdtRolesF, _ := NewESDTRolesFunc(&mock.MarshalizerMock{Fail: true}, &mock.CrossChainTokenCheckerMock{}, false)

	_, err := esdtRolesF.ProcessBuiltinFunction(nil, &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					return nil, 0, nil
				},
			}
		},
	}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: core.ESDTSCAddress,
			Arguments:  [][]byte{[]byte("1"), []byte("2")},
		},
	})
	require.Error(t, err)
}

func TestEsdtRoles_ProcessBuiltinFunction_GetRolesFailShouldWorkEvenIfAccntTrieIsNil(t *testing.T) {
	t.Parallel()

	saveKeyWasCalled := false
	esdtRolesF, _ := NewESDTRolesFunc(&mock.MarshalizerMock{}, &mock.CrossChainTokenCheckerMock{}, false)

	_, err := esdtRolesF.ProcessBuiltinFunction(nil, &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					return nil, 0, nil
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
			CallerAddr: core.ESDTSCAddress,
			Arguments:  [][]byte{[]byte("1"), []byte("2")},
		},
	})
	require.NoError(t, err)
	require.True(t, saveKeyWasCalled)
}

func TestEsdtRoles_ProcessBuiltinFunction_SetRolesShouldWork(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	esdtRolesF, _ := NewESDTRolesFunc(marshaller, &mock.CrossChainTokenCheckerMock{}, true)

	acc := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					roles := &esdt.ESDTRoles{}
					serializedRoles, err := marshaller.Marshal(roles)
					return serializedRoles, 0, err
				},
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					roles := &esdt.ESDTRoles{}
					_ = marshaller.Unmarshal(roles, value)
					require.Equal(t, roles.Roles, [][]byte{[]byte(core.ESDTRoleLocalMint)})
					return nil
				},
			}
		},
	}
	_, err := esdtRolesF.ProcessBuiltinFunction(nil, acc, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: core.ESDTSCAddress,
			Arguments:  [][]byte{[]byte("1"), []byte(core.ESDTRoleLocalMint)},
		},
	})
	require.Nil(t, err)
}

func TestEsdtRoles_ProcessBuiltinFunction_SetRolesMultiNFT(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	esdtRolesF, _ := NewESDTRolesFunc(marshaller, &mock.CrossChainTokenCheckerMock{}, true)

	tokenID := []byte("tokenID")
	roleKey := append(roleKeyPrefix, tokenID...)

	saveNonceCalled := false
	acc := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					roles := &esdt.ESDTRoles{}
					serializedRoles, err := marshaller.Marshal(roles)
					return serializedRoles, 0, err
				},
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					if bytes.Equal(key, roleKey) {
						roles := &esdt.ESDTRoles{}
						_ = marshaller.Unmarshal(roles, value)
						require.Equal(t, roles.Roles, [][]byte{[]byte(core.ESDTRoleNFTCreate), []byte(core.ESDTRoleNFTCreateMultiShard)})
						return nil
					}

					if bytes.Equal(key, getNonceKey(tokenID)) {
						saveNonceCalled = true
						require.Equal(t, uint64(math.MaxUint64/256), big.NewInt(0).SetBytes(value).Uint64())
					}

					return nil
				},
			}
		},
	}
	dstAddr := bytes.Repeat([]byte{1}, 32)
	_, err := esdtRolesF.ProcessBuiltinFunction(nil, acc, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: core.ESDTSCAddress,
			Arguments:  [][]byte{tokenID, []byte(core.ESDTRoleNFTCreate), []byte(core.ESDTRoleNFTCreateMultiShard)},
		},
		RecipientAddr: dstAddr,
	})

	require.Nil(t, err)
	require.True(t, saveNonceCalled)
}

func TestEsdtRoles_ProcessBuiltinFunction_SaveFailedShouldErr(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	esdtRolesF, _ := NewESDTRolesFunc(marshaller, &mock.CrossChainTokenCheckerMock{}, true)

	localErr := errors.New("local err")
	acc := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					roles := &esdt.ESDTRoles{}
					serializedRoles, err := marshaller.Marshal(roles)
					return serializedRoles, 0, err
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
			CallerAddr: core.ESDTSCAddress,
			Arguments:  [][]byte{[]byte("1"), []byte(core.ESDTRoleLocalMint)},
		},
	})
	require.Equal(t, localErr, err)
}

func TestEsdtRoles_ProcessBuiltinFunction_UnsetRolesDoesNotExistsShouldWork(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	esdtRolesF, _ := NewESDTRolesFunc(marshaller, &mock.CrossChainTokenCheckerMock{}, false)

	acc := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					roles := &esdt.ESDTRoles{}
					serializedRoles, err := marshaller.Marshal(roles)
					return serializedRoles, 0, err
				},
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					roles := &esdt.ESDTRoles{}
					_ = marshaller.Unmarshal(roles, value)
					require.Len(t, roles.Roles, 0)
					return nil
				},
			}
		},
	}
	_, err := esdtRolesF.ProcessBuiltinFunction(nil, acc, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: core.ESDTSCAddress,
			Arguments:  [][]byte{[]byte("1"), []byte(core.ESDTRoleLocalMint)},
		},
	})
	require.Nil(t, err)
}

func TestEsdtRoles_ProcessBuiltinFunction_UnsetRolesShouldWork(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	esdtRolesF, _ := NewESDTRolesFunc(marshaller, &mock.CrossChainTokenCheckerMock{}, false)

	acc := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					roles := &esdt.ESDTRoles{
						Roles: [][]byte{[]byte(core.ESDTRoleLocalMint)},
					}
					serializedRoles, err := marshaller.Marshal(roles)
					return serializedRoles, 0, err
				},
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					roles := &esdt.ESDTRoles{}
					_ = marshaller.Unmarshal(roles, value)
					require.Len(t, roles.Roles, 0)
					return nil
				},
			}
		},
	}
	_, err := esdtRolesF.ProcessBuiltinFunction(nil, acc, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: core.ESDTSCAddress,
			Arguments:  [][]byte{[]byte("1"), []byte(core.ESDTRoleLocalMint)},
		},
	})
	require.Nil(t, err)
}

func TestEsdtRoles_CheckAllowedToExecuteNilAccountShouldErr(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	esdtRolesF, _ := NewESDTRolesFunc(marshaller, &mock.CrossChainTokenCheckerMock{}, false)

	err := esdtRolesF.CheckAllowedToExecute(nil, []byte("ID"), []byte(core.ESDTRoleLocalBurn))
	require.Equal(t, ErrNilUserAccount, err)
}

func TestEsdtRoles_CheckAllowedToExecuteCannotGetESDTRole(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{Fail: true}
	esdtRolesF, _ := NewESDTRolesFunc(marshaller, &mock.CrossChainTokenCheckerMock{}, false)

	err := esdtRolesF.CheckAllowedToExecute(&mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					return nil, 0, nil
				},
			}
		},
	}, []byte("ID"), []byte(core.ESDTRoleLocalBurn))
	require.Error(t, err)
}

func TestEsdtRoles_CheckAllowedToExecuteIsNewNotAllowed(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	esdtRolesF, _ := NewESDTRolesFunc(marshaller, &mock.CrossChainTokenCheckerMock{}, false)

	err := esdtRolesF.CheckAllowedToExecute(&mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					return nil, 0, nil
				},
			}
		},
	}, []byte("ID"), []byte(core.ESDTRoleLocalBurn))
	require.Equal(t, ErrActionNotAllowed, err)
}

func TestEsdtRoles_CheckAllowed_ShouldWork(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	esdtRolesF, _ := NewESDTRolesFunc(marshaller, &mock.CrossChainTokenCheckerMock{}, false)

	err := esdtRolesF.CheckAllowedToExecute(&mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					roles := &esdt.ESDTRoles{
						Roles: [][]byte{[]byte(core.ESDTRoleLocalMint)},
					}
					serializedRoles, err := marshaller.Marshal(roles)
					return serializedRoles, 0, err
				},
			}
		},
	}, []byte("ID"), []byte(core.ESDTRoleLocalMint))
	require.Nil(t, err)
}

func TestEsdtRoles_CheckAllowedToExecuteRoleNotFind(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	esdtRolesF, _ := NewESDTRolesFunc(marshaller, &mock.CrossChainTokenCheckerMock{}, false)

	err := esdtRolesF.CheckAllowedToExecute(&mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					roles := &esdt.ESDTRoles{
						Roles: [][]byte{[]byte(core.ESDTRoleLocalBurn)},
					}
					serializedRoles, err := marshaller.Marshal(roles)
					return serializedRoles, 0, err
				},
			}
		},
	}, []byte("ID"), []byte(core.ESDTRoleLocalMint))
	require.Equal(t, ErrActionNotAllowed, err)
}
