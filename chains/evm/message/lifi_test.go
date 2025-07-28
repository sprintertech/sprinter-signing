package message_test

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/contracts"
	"github.com/sprintertech/sprinter-signing/chains/evm/message"
	mock_message "github.com/sprintertech/sprinter-signing/chains/evm/message/mock"
	"github.com/sprintertech/sprinter-signing/comm"
	mock_communication "github.com/sprintertech/sprinter-signing/comm/mock"
	mock_host "github.com/sprintertech/sprinter-signing/comm/p2p/mock/host"
	"github.com/sprintertech/sprinter-signing/config"
	"github.com/sprintertech/sprinter-signing/keyshare"
	"github.com/sprintertech/sprinter-signing/protocol/lifi"
	"github.com/sprintertech/sprinter-signing/protocol/lifi/mock"
	mock_tss "github.com/sprintertech/sprinter-signing/tss/ecdsa/common/mock"
	"github.com/stretchr/testify/suite"
	"github.com/sygmaprotocol/sygma-core/crypto/secp256k1"
	coreMessage "github.com/sygmaprotocol/sygma-core/relayer/message"
	"go.uber.org/mock/gomock"
)

type LifiCompactMessageHandlerTestSuite struct {
	suite.Suite

	mockCommunication *mock_communication.MockCommunication
	mockCoordinator   *mock_message.MockCoordinator
	mockHost          *mock_host.MockHost
	mockFetcher       *mock_tss.MockSaveDataFetcher
	mockOrder         *lifi.LifiOrder
	sigChn            chan interface{}

	sponsor   common.Address
	allocator common.Address

	mockOrderFetcher *mock_message.MockOrderFetcher
	mockCompact      *mock_message.MockCompact
	handler          *message.LifiCompactMessageHandler
}

func TestRunLifiCompactMessageHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(LifiCompactMessageHandlerTestSuite))
}

func (s *LifiCompactMessageHandlerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())

	s.mockCommunication = mock_communication.NewMockCommunication(ctrl)
	s.mockCoordinator = mock_message.NewMockCoordinator(ctrl)

	s.mockHost = mock_host.NewMockHost(ctrl)
	s.mockHost.EXPECT().ID().Return(peer.ID("")).AnyTimes()

	s.mockFetcher = mock_tss.NewMockSaveDataFetcher(ctrl)
	s.mockFetcher.EXPECT().UnlockKeyshare().AnyTimes()
	s.mockFetcher.EXPECT().LockKeyshare().AnyTimes()
	s.mockFetcher.EXPECT().GetKeyshare().AnyTimes().Return(keyshare.ECDSAKeyshare{}, nil)

	s.mockOrderFetcher = mock_message.NewMockOrderFetcher(ctrl)
	s.mockCompact = mock_message.NewMockCompact(ctrl)

	s.sigChn = make(chan interface{}, 1)

	lifiAddresses := make(map[uint64]common.Address)
	lifiAddresses[8453] = common.HexToAddress("0x0000000000000000000000000000000000000010")

	tokens := make(map[uint64]map[string]config.TokenConfig)
	tokens[42161] = make(map[string]config.TokenConfig)
	tokens[42161]["WETH"] = config.TokenConfig{
		Address:  common.HexToAddress("0x036CbD53842c5426634e7929541eC2318f3dCF7e"),
		Decimals: 18,
	}
	tokens[8453] = make(map[string]config.TokenConfig)
	tokens[8453]["WETH"] = config.TokenConfig{
		Address:  common.HexToAddress("0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238"),
		Decimals: 18,
	}
	tokenStore := config.TokenStore{
		Tokens: tokens,
	}
	confirmations := make(map[uint64]uint64)
	confirmations[1000] = 100
	confirmations[2000] = 200

	s.mockCommunication.EXPECT().Broadcast(
		gomock.Any(),
		gomock.Any(),
		comm.LifiMsg,
		fmt.Sprintf("%d-%s", 8453, comm.LifiSessionID),
	).Return(nil)
	p, _ := pstoremem.NewPeerstore()
	s.mockHost.EXPECT().Peerstore().Return(p)

	var order *lifi.LifiOrder
	_ = json.Unmarshal([]byte(mock.ExpectedLifiResponse), &order)
	s.mockOrder = order
	s.mockOrder.Order.FillDeadline = time.Now().Add(time.Minute * 10).Unix()
	s.mockOrder.Order.Expires = time.Now().Add(time.Hour * 2).Unix()

	sponsorKp, _ := secp256k1.GenerateKeypair()
	s.sponsor = common.HexToAddress(sponsorKp.Address())

	allocatorKp, _ := secp256k1.GenerateKeypair()
	s.allocator = common.HexToAddress(allocatorKp.Address())
	s.mockOrder.Order.User = s.sponsor.Hex()
	s.mockOrder.Order.Outputs[0].Settler = common.BytesToHash(s.sponsor.Bytes()).Hex()

	s.mockCompact.EXPECT().Address().Return(
		common.HexToAddress("0x0000000000000000000000000000000000000020"),
	).AnyTimes()

	digest, _, _ := lifi.GenerateCompactDigest(big.NewInt(8453), s.mockCompact.Address(), *s.mockOrder)
	sponsorSig, _ := sponsorKp.Sign(digest)
	allocatorSig, _ := allocatorKp.Sign(digest)

	s.mockOrder.AllocatorSignature = "0x" + hex.EncodeToString(allocatorSig)
	s.mockOrder.SponsorSignature = "0x" + hex.EncodeToString(sponsorSig)

	s.handler = message.NewLifiCompactMessageHandler(
		8453,
		common.HexToAddress("0x0000000000000000000000000000000000000001"),
		lifiAddresses,
		tokenStore,
		s.mockOrderFetcher,
		s.mockCompact,
		s.mockCoordinator,
		s.mockHost,
		s.mockCommunication,
		s.mockFetcher,
		s.sigChn,
	)
}

func (s *LifiCompactMessageHandlerTestSuite) Test_HandleMessage_OrderFetchingFails() {
	errChn := make(chan error, 1)
	ad := &message.LifiData{
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

func (s *LifiCompactMessageHandlerTestSuite) Test_HandleMessage_InvalidFillDeadline() {
	errChn := make(chan error, 1)
	ad := &message.LifiData{
		ErrChn:        errChn,
		Nonce:         big.NewInt(101),
		LiquidityPool: common.HexToAddress("0xe59aaf21c4D9Cf92d9eD4537f4404BA031f83b23"),
		Caller:        common.HexToAddress("0xde526bA5d1ad94cC59D7A79d99A59F607d31A657"),
		BorrowAmount:  big.NewInt(2493365192379644),
		OrderID:       "orderID",
	}

	s.mockOrder.Order.FillDeadline = time.Now().Unix()
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

func (s *LifiCompactMessageHandlerTestSuite) Test_HandleMessage_InvalidExpiry() {
	errChn := make(chan error, 1)
	ad := &message.LifiData{
		ErrChn:        errChn,
		Nonce:         big.NewInt(101),
		LiquidityPool: common.HexToAddress("0xe59aaf21c4D9Cf92d9eD4537f4404BA031f83b23"),
		Caller:        common.HexToAddress("0xde526bA5d1ad94cC59D7A79d99A59F607d31A657"),
		BorrowAmount:  big.NewInt(2493365192379644),
		OrderID:       "orderID",
	}

	s.mockOrder.Order.Expires = time.Now().Unix()
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

func (s *LifiCompactMessageHandlerTestSuite) Test_HandleMessage_OrderInputsNotWhitelisted() {
	errChn := make(chan error, 1)
	ad := &message.LifiData{
		ErrChn:        errChn,
		Nonce:         big.NewInt(101),
		LiquidityPool: common.HexToAddress("0xe59aaf21c4D9Cf92d9eD4537f4404BA031f83b23"),
		Caller:        s.sponsor,
		BorrowAmount:  big.NewInt(2493365192379644),
		OrderID:       "orderID",
	}

	s.mockOrder.Order.Inputs[0][0] = &lifi.BigInt{Int: big.NewInt(0)}
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

func (s *LifiCompactMessageHandlerTestSuite) Test_HandleMessage_OrderInvalidWithdrawalStatus() {
	errChn := make(chan error, 1)
	ad := &message.LifiData{
		ErrChn:        errChn,
		Nonce:         big.NewInt(101),
		LiquidityPool: common.HexToAddress("0xe59aaf21c4D9Cf92d9eD4537f4404BA031f83b23"),
		Caller:        s.sponsor,
		BorrowAmount:  big.NewInt(2493365192379644),
		OrderID:       "orderID",
	}

	s.mockCompact.EXPECT().GetForcedWithdrawalStatus(
		common.HexToAddress(s.mockOrder.Order.User),
		gomock.Any(),
	).Return(contracts.STATUS_PENDING, nil)
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

func (s *LifiCompactMessageHandlerTestSuite) Test_HandleMessage_OrderInvalidOutputs() {
	errChn := make(chan error, 1)
	ad := &message.LifiData{
		ErrChn:        errChn,
		Nonce:         big.NewInt(101),
		LiquidityPool: common.HexToAddress("0xe59aaf21c4D9Cf92d9eD4537f4404BA031f83b23"),
		Caller:        s.sponsor,
		BorrowAmount:  big.NewInt(2493365192379644),
		OrderID:       "orderID",
	}

	s.mockCompact.EXPECT().GetForcedWithdrawalStatus(
		common.HexToAddress(s.mockOrder.Order.User),
		gomock.Any(),
	).Return(contracts.STATUS_DISABLED, nil)
	s.mockOrder.Order.Outputs[0].Token = "0x000000000000000000000000036CbD53842c5426634e7929541eC2318f3dCF7b"
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

func (s *LifiCompactMessageHandlerTestSuite) Test_HandleMessage_BorrowAmountExceedsAmount() {
	errChn := make(chan error, 1)
	ad := &message.LifiData{
		ErrChn:        errChn,
		Nonce:         big.NewInt(101),
		LiquidityPool: common.HexToAddress("0xe59aaf21c4D9Cf92d9eD4537f4404BA031f83b23"),
		Caller:        s.sponsor,
		BorrowAmount:  big.NewInt(10001),
		OrderID:       "orderID",
	}

	s.mockCompact.EXPECT().GetForcedWithdrawalStatus(
		common.HexToAddress(s.mockOrder.Order.User),
		gomock.Any(),
	).Return(contracts.STATUS_DISABLED, nil)
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

func (s *LifiCompactMessageHandlerTestSuite) Test_HandleMessage_InvalidSignature() {
	errChn := make(chan error, 1)
	ad := &message.LifiData{
		ErrChn:        errChn,
		Nonce:         big.NewInt(101),
		LiquidityPool: common.HexToAddress("0xe59aaf21c4D9Cf92d9eD4537f4404BA031f83b23"),
		Caller:        s.sponsor,
		BorrowAmount:  big.NewInt(9999),
		OrderID:       "orderID",
	}

	s.mockCompact.EXPECT().GetForcedWithdrawalStatus(
		gomock.Any(),
		gomock.Any(),
	).Return(contracts.STATUS_DISABLED, nil)
	s.mockOrder.Order.User = "0xinvalid"
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

func (s *LifiCompactMessageHandlerTestSuite) Test_HandleMessage_ValidOrder() {
	errChn := make(chan error, 1)
	ad := &message.LifiData{
		ErrChn:        errChn,
		Nonce:         big.NewInt(101),
		LiquidityPool: common.HexToAddress("0xe59aaf21c4D9Cf92d9eD4537f4404BA031f83b23"),
		Caller:        s.sponsor,
		BorrowAmount:  big.NewInt(9999),
		OrderID:       "orderID",
	}

	s.mockOrder.Order.User = s.sponsor.Hex()
	s.mockCompact.EXPECT().GetForcedWithdrawalStatus(
		common.HexToAddress(s.mockOrder.Order.User),
		gomock.Any(),
	).Return(contracts.STATUS_DISABLED, nil)
	s.mockCompact.EXPECT().Allocator(gomock.Any()).Return(s.allocator, nil)
	s.mockCompact.EXPECT().HasConsumedAllocatorNonce(s.allocator, gomock.Any()).Return(false, nil)
	s.mockOrderFetcher.EXPECT().GetOrder("orderID").Return(s.mockOrder, nil)
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
