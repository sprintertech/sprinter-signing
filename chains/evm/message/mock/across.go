// Code generated by MockGen. DO NOT EDIT.
// Source: ./chains/evm/message/across.go
//
// Generated by this command:
//
//	mockgen -source=./chains/evm/message/across.go -destination=./chains/evm/message/mock/across.go
//

// Package mock_message is a generated GoMock package.
package mock_message

import (
	context "context"
	big "math/big"
	reflect "reflect"

	ethereum "github.com/ethereum/go-ethereum"
	common "github.com/ethereum/go-ethereum/common"
	types "github.com/ethereum/go-ethereum/core/types"
	peer "github.com/libp2p/go-libp2p/core/peer"
	evm "github.com/sprintertech/sprinter-signing/chains/evm"
	tss "github.com/sprintertech/sprinter-signing/tss"
	gomock "go.uber.org/mock/gomock"
)

// MockEventFilterer is a mock of EventFilterer interface.
type MockEventFilterer struct {
	ctrl     *gomock.Controller
	recorder *MockEventFiltererMockRecorder
	isgomock struct{}
}

// MockEventFiltererMockRecorder is the mock recorder for MockEventFilterer.
type MockEventFiltererMockRecorder struct {
	mock *MockEventFilterer
}

// NewMockEventFilterer creates a new mock instance.
func NewMockEventFilterer(ctrl *gomock.Controller) *MockEventFilterer {
	mock := &MockEventFilterer{ctrl: ctrl}
	mock.recorder = &MockEventFiltererMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEventFilterer) EXPECT() *MockEventFiltererMockRecorder {
	return m.recorder
}

// FilterLogs mocks base method.
func (m *MockEventFilterer) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FilterLogs", ctx, q)
	ret0, _ := ret[0].([]types.Log)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FilterLogs indicates an expected call of FilterLogs.
func (mr *MockEventFiltererMockRecorder) FilterLogs(ctx, q any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FilterLogs", reflect.TypeOf((*MockEventFilterer)(nil).FilterLogs), ctx, q)
}

// LatestBlock mocks base method.
func (m *MockEventFilterer) LatestBlock() (*big.Int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LatestBlock")
	ret0, _ := ret[0].(*big.Int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LatestBlock indicates an expected call of LatestBlock.
func (mr *MockEventFiltererMockRecorder) LatestBlock() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LatestBlock", reflect.TypeOf((*MockEventFilterer)(nil).LatestBlock))
}

// TransactionReceipt mocks base method.
func (m *MockEventFilterer) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TransactionReceipt", ctx, txHash)
	ret0, _ := ret[0].(*types.Receipt)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// TransactionReceipt indicates an expected call of TransactionReceipt.
func (mr *MockEventFiltererMockRecorder) TransactionReceipt(ctx, txHash any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TransactionReceipt", reflect.TypeOf((*MockEventFilterer)(nil).TransactionReceipt), ctx, txHash)
}

// MockCoordinator is a mock of Coordinator interface.
type MockCoordinator struct {
	ctrl     *gomock.Controller
	recorder *MockCoordinatorMockRecorder
	isgomock struct{}
}

// MockCoordinatorMockRecorder is the mock recorder for MockCoordinator.
type MockCoordinatorMockRecorder struct {
	mock *MockCoordinator
}

// NewMockCoordinator creates a new mock instance.
func NewMockCoordinator(ctrl *gomock.Controller) *MockCoordinator {
	mock := &MockCoordinator{ctrl: ctrl}
	mock.recorder = &MockCoordinatorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCoordinator) EXPECT() *MockCoordinatorMockRecorder {
	return m.recorder
}

// Execute mocks base method.
func (m *MockCoordinator) Execute(ctx context.Context, tssProcesses []tss.TssProcess, resultChn chan any, coordinator peer.ID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Execute", ctx, tssProcesses, resultChn, coordinator)
	ret0, _ := ret[0].(error)
	return ret0
}

// Execute indicates an expected call of Execute.
func (mr *MockCoordinatorMockRecorder) Execute(ctx, tssProcesses, resultChn, coordinator any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Execute", reflect.TypeOf((*MockCoordinator)(nil).Execute), ctx, tssProcesses, resultChn, coordinator)
}

// MockTokenMatcher is a mock of TokenMatcher interface.
type MockTokenMatcher struct {
	ctrl     *gomock.Controller
	recorder *MockTokenMatcherMockRecorder
	isgomock struct{}
}

// MockTokenMatcherMockRecorder is the mock recorder for MockTokenMatcher.
type MockTokenMatcherMockRecorder struct {
	mock *MockTokenMatcher
}

// NewMockTokenMatcher creates a new mock instance.
func NewMockTokenMatcher(ctrl *gomock.Controller) *MockTokenMatcher {
	mock := &MockTokenMatcher{ctrl: ctrl}
	mock.recorder = &MockTokenMatcherMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTokenMatcher) EXPECT() *MockTokenMatcherMockRecorder {
	return m.recorder
}

// DestinationToken mocks base method.
func (m *MockTokenMatcher) DestinationToken(destinationChainId *big.Int, symbol string) (common.Address, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DestinationToken", destinationChainId, symbol)
	ret0, _ := ret[0].(common.Address)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DestinationToken indicates an expected call of DestinationToken.
func (mr *MockTokenMatcherMockRecorder) DestinationToken(destinationChainId, symbol any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DestinationToken", reflect.TypeOf((*MockTokenMatcher)(nil).DestinationToken), destinationChainId, symbol)
}

// MockConfirmationWatcher is a mock of ConfirmationWatcher interface.
type MockConfirmationWatcher struct {
	ctrl     *gomock.Controller
	recorder *MockConfirmationWatcherMockRecorder
	isgomock struct{}
}

// MockConfirmationWatcherMockRecorder is the mock recorder for MockConfirmationWatcher.
type MockConfirmationWatcherMockRecorder struct {
	mock *MockConfirmationWatcher
}

// NewMockConfirmationWatcher creates a new mock instance.
func NewMockConfirmationWatcher(ctrl *gomock.Controller) *MockConfirmationWatcher {
	mock := &MockConfirmationWatcher{ctrl: ctrl}
	mock.recorder = &MockConfirmationWatcherMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockConfirmationWatcher) EXPECT() *MockConfirmationWatcherMockRecorder {
	return m.recorder
}

// TokenConfig mocks base method.
func (m *MockConfirmationWatcher) TokenConfig(chainID uint64, token common.Address) (string, evm.TokenConfig, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TokenConfig", chainID, token)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(evm.TokenConfig)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// TokenConfig indicates an expected call of TokenConfig.
func (mr *MockConfirmationWatcherMockRecorder) TokenConfig(chainID, token any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TokenConfig", reflect.TypeOf((*MockConfirmationWatcher)(nil).TokenConfig), chainID, token)
}

// WaitForConfirmations mocks base method.
func (m *MockConfirmationWatcher) WaitForConfirmations(ctx context.Context, chainID uint64, txHash common.Hash, token common.Address, amount *big.Int) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WaitForConfirmations", ctx, chainID, txHash, token, amount)
	ret0, _ := ret[0].(error)
	return ret0
}

// WaitForConfirmations indicates an expected call of WaitForConfirmations.
func (mr *MockConfirmationWatcherMockRecorder) WaitForConfirmations(ctx, chainID, txHash, token, amount any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WaitForConfirmations", reflect.TypeOf((*MockConfirmationWatcher)(nil).WaitForConfirmations), ctx, chainID, txHash, token, amount)
}
