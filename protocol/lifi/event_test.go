package lifi_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/sprintertech/sprinter-signing/protocol/lifi"
	mock_lifi "github.com/sprintertech/sprinter-signing/protocol/lifi/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const testChainID uint64 = 1

type OrderTestSuite struct {
	suite.Suite
	fetcher      *lifi.LifiEventFetcher
	mockClient   *mock_lifi.MockReceiptFetcher
	inputSettler common.Address
}

func TestRunOrderTestSuite(t *testing.T) {
	suite.Run(t, new(OrderTestSuite))
}

func (s *OrderTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.mockClient = mock_lifi.NewMockReceiptFetcher(ctrl)
	s.inputSettler = common.HexToAddress("0x000025c3226C00B2Cdc200005a1600509f4e00C0")
	clients := map[uint64]lifi.ReceiptFetcher{
		testChainID: s.mockClient,
	}
	s.fetcher = lifi.NewLifiEventFetcher(clients, s.inputSettler)
}

func (s *OrderTestSuite) Test_Order_UnsupportedChain() {
	_, err := s.fetcher.Order(context.Background(), 999, common.Hash{}, common.Hash{})

	s.NotNil(err)
	s.Contains(err.Error(), "no client configured for source chain 999")
}

func (s *OrderTestSuite) Test_Order_FetchingTxFails() {
	s.mockClient.EXPECT().TransactionReceipt(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error"))

	_, err := s.fetcher.Order(context.Background(), testChainID, common.Hash{}, common.Hash{})

	s.NotNil(err)
}

func (s *OrderTestSuite) Test_Order_NoEvents() {
	s.mockClient.EXPECT().TransactionReceipt(gomock.Any(), gomock.Any()).Return(&types.Receipt{
		Logs: make([]*types.Log, 0),
	}, nil)

	_, err := s.fetcher.Order(context.Background(), testChainID, common.Hash{}, common.Hash{})

	s.NotNil(err)
}

func (s *OrderTestSuite) Test_Order_InvalidLogs() {
	validID := common.HexToHash("0x696838617ea58d56a209e54b87240778a70fb6eb0a9da7ac6d0d9de1b1a5b775")
	invalidID := common.HexToHash("0x706838617ea58d56a209e54b87240778a70fb6eb0a9da7ac6d0d9de1b1a5b775")
	invalidTopic := common.HexToHash("invalid")

	s.mockClient.EXPECT().TransactionReceipt(gomock.Any(), gomock.Any()).Return(&types.Receipt{
		Logs: []*types.Log{
			{
				Topics: []common.Hash{
					invalidTopic,
					validID,
				},
				Address: s.inputSettler,
			},
			{
				Topics: []common.Hash{
					common.HexToHash(lifi.OpenEventTopic),
					validID,
				},
				Address: s.inputSettler,
			},
			{
				Topics: []common.Hash{
					common.HexToHash(lifi.OpenEventTopic),
					validID,
				},
				Address: common.HexToAddress(""),
			},
			{
				Topics: []common.Hash{
					common.HexToHash(lifi.OpenEventTopic),
					invalidID,
				},
				Address: s.inputSettler,
			},
		},
	}, nil)

	_, err := s.fetcher.Order(context.Background(), testChainID, common.Hash{}, common.Hash{})

	s.NotNil(err)
}
