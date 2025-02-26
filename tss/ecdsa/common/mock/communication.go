// Code generated by MockGen. DO NOT EDIT.
// Source: ./tss/ecdsa/common/base.go
//
// Generated by this command:
//
//	mockgen -destination=./tss/ecdsa/common/mock/communication.go -source=./tss/ecdsa/common/base.go -package mock_tss
//

// Package mock_tss is a generated GoMock package.
package mock_tss

import (
	big "math/big"
	reflect "reflect"

	tss "github.com/binance-chain/tss-lib/tss"
	gomock "go.uber.org/mock/gomock"
)

// MockParty is a mock of Party interface.
type MockParty struct {
	ctrl     *gomock.Controller
	recorder *MockPartyMockRecorder
	isgomock struct{}
}

// MockPartyMockRecorder is the mock recorder for MockParty.
type MockPartyMockRecorder struct {
	mock *MockParty
}

// NewMockParty creates a new mock instance.
func NewMockParty(ctrl *gomock.Controller) *MockParty {
	mock := &MockParty{ctrl: ctrl}
	mock.recorder = &MockPartyMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockParty) EXPECT() *MockPartyMockRecorder {
	return m.recorder
}

// Start mocks base method.
func (m *MockParty) Start() *tss.Error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Start")
	ret0, _ := ret[0].(*tss.Error)
	return ret0
}

// Start indicates an expected call of Start.
func (mr *MockPartyMockRecorder) Start() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockParty)(nil).Start))
}

// UpdateFromBytes mocks base method.
func (m *MockParty) UpdateFromBytes(wireBytes []byte, from *tss.PartyID, isBroadcast bool, sessionID *big.Int) (bool, *tss.Error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateFromBytes", wireBytes, from, isBroadcast, sessionID)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(*tss.Error)
	return ret0, ret1
}

// UpdateFromBytes indicates an expected call of UpdateFromBytes.
func (mr *MockPartyMockRecorder) UpdateFromBytes(wireBytes, from, isBroadcast, sessionID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateFromBytes", reflect.TypeOf((*MockParty)(nil).UpdateFromBytes), wireBytes, from, isBroadcast, sessionID)
}

// WaitingFor mocks base method.
func (m *MockParty) WaitingFor() []*tss.PartyID {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WaitingFor")
	ret0, _ := ret[0].([]*tss.PartyID)
	return ret0
}

// WaitingFor indicates an expected call of WaitingFor.
func (mr *MockPartyMockRecorder) WaitingFor() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WaitingFor", reflect.TypeOf((*MockParty)(nil).WaitingFor))
}
