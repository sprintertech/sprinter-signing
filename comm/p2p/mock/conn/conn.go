// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/libp2p/go-libp2p/core/network (interfaces: Conn)
//
// Generated by this command:
//
//	mockgen -destination=./comm/p2p/mock/conn/conn.go github.com/libp2p/go-libp2p/core/network Conn
//

// Package mock_network is a generated GoMock package.
package mock_network

import (
	context "context"
	reflect "reflect"

	crypto "github.com/libp2p/go-libp2p/core/crypto"
	network "github.com/libp2p/go-libp2p/core/network"
	peer "github.com/libp2p/go-libp2p/core/peer"
	multiaddr "github.com/multiformats/go-multiaddr"
	gomock "go.uber.org/mock/gomock"
)

// MockConn is a mock of Conn interface.
type MockConn struct {
	ctrl     *gomock.Controller
	recorder *MockConnMockRecorder
	isgomock struct{}
}

// MockConnMockRecorder is the mock recorder for MockConn.
type MockConnMockRecorder struct {
	mock *MockConn
}

// NewMockConn creates a new mock instance.
func NewMockConn(ctrl *gomock.Controller) *MockConn {
	mock := &MockConn{ctrl: ctrl}
	mock.recorder = &MockConnMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockConn) EXPECT() *MockConnMockRecorder {
	return m.recorder
}

// Close mocks base method.
func (m *MockConn) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockConnMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockConn)(nil).Close))
}

// ConnState mocks base method.
func (m *MockConn) ConnState() network.ConnectionState {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConnState")
	ret0, _ := ret[0].(network.ConnectionState)
	return ret0
}

// ConnState indicates an expected call of ConnState.
func (mr *MockConnMockRecorder) ConnState() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConnState", reflect.TypeOf((*MockConn)(nil).ConnState))
}

// GetStreams mocks base method.
func (m *MockConn) GetStreams() []network.Stream {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStreams")
	ret0, _ := ret[0].([]network.Stream)
	return ret0
}

// GetStreams indicates an expected call of GetStreams.
func (mr *MockConnMockRecorder) GetStreams() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStreams", reflect.TypeOf((*MockConn)(nil).GetStreams))
}

// ID mocks base method.
func (m *MockConn) ID() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ID")
	ret0, _ := ret[0].(string)
	return ret0
}

// ID indicates an expected call of ID.
func (mr *MockConnMockRecorder) ID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ID", reflect.TypeOf((*MockConn)(nil).ID))
}

// IsClosed mocks base method.
func (m *MockConn) IsClosed() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsClosed")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsClosed indicates an expected call of IsClosed.
func (mr *MockConnMockRecorder) IsClosed() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsClosed", reflect.TypeOf((*MockConn)(nil).IsClosed))
}

// LocalMultiaddr mocks base method.
func (m *MockConn) LocalMultiaddr() multiaddr.Multiaddr {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LocalMultiaddr")
	ret0, _ := ret[0].(multiaddr.Multiaddr)
	return ret0
}

// LocalMultiaddr indicates an expected call of LocalMultiaddr.
func (mr *MockConnMockRecorder) LocalMultiaddr() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LocalMultiaddr", reflect.TypeOf((*MockConn)(nil).LocalMultiaddr))
}

// LocalPeer mocks base method.
func (m *MockConn) LocalPeer() peer.ID {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LocalPeer")
	ret0, _ := ret[0].(peer.ID)
	return ret0
}

// LocalPeer indicates an expected call of LocalPeer.
func (mr *MockConnMockRecorder) LocalPeer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LocalPeer", reflect.TypeOf((*MockConn)(nil).LocalPeer))
}

// NewStream mocks base method.
func (m *MockConn) NewStream(arg0 context.Context) (network.Stream, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewStream", arg0)
	ret0, _ := ret[0].(network.Stream)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewStream indicates an expected call of NewStream.
func (mr *MockConnMockRecorder) NewStream(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewStream", reflect.TypeOf((*MockConn)(nil).NewStream), arg0)
}

// RemoteMultiaddr mocks base method.
func (m *MockConn) RemoteMultiaddr() multiaddr.Multiaddr {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoteMultiaddr")
	ret0, _ := ret[0].(multiaddr.Multiaddr)
	return ret0
}

// RemoteMultiaddr indicates an expected call of RemoteMultiaddr.
func (mr *MockConnMockRecorder) RemoteMultiaddr() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoteMultiaddr", reflect.TypeOf((*MockConn)(nil).RemoteMultiaddr))
}

// RemotePeer mocks base method.
func (m *MockConn) RemotePeer() peer.ID {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemotePeer")
	ret0, _ := ret[0].(peer.ID)
	return ret0
}

// RemotePeer indicates an expected call of RemotePeer.
func (mr *MockConnMockRecorder) RemotePeer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemotePeer", reflect.TypeOf((*MockConn)(nil).RemotePeer))
}

// RemotePublicKey mocks base method.
func (m *MockConn) RemotePublicKey() crypto.PubKey {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemotePublicKey")
	ret0, _ := ret[0].(crypto.PubKey)
	return ret0
}

// RemotePublicKey indicates an expected call of RemotePublicKey.
func (mr *MockConnMockRecorder) RemotePublicKey() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemotePublicKey", reflect.TypeOf((*MockConn)(nil).RemotePublicKey))
}

// Scope mocks base method.
func (m *MockConn) Scope() network.ConnScope {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Scope")
	ret0, _ := ret[0].(network.ConnScope)
	return ret0
}

// Scope indicates an expected call of Scope.
func (mr *MockConnMockRecorder) Scope() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Scope", reflect.TypeOf((*MockConn)(nil).Scope))
}

// Stat mocks base method.
func (m *MockConn) Stat() network.ConnStats {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stat")
	ret0, _ := ret[0].(network.ConnStats)
	return ret0
}

// Stat indicates an expected call of Stat.
func (mr *MockConnMockRecorder) Stat() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stat", reflect.TypeOf((*MockConn)(nil).Stat))
}
