// Code generated by MockGen. DO NOT EDIT.
// Source: peerrole.go

// Package p2pmock is a generated GoMock package.
package p2pmock

import (
	p2pcommon "github.com/aergoio/aergo/p2p/p2pcommon"
	types "github.com/aergoio/aergo/types"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockPeerRoleManager is a mock of PeerRoleManager interface
type MockPeerRoleManager struct {
	ctrl     *gomock.Controller
	recorder *MockPeerRoleManagerMockRecorder
}

// MockPeerRoleManagerMockRecorder is the mock recorder for MockPeerRoleManager
type MockPeerRoleManagerMockRecorder struct {
	mock *MockPeerRoleManager
}

// NewMockPeerRoleManager creates a new mock instance
func NewMockPeerRoleManager(ctrl *gomock.Controller) *MockPeerRoleManager {
	mock := &MockPeerRoleManager{ctrl: ctrl}
	mock.recorder = &MockPeerRoleManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockPeerRoleManager) EXPECT() *MockPeerRoleManagerMockRecorder {
	return m.recorder
}

// UpdateBP mocks base method
func (m *MockPeerRoleManager) UpdateBP(toAdd, toRemove []types.PeerID) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "UpdateBP", toAdd, toRemove)
}

// UpdateBP indicates an expected call of UpdateBP
func (mr *MockPeerRoleManagerMockRecorder) UpdateBP(toAdd, toRemove interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateBP", reflect.TypeOf((*MockPeerRoleManager)(nil).UpdateBP), toAdd, toRemove)
}

// SelfRole mocks base method
func (m *MockPeerRoleManager) SelfRole() p2pcommon.PeerRole {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SelfRole")
	ret0, _ := ret[0].(p2pcommon.PeerRole)
	return ret0
}

// SelfRole indicates an expected call of SelfRole
func (mr *MockPeerRoleManagerMockRecorder) SelfRole() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SelfRole", reflect.TypeOf((*MockPeerRoleManager)(nil).SelfRole))
}

// GetRole mocks base method
func (m *MockPeerRoleManager) GetRole(pid types.PeerID) p2pcommon.PeerRole {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRole", pid)
	ret0, _ := ret[0].(p2pcommon.PeerRole)
	return ret0
}

// GetRole indicates an expected call of GetRole
func (mr *MockPeerRoleManagerMockRecorder) GetRole(pid interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRole", reflect.TypeOf((*MockPeerRoleManager)(nil).GetRole), pid)
}

// NotifyNewBlockMsg mocks base method
func (m *MockPeerRoleManager) NotifyNewBlockMsg(mo p2pcommon.MsgOrder, peers []p2pcommon.RemotePeer) (int, int) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NotifyNewBlockMsg", mo, peers)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(int)
	return ret0, ret1
}

// NotifyNewBlockMsg indicates an expected call of NotifyNewBlockMsg
func (mr *MockPeerRoleManagerMockRecorder) NotifyNewBlockMsg(mo, peers interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NotifyNewBlockMsg", reflect.TypeOf((*MockPeerRoleManager)(nil).NotifyNewBlockMsg), mo, peers)
}
