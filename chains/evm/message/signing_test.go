package message_test

import (
	"math/big"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	evmMessage "github.com/sprintertech/sprinter-signing/chains/evm/message"
	mock_message "github.com/sprintertech/sprinter-signing/chains/evm/message/mock"
	mock_communication "github.com/sprintertech/sprinter-signing/comm/mock"
	mock_host "github.com/sprintertech/sprinter-signing/comm/p2p/mock/host"
	"github.com/sprintertech/sprinter-signing/keyshare"
	mock_tss "github.com/sprintertech/sprinter-signing/tss/ecdsa/common/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type SignHandlerTestSuite struct {
	suite.Suite

	mockCoordinator *mock_message.MockCoordinator
	mockFetcher     *mock_tss.MockSaveDataFetcher

	handler *evmMessage.SignHandler
}

func TestRunSignHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(SignHandlerTestSuite))
}

func (s *SignHandlerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())

	s.mockCoordinator = mock_message.NewMockCoordinator(ctrl)
	mockHost := mock_host.NewMockHost(ctrl)
	mockCommunication := mock_communication.NewMockCommunication(ctrl)
	s.mockFetcher = mock_tss.NewMockSaveDataFetcher(ctrl)

	s.handler = evmMessage.NewSignHandler(s.mockCoordinator, mockHost, mockCommunication, s.mockFetcher)
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

	req := &evmMessage.SignRequest{
		Calldata:      []byte("calldata"),
		BorrowAmount:  big.NewInt(1000),
		BorrowToken:   "0x0000000000000000000000000000000000000001",
		Target:        "0x0000000000000000000000000000000000000002",
		Deadline:      1234,
		Caller:        "0x0000000000000000000000000000000000000003",
		LiquidityPool: "0x0000000000000000000000000000000000000004",
		Nonce:         big.NewInt(1),
		SessionID:     "1-orderID",
		Coordinator:   coordinatorID,
		ResultChn:     resultChn,
	}
	m := evmMessage.NewSignMessage(0, 137, req)

	prop, err := s.handler.HandleMessage(m)

	s.Nil(prop)
	s.Nil(err)
}
