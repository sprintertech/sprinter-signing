package message_test

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
	"github.com/sprintertech/sprinter-signing/chains/evm/message"
	mock_message "github.com/sprintertech/sprinter-signing/chains/evm/message/mock"
	"github.com/sprintertech/sprinter-signing/comm"
	mock_communication "github.com/sprintertech/sprinter-signing/comm/mock"
	mock_host "github.com/sprintertech/sprinter-signing/comm/p2p/mock/host"
	"github.com/sprintertech/sprinter-signing/keyshare"
	mock_tss "github.com/sprintertech/sprinter-signing/tss/ecdsa/common/mock"
	"github.com/stretchr/testify/suite"
	coreMessage "github.com/sygmaprotocol/sygma-core/relayer/message"
	"go.uber.org/mock/gomock"
)

type LifiUnlockHandlerTestSuite struct {
	suite.Suite

	mockCommunication *mock_communication.MockCommunication
	mockCoordinator   *mock_message.MockCoordinator
	mockHost          *mock_host.MockHost
	mockFetcher       *mock_tss.MockSaveDataFetcher

	handler *message.LifiUnlockHandler
}

func TestRunLifiUnlockHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(LifiUnlockHandlerTestSuite))
}

func (s *LifiUnlockHandlerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())

	repayers := make(map[uint64]common.Address)
	repayers[10] = common.HexToAddress("0x5c7BCd6E7De5423a257D81B442095A1a6ced35C6")

	s.mockCommunication = mock_communication.NewMockCommunication(ctrl)
	s.mockCoordinator = mock_message.NewMockCoordinator(ctrl)
	s.mockHost = mock_host.NewMockHost(ctrl)
	s.mockHost.EXPECT().ID().Return(peer.ID("")).AnyTimes()
	s.mockFetcher = mock_tss.NewMockSaveDataFetcher(ctrl)
	s.mockFetcher.EXPECT().UnlockKeyshare().AnyTimes()
	s.mockFetcher.EXPECT().LockKeyshare().AnyTimes()
	s.mockFetcher.EXPECT().GetKeyshare().AnyTimes().Return(keyshare.ECDSAKeyshare{}, nil)
	p, _ := pstoremem.NewPeerstore()
	s.mockHost.EXPECT().Peerstore().Return(p)
	s.mockCommunication.EXPECT().Broadcast(
		gomock.Any(),
		gomock.Any(),
		comm.LifiUnlockMsg,
		fmt.Sprintf("%d-%s", 10, comm.LifiUnlockMsg),
	).Return(nil)

	s.handler = message.NewLifiUnlockHandler(
		10,
		repayers,
		s.mockCoordinator,
		s.mockHost,
		s.mockCommunication,
		s.mockFetcher,
	)
}

func (s *LifiUnlockHandlerTestSuite) Test_HandleMessage_ValidMessage() {
	sigChn := make(chan interface{}, 1)
	ad := &message.LifiUnlockData{
		SigChn:  sigChn,
		OrderID: "id",
		Settler: common.HexToAddress("abcd"),
	}
	s.mockCoordinator.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	m := &coreMessage.Message{
		Data:        ad,
		Source:      0,
		Destination: 10,
	}

	prop, err := s.handler.HandleMessage(m)

	s.Nil(prop)
	s.Nil(err)
}
