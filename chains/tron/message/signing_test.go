package message_test

import (
	"math/big"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	evmMessage "github.com/sprintertech/sprinter-signing/chains/evm/message"
	"github.com/sprintertech/sprinter-signing/chains/tron"
	"github.com/sprintertech/sprinter-signing/chains/tron/message"
	mock_tron "github.com/sprintertech/sprinter-signing/chains/tron/message/mock"
	mock_communication "github.com/sprintertech/sprinter-signing/comm/mock"
	mock_host "github.com/sprintertech/sprinter-signing/comm/p2p/mock/host"
	"github.com/sprintertech/sprinter-signing/keyshare"
	mock_tss "github.com/sprintertech/sprinter-signing/tss/ecdsa/common/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type SignHandlerTestSuite struct {
	suite.Suite

	mockCoordinator *mock_tron.MockCoordinator
	mockFetcher     *mock_tss.MockSaveDataFetcher

	handler *message.SignHandler
}

func TestRunSignHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(SignHandlerTestSuite))
}

func (s *SignHandlerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())

	s.mockCoordinator = mock_tron.NewMockCoordinator(ctrl)
	mockHost := mock_host.NewMockHost(ctrl)
	mockCommunication := mock_communication.NewMockCommunication(ctrl)
	s.mockFetcher = mock_tss.NewMockSaveDataFetcher(ctrl)

	s.handler = message.NewSignHandler(s.mockCoordinator, mockHost, mockCommunication, s.mockFetcher)
}

func (s *SignHandlerTestSuite) baseRequest(resultChn chan any, coordinatorID peer.ID) *evmMessage.SignRequest {
	// TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t is Tron's well-known USDT (TRC20)
	// contract address.
	const tronAddress = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
	return &evmMessage.SignRequest{
		Calldata:      []byte("calldata"),
		BorrowAmount:  big.NewInt(1000),
		BorrowToken:   tronAddress,
		Target:        tronAddress,
		Deadline:      1234,
		Caller:        tronAddress,
		LiquidityPool: tronAddress,
		Nonce:         big.NewInt(1),
		SessionID:     "1-orderID",
		Coordinator:   coordinatorID,
		ResultChn:     resultChn,
	}
}

func (s *SignHandlerTestSuite) Test_HandleMessage_ComputesHashAndRunsSigningSession() {
	s.mockFetcher.EXPECT().LockKeyshare()
	s.mockFetcher.EXPECT().UnlockKeyshare()
	s.mockFetcher.EXPECT().GetKeyshare().Return(keyshare.ECDSAKeyshare{}, nil)

	resultChn := make(chan any, 1)
	coordinatorID := peer.ID("coordinator")
	s.mockCoordinator.EXPECT().
		Execute(gomock.Any(), gomock.Any(), gomock.Eq(resultChn), gomock.Eq(coordinatorID)).
		Return(nil)

	req := s.baseRequest(resultChn, coordinatorID)
	m := evmMessage.NewSignMessage(0, tron.ChainID, req)

	prop, err := s.handler.HandleMessage(m)

	s.Nil(prop)
	s.Nil(err)
}

func (s *SignHandlerTestSuite) Test_HandleMessage_InvalidAddress() {
	req := s.baseRequest(make(chan any, 1), peer.ID(""))
	req.BorrowToken = "not-a-valid-tron-address"
	m := evmMessage.NewSignMessage(0, tron.ChainID, req)

	prop, err := s.handler.HandleMessage(m)

	s.Nil(prop)
	s.NotNil(err)
}
