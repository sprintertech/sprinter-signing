package across_test

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/events"
	"github.com/sprintertech/sprinter-signing/config"
	"github.com/sprintertech/sprinter-signing/protocol/across"
	mock_across "github.com/sprintertech/sprinter-signing/protocol/across/mock"
	"github.com/sprintertech/sprinter-signing/protocol/lifi"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type DepositTestSuite struct {
	suite.Suite
	fetcher           *across.AcrossDepositFetcher
	mockClient        *mock_across.MockEventFilterer
	mockTockenMatcher *mock_across.MockTokenMatcher
	spokePool         common.Address
}

func TestRunDepositTestSuite(t *testing.T) {
	suite.Run(t, new(DepositTestSuite))
}

func (s *DepositTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.mockClient = mock_across.NewMockEventFilterer(ctrl)
	s.mockTockenMatcher = mock_across.NewMockTokenMatcher(ctrl)
	s.spokePool = common.HexToAddress("0x000025c3226C00B2Cdc200005a1600509f4e00C0")
	s.fetcher = across.NewAcrossDepositFetcher(
		8453,
		s.spokePool,
		config.TokenStore{},
		s.mockClient,
		s.mockTockenMatcher,
	)
}

func (s *DepositTestSuite) Test_Deposit_FetchingTxFails() {
	s.mockClient.EXPECT().TransactionReceipt(
		gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	_, err := s.fetcher.Deposit(s.T().Context(), common.Hash{}, big.NewInt(1))

	s.NotNil(err)
}

func (s *DepositTestSuite) Test_Deposit_InvalidLogs() {
	validID := common.HexToHash("0x696838617ea58d56a209e54b87240778a70fb6eb0a9da7ac6d0d9de1b1a5b775")
	invalidID := common.HexToHash("0x706838617ea58d56a209e54b87240778a70fb6eb0a9da7ac6d0d9de1b1a5b775")
	invalidTopic := common.HexToHash("invalid")

	s.mockClient.EXPECT().TransactionReceipt(gomock.Any(), gomock.Any()).Return(&types.Receipt{
		Logs: []*types.Log{
			{
				Topics: []common.Hash{
					invalidTopic,
					validID,
					validID,
				},
				Address: s.spokePool,
			},
			{
				Topics: []common.Hash{
					common.HexToHash(lifi.OpenEventTopic),
					validID,
					validID,
				},
				Address: common.Address{},
			},
			{
				Topics: []common.Hash{
					common.HexToHash(events.AcrossDepositSig.GetTopic().Hex()),
					validID,
					invalidID,
				},
				Address: s.spokePool,
			},
		},
	}, nil)

	_, err := s.fetcher.Deposit(
		s.T().Context(), common.Hash{}, new(big.Int).SetBytes(validID[:]))

	s.NotNil(err)
}
