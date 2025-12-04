package message_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
	"github.com/sprintertech/sprinter-signing/chains/lighter/message"
	mock_message "github.com/sprintertech/sprinter-signing/chains/lighter/message/mock"
	"github.com/sprintertech/sprinter-signing/comm"
	mock_communication "github.com/sprintertech/sprinter-signing/comm/mock"
	mock_host "github.com/sprintertech/sprinter-signing/comm/p2p/mock/host"
	"github.com/sprintertech/sprinter-signing/keyshare"
	"github.com/sprintertech/sprinter-signing/protocol/lighter"
	mock_tss "github.com/sprintertech/sprinter-signing/tss/ecdsa/common/mock"
	"github.com/stretchr/testify/suite"
	coreMessage "github.com/sygmaprotocol/sygma-core/relayer/message"
	"go.uber.org/mock/gomock"
)

type LighterMessageHandlerTestSuite struct {
	suite.Suite

	mockCommunication *mock_communication.MockCommunication
	mockCoordinator   *mock_message.MockCoordinator
	mockHost          *mock_host.MockHost
	mockFetcher       *mock_tss.MockSaveDataFetcher
	mockTxFetcher     *mock_message.MockTxFetcher

	handler *message.LighterMessageHandler
	sigChn  chan interface{}
}

func TestRunLighterMessageHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(LighterMessageHandlerTestSuite))
}

func (s *LighterMessageHandlerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())

	s.mockCommunication = mock_communication.NewMockCommunication(ctrl)
	s.mockCoordinator = mock_message.NewMockCoordinator(ctrl)
	s.mockHost = mock_host.NewMockHost(ctrl)
	s.mockHost.EXPECT().ID().Return(peer.ID("")).AnyTimes()

	s.mockFetcher = mock_tss.NewMockSaveDataFetcher(ctrl)
	s.mockFetcher.EXPECT().UnlockKeyshare().AnyTimes()
	s.mockFetcher.EXPECT().LockKeyshare().AnyTimes()
	s.mockFetcher.EXPECT().GetKeyshare().AnyTimes().Return(keyshare.ECDSAKeyshare{}, nil)

	s.mockTxFetcher = mock_message.NewMockTxFetcher(ctrl)

	s.sigChn = make(chan interface{}, 1)

	s.handler = message.NewLighterMessageHandler(
		common.Address{},
		common.Address{},
		"3",
		s.mockTxFetcher,
		s.mockCoordinator,
		s.mockHost,
		s.mockCommunication,
		s.mockFetcher,
		s.sigChn,
	)
}

func (s *LighterMessageHandlerTestSuite) Test_HandleMessage_ValidMessage() {
	s.mockCommunication.EXPECT().Broadcast(
		gomock.Any(),
		gomock.Any(),
		comm.LighterMsg,
		"lighter",
	).Return(nil)
	p, _ := pstoremem.NewPeerstore()
	s.mockHost.EXPECT().Peerstore().Return(p)

	errChn := make(chan error, 1)
	ad := &message.LighterData{
		ErrChn:        errChn,
		Nonce:         big.NewInt(101),
		LiquidityPool: common.HexToAddress("0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657"),
		OrderHash:     "orderHash",
		DepositTxHash: "orderHash",
	}

	s.mockCoordinator.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	s.mockTxFetcher.EXPECT().GetTx(ad.OrderHash).Return(&lighter.LighterTx{
		Type: lighter.TxTypeL2Transfer,
		Transfer: &lighter.Transfer{
			Amount:         2000001,
			AssetIndex:     3,
			ToAccountIndex: 3,
			Memo:           []byte{238, 123, 250, 212, 202, 237, 62, 98, 106, 248, 169, 199, 213, 3, 76, 213, 137, 238, 73, 144, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
	}, nil)

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

func (s *LighterMessageHandlerTestSuite) Test_HandleMessage_FeeHigherThanAmount() {
	s.mockCommunication.EXPECT().Broadcast(
		gomock.Any(),
		gomock.Any(),
		comm.LighterMsg,
		"lighter",
	).Return(nil)
	p, _ := pstoremem.NewPeerstore()
	s.mockHost.EXPECT().Peerstore().Return(p)

	errChn := make(chan error, 1)
	ad := &message.LighterData{
		ErrChn:        errChn,
		Nonce:         big.NewInt(101),
		LiquidityPool: common.HexToAddress("0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657"),
		OrderHash:     "orderHash",
		DepositTxHash: "orderHash",
	}

	s.mockTxFetcher.EXPECT().GetTx(ad.OrderHash).Return(&lighter.LighterTx{
		Type: lighter.TxTypeL2Transfer,
		Transfer: &lighter.Transfer{
			Amount:         2000000,
			AssetIndex:     3,
			ToAccountIndex: 3,
			Memo:           []byte{238, 123, 250, 212, 202, 237, 62, 98, 106, 248, 169, 199, 213, 3, 76, 213, 137, 238, 73, 144, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
	}, nil)

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

func (s *LighterMessageHandlerTestSuite) Test_HandleMessage_InvalidTxType() {
	s.mockCommunication.EXPECT().Broadcast(
		gomock.Any(),
		gomock.Any(),
		comm.LighterMsg,
		"lighter",
	).Return(nil)
	p, _ := pstoremem.NewPeerstore()
	s.mockHost.EXPECT().Peerstore().Return(p)

	errChn := make(chan error, 1)
	ad := &message.LighterData{
		ErrChn:        errChn,
		Nonce:         big.NewInt(101),
		LiquidityPool: common.HexToAddress("0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657"),
		OrderHash:     "orderHash",
		DepositTxHash: "orderHash",
	}
	s.mockTxFetcher.EXPECT().GetTx(ad.OrderHash).Return(&lighter.LighterTx{
		Type: lighter.TxTypeL2Withdraw,
		Transfer: &lighter.Transfer{
			Amount:         2000001,
			AssetIndex:     3,
			ToAccountIndex: 3,
			Memo:           []byte{238, 123, 250, 212, 202, 237, 62, 98, 106, 248, 169, 199, 213, 3, 76, 213, 137, 238, 73, 144, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
	}, nil)

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

func (s *LighterMessageHandlerTestSuite) Test_HandleMessage_InvalidAccount() {
	s.mockCommunication.EXPECT().Broadcast(
		gomock.Any(),
		gomock.Any(),
		comm.LighterMsg,
		"lighter",
	).Return(nil)
	p, _ := pstoremem.NewPeerstore()
	s.mockHost.EXPECT().Peerstore().Return(p)

	errChn := make(chan error, 1)
	ad := &message.LighterData{
		ErrChn:        errChn,
		Nonce:         big.NewInt(101),
		LiquidityPool: common.HexToAddress("0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657"),
		OrderHash:     "orderHash",
		DepositTxHash: "orderHash",
	}
	s.mockTxFetcher.EXPECT().GetTx(ad.OrderHash).Return(&lighter.LighterTx{
		Type: lighter.TxTypeL2Transfer,
		Transfer: &lighter.Transfer{
			Amount:         2000001,
			AssetIndex:     3,
			ToAccountIndex: 5,
			Memo:           []byte{238, 123, 250, 212, 202, 237, 62, 98, 106, 248, 169, 199, 213, 3, 76, 213, 137, 238, 73, 144, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
	}, nil)

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

func (s *LighterMessageHandlerTestSuite) Test_HandleMessage_MissingTx() {
	s.mockCommunication.EXPECT().Broadcast(
		gomock.Any(),
		gomock.Any(),
		comm.LighterMsg,
		"lighter",
	).Return(nil)
	p, _ := pstoremem.NewPeerstore()
	s.mockHost.EXPECT().Peerstore().Return(p)

	errChn := make(chan error, 1)
	ad := &message.LighterData{
		ErrChn:        errChn,
		Nonce:         big.NewInt(101),
		LiquidityPool: common.HexToAddress("0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657"),
		OrderHash:     "orderHash",
		DepositTxHash: "orderHash",
	}
	s.mockTxFetcher.EXPECT().GetTx(ad.OrderHash).Return(nil, fmt.Errorf("not found"))

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
