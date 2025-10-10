package message_test

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/sprintertech/sprinter-signing/chains/evm/message"
	mock_message "github.com/sprintertech/sprinter-signing/chains/evm/message/mock"
	"github.com/sprintertech/sprinter-signing/config"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type WatcherTestSuite struct {
	suite.Suite

	watcher *message.Watcher

	mockClient *mock_message.MockEventFilterer
	mockPricer *mock_message.MockTokenPricer

	usdcToken common.Address
}

func TestRunWatcherTestSuite(t *testing.T) {
	suite.Run(t, new(WatcherTestSuite))
}

func (s *WatcherTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())

	s.mockClient = mock_message.NewMockEventFilterer(ctrl)
	s.mockPricer = mock_message.NewMockTokenPricer(ctrl)

	confirmations := make(map[uint64]uint64)
	confirmations[500] = 2

	s.usdcToken = common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	tokens := make(map[uint64]map[string]config.TokenConfig)
	tokens[1] = make(map[string]config.TokenConfig)
	tokens[1]["USDC"] = config.TokenConfig{
		Decimals: 6,
		Address:  s.usdcToken,
	}
	tokenStore := config.TokenStore{
		Tokens: tokens,
	}

	s.watcher = message.NewWatcher(
		s.mockClient,
		s.mockPricer,
		tokenStore,
		confirmations,
		time.Millisecond,
	)
}

func (s *WatcherTestSuite) Test_WaitForTokenConfirmations_InvalidToken() {
	err := s.watcher.WaitForTokenConfirmations(context.Background(), 1, common.Hash{}, common.Address{}, big.NewInt(1000))

	s.NotNil(err)
}

func (s *WatcherTestSuite) Test_WaitForTokenConfirmations_InvalidOrderValue() {
	s.mockPricer.EXPECT().TokenPrice("USDC").Return(float64(0.99), nil)

	err := s.watcher.WaitForTokenConfirmations(context.Background(), 1, common.Hash{}, s.usdcToken, big.NewInt(1000000000))

	s.NotNil(err)
}

func (s *WatcherTestSuite) Test_WaitForTokenConfirmations_TxTimeout() {
	s.mockPricer.EXPECT().TokenPrice("USDC").Return(float64(0.99), nil)

	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()

	err := s.watcher.WaitForTokenConfirmations(ctx, 1, common.Hash{}, s.usdcToken, big.NewInt(499000000))

	s.NotNil(err)
}

func (s *WatcherTestSuite) Test_WaitForTokenConfirmations_ValidTransaction() {
	s.mockPricer.EXPECT().TokenPrice("USDC").Return(float64(0.99), nil)
	s.mockClient.EXPECT().TransactionReceipt(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error"))
	s.mockClient.EXPECT().TransactionReceipt(gomock.Any(), gomock.Any()).Return(nil, nil)
	s.mockClient.EXPECT().TransactionReceipt(gomock.Any(), gomock.Any()).Return(&types.Receipt{
		BlockNumber: big.NewInt(100),
	}, nil).AnyTimes()
	s.mockClient.EXPECT().LatestBlock().Return(nil, fmt.Errorf("error"))
	s.mockClient.EXPECT().LatestBlock().Return(big.NewInt(100), nil)
	s.mockClient.EXPECT().LatestBlock().Return(big.NewInt(102), nil)

	err := s.watcher.WaitForTokenConfirmations(context.Background(), 1, common.Hash{}, s.usdcToken, big.NewInt(499000000))

	s.Nil(err)
}

func (s *WatcherTestSuite) Test_WaitForOrderConfirmations_ValidTransaction() {
	s.mockClient.EXPECT().TransactionReceipt(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error"))
	s.mockClient.EXPECT().TransactionReceipt(gomock.Any(), gomock.Any()).Return(nil, nil)
	s.mockClient.EXPECT().TransactionReceipt(gomock.Any(), gomock.Any()).Return(&types.Receipt{
		BlockNumber: big.NewInt(100),
	}, nil).AnyTimes()
	s.mockClient.EXPECT().LatestBlock().Return(nil, fmt.Errorf("error"))
	s.mockClient.EXPECT().LatestBlock().Return(big.NewInt(100), nil)
	s.mockClient.EXPECT().LatestBlock().Return(big.NewInt(102), nil)

	err := s.watcher.WaitForOrderConfirmations(context.Background(), 1, common.Hash{}, 499.95)

	s.Nil(err)
}
