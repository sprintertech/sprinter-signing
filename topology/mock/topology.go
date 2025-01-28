// Code generated by MockGen. DO NOT EDIT.
// Source: ./topology/topology.go

// Package mock_topology is a generated GoMock package.
package mock_topology

import (
	http "net/http"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	topology "github.com/sprintertech/sprinter-signing/topology"
)

// MockFetcher is a mock of Fetcher interface.
type MockFetcher struct {
	ctrl     *gomock.Controller
	recorder *MockFetcherMockRecorder
}

// MockFetcherMockRecorder is the mock recorder for MockFetcher.
type MockFetcherMockRecorder struct {
	mock *MockFetcher
}

// NewMockFetcher creates a new mock instance.
func NewMockFetcher(ctrl *gomock.Controller) *MockFetcher {
	mock := &MockFetcher{ctrl: ctrl}
	mock.recorder = &MockFetcherMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockFetcher) EXPECT() *MockFetcherMockRecorder {
	return m.recorder
}

// Get mocks base method.
func (m *MockFetcher) Get(url string) (*http.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", url)
	ret0, _ := ret[0].(*http.Response)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockFetcherMockRecorder) Get(url interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockFetcher)(nil).Get), url)
}

// MockDecrypter is a mock of Decrypter interface.
type MockDecrypter struct {
	ctrl     *gomock.Controller
	recorder *MockDecrypterMockRecorder
}

// MockDecrypterMockRecorder is the mock recorder for MockDecrypter.
type MockDecrypterMockRecorder struct {
	mock *MockDecrypter
}

// NewMockDecrypter creates a new mock instance.
func NewMockDecrypter(ctrl *gomock.Controller) *MockDecrypter {
	mock := &MockDecrypter{ctrl: ctrl}
	mock.recorder = &MockDecrypterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDecrypter) EXPECT() *MockDecrypterMockRecorder {
	return m.recorder
}

// Decrypt mocks base method.
func (m *MockDecrypter) Decrypt(data []byte) []byte {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Decrypt", data)
	ret0, _ := ret[0].([]byte)
	return ret0
}

// Decrypt indicates an expected call of Decrypt.
func (mr *MockDecrypterMockRecorder) Decrypt(data interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Decrypt", reflect.TypeOf((*MockDecrypter)(nil).Decrypt), data)
}

// MockNetworkTopologyProvider is a mock of NetworkTopologyProvider interface.
type MockNetworkTopologyProvider struct {
	ctrl     *gomock.Controller
	recorder *MockNetworkTopologyProviderMockRecorder
}

// MockNetworkTopologyProviderMockRecorder is the mock recorder for MockNetworkTopologyProvider.
type MockNetworkTopologyProviderMockRecorder struct {
	mock *MockNetworkTopologyProvider
}

// NewMockNetworkTopologyProvider creates a new mock instance.
func NewMockNetworkTopologyProvider(ctrl *gomock.Controller) *MockNetworkTopologyProvider {
	mock := &MockNetworkTopologyProvider{ctrl: ctrl}
	mock.recorder = &MockNetworkTopologyProviderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockNetworkTopologyProvider) EXPECT() *MockNetworkTopologyProviderMockRecorder {
	return m.recorder
}

// NetworkTopology mocks base method.
func (m *MockNetworkTopologyProvider) NetworkTopology(hash string) (*topology.NetworkTopology, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NetworkTopology", hash)
	ret0, _ := ret[0].(*topology.NetworkTopology)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NetworkTopology indicates an expected call of NetworkTopology.
func (mr *MockNetworkTopologyProviderMockRecorder) NetworkTopology(hash interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NetworkTopology", reflect.TypeOf((*MockNetworkTopologyProvider)(nil).NetworkTopology), hash)
}
