package message_test

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
	lifiTypes "github.com/sprintertech/lifi-solver/pkg/protocols/lifi"
	"github.com/sprintertech/sprinter-signing/chains/evm/message"
	mock_message "github.com/sprintertech/sprinter-signing/chains/evm/message/mock"
	"github.com/sprintertech/sprinter-signing/comm"
	mock_communication "github.com/sprintertech/sprinter-signing/comm/mock"
	mock_host "github.com/sprintertech/sprinter-signing/comm/p2p/mock/host"
	"github.com/sprintertech/sprinter-signing/config"
	"github.com/sprintertech/sprinter-signing/keyshare"
	"github.com/sprintertech/sprinter-signing/protocol/lifi/mock"
	mock_tss "github.com/sprintertech/sprinter-signing/tss/ecdsa/common/mock"
	"github.com/stretchr/testify/suite"
	coreMessage "github.com/sygmaprotocol/sygma-core/relayer/message"
	"go.uber.org/mock/gomock"
)

type LifiEscrowMessageHandlerTestSuite struct {
	suite.Suite

	mockCommunication *mock_communication.MockCommunication
	mockCoordinator   *mock_message.MockCoordinator
	mockHost          *mock_host.MockHost
	mockFetcher       *mock_tss.MockSaveDataFetcher
	mockOrder         *lifiTypes.LifiOrder
	mockWatcher       *mock_message.MockConfirmationWatcher

	sigChn chan interface{}

	mockOrderFetcher   *mock_message.MockOrderFetcher
	mockOrderPricer    *mock_message.MockOrderPricer
	mockOrderValidator *mock_message.MockOrderValidator

	handler *message.LifiEscrowMessageHandler
}

func TestRunLifiEscrowMessageHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(LifiEscrowMessageHandlerTestSuite))
}

func (s *LifiEscrowMessageHandlerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())

	lifiAddresses := make(map[uint64]common.Address)
	lifiAddresses[8453] = common.HexToAddress("0x0000000000000000000000000000000000000010")

	tokens := make(map[uint64]map[string]config.TokenConfig)
	tokens[42161] = make(map[string]config.TokenConfig)
	tokens[42161]["USDC"] = config.TokenConfig{
		Address:  common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"),
		Decimals: 6,
	}
	tokens[8453] = make(map[string]config.TokenConfig)
	tokens[8453]["USDC"] = config.TokenConfig{
		Address:  common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"),
		Decimals: 6,
	}
	tokenStore := config.TokenStore{
		Tokens: tokens,
	}
	confirmations := make(map[uint64]uint64)
	confirmations[1000] = 100
	confirmations[2000] = 200

	var order *lifiTypes.LifiOrder
	_ = json.Unmarshal([]byte(mock.ExpectedLifiResponse), &order)
	s.mockOrder = order

	s.mockOrderPricer = mock_message.NewMockOrderPricer(ctrl)
	s.mockOrderValidator = mock_message.NewMockOrderValidator(ctrl)
	s.mockOrderFetcher = mock_message.NewMockOrderFetcher(ctrl)
	s.mockWatcher = mock_message.NewMockConfirmationWatcher(ctrl)
	s.mockCommunication = mock_communication.NewMockCommunication(ctrl)
	s.mockCoordinator = mock_message.NewMockCoordinator(ctrl)
	s.mockHost = mock_host.NewMockHost(ctrl)
	s.mockHost.EXPECT().ID().Return(peer.ID("")).AnyTimes()
	s.mockFetcher = mock_tss.NewMockSaveDataFetcher(ctrl)
	s.mockFetcher.EXPECT().UnlockKeyshare().AnyTimes()
	s.mockFetcher.EXPECT().LockKeyshare().AnyTimes()
	s.mockFetcher.EXPECT().GetKeyshare().AnyTimes().Return(keyshare.ECDSAKeyshare{}, nil)

	s.mockCommunication.EXPECT().Broadcast(
		gomock.Any(),
		gomock.Any(),
		comm.LifiEscrowMsg,
		fmt.Sprintf("%d-%s", 8453, comm.LifiEscrowSessionID),
	).Return(nil)
	p, _ := pstoremem.NewPeerstore()
	s.mockHost.EXPECT().Peerstore().Return(p)

	s.sigChn = make(chan interface{}, 1)

	s.handler = message.NewLifiEscrowMessageHandler(
		8453,
		common.Address{},
		lifiAddresses,
		s.mockCoordinator,
		s.mockHost,
		s.mockCommunication,
		s.mockFetcher,
		s.mockWatcher,
		tokenStore,
		s.mockOrderFetcher,
		s.mockOrderPricer,
		s.mockOrderValidator,
		s.sigChn,
	)
}

func (s *LifiEscrowMessageHandlerTestSuite) Test_HandleMessage_OrderFetchingFails() {
	errChn := make(chan error, 1)
	ad := &message.LifiEscrowData{
		ErrChn:        errChn,
		Nonce:         big.NewInt(101),
		LiquidityPool: common.HexToAddress("0xe59aaf21c4D9Cf92d9eD4537f4404BA031f83b23"),
		Caller:        common.HexToAddress("0xde526bA5d1ad94cC59D7A79d99A59F607d31A657"),
		BorrowAmount:  big.NewInt(2493365192379644),
		OrderID:       "orderID",
	}

	s.mockOrderFetcher.EXPECT().GetOrder("orderID").Return(nil, fmt.Errorf("error"))

	m := &coreMessage.Message{
		Data:        ad,
		Source:      0,
		Destination: 10,
	}

	prop, err := s.handler.HandleMessage(m)

	s.Nil(prop)
	s.NotNil(err)

	err = <-errChn
	s.NotNil(err)
}

func (s *LifiEscrowMessageHandlerTestSuite) Test_HandleMessage_BorrowAmountExceedsAmount() {
	errChn := make(chan error, 1)
	ad := &message.LifiEscrowData{
		ErrChn:        errChn,
		Nonce:         big.NewInt(101),
		LiquidityPool: common.HexToAddress("0xe59aaf21c4D9Cf92d9eD4537f4404BA031f83b23"),
		BorrowAmount:  big.NewInt(100001),
		OrderID:       "orderID",
	}
	s.mockOrderFetcher.EXPECT().GetOrder("orderID").Return(s.mockOrder, nil)

	m := &coreMessage.Message{
		Data:        ad,
		Source:      0,
		Destination: 10,
	}

	prop, err := s.handler.HandleMessage(m)

	s.Nil(prop)
	s.NotNil(err)

	err = <-errChn
	s.NotNil(err)
}

func (s *LifiEscrowMessageHandlerTestSuite) Test_HandleMessage_ValidationError() {
	errChn := make(chan error, 1)
	ad := &message.LifiEscrowData{
		ErrChn:        errChn,
		Nonce:         big.NewInt(101),
		LiquidityPool: common.HexToAddress("0xe59aaf21c4D9Cf92d9eD4537f4404BA031f83b23"),
		BorrowAmount:  big.NewInt(10000),
		OrderID:       "orderID",
	}
	s.mockOrderFetcher.EXPECT().GetOrder("orderID").Return(s.mockOrder, nil)
	s.mockOrderValidator.EXPECT().Validate(s.mockOrder).Return(fmt.Errorf("error"))

	m := &coreMessage.Message{
		Data:        ad,
		Source:      0,
		Destination: 10,
	}

	prop, err := s.handler.HandleMessage(m)

	s.Nil(prop)
	s.NotNil(err)

	err = <-errChn
	s.NotNil(err)
}

func (s *LifiEscrowMessageHandlerTestSuite) Test_HandleMessage_ValidOrder() {
	errChn := make(chan error, 1)
	ad := &message.LifiEscrowData{
		ErrChn:        errChn,
		Nonce:         big.NewInt(101),
		LiquidityPool: common.HexToAddress("0xe59aaf21c4D9Cf92d9eD4537f4404BA031f83b23"),
		BorrowAmount:  big.NewInt(10000),
		OrderID:       "orderID",
		DepositTxHash: "0xhash",
	}
	s.mockOrderFetcher.EXPECT().GetOrder("orderID").Return(s.mockOrder, nil)
	s.mockOrderValidator.EXPECT().Validate(s.mockOrder).Return(nil)
	s.mockOrderPricer.EXPECT().PriceInputs(gomock.Any()).Return(float64(1000), nil)
	s.mockWatcher.EXPECT().WaitForOrderConfirmations(gomock.Any(), uint64(8453), common.HexToHash(ad.DepositTxHash), float64(1000)).Return(nil)
	s.mockCoordinator.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	m := &coreMessage.Message{
		Data:        ad,
		Source:      0,
		Destination: 10,
	}

	prop, err := s.handler.HandleMessage(m)

	s.Nil(prop)
	s.Nil(err)

	err = <-errChn
	s.Nil(err)
}
