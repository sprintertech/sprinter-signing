package message_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
	evmMessage "github.com/sprintertech/sprinter-signing/chains/evm/message"
	"github.com/sprintertech/sprinter-signing/chains/lighter/message"
	mock_message "github.com/sprintertech/sprinter-signing/chains/lighter/message/mock"
	"github.com/sprintertech/sprinter-signing/comm"
	mock_communication "github.com/sprintertech/sprinter-signing/comm/mock"
	mock_host "github.com/sprintertech/sprinter-signing/comm/p2p/mock/host"
	"github.com/sprintertech/sprinter-signing/protocol/lighter"
	"github.com/stretchr/testify/suite"
	coreMessage "github.com/sygmaprotocol/sygma-core/relayer/message"
	"go.uber.org/mock/gomock"
)

type LighterMessageHandlerTestSuite struct {
	suite.Suite

	mockCommunication *mock_communication.MockCommunication
	mockHost          *mock_host.MockHost
	mockTxFetcher     *mock_message.MockTxFetcher

	handler *message.LighterMessageHandler
	sigChn  chan interface{}
	msgChan chan []*coreMessage.Message
}

func TestRunLighterMessageHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(LighterMessageHandlerTestSuite))
}

func (s *LighterMessageHandlerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())

	s.mockCommunication = mock_communication.NewMockCommunication(ctrl)
	s.mockHost = mock_host.NewMockHost(ctrl)
	s.mockHost.EXPECT().ID().Return(peer.ID("")).AnyTimes()

	s.mockTxFetcher = mock_message.NewMockTxFetcher(ctrl)

	s.sigChn = make(chan interface{}, 1)
	s.msgChan = make(chan []*coreMessage.Message, 1)
	confirmations := make(map[uint64]uint64)
	confirmations[200] = 0

	s.handler = message.NewLighterMessageHandler(
		common.Address{},
		common.Address{},
		"3",
		confirmations,
		s.mockTxFetcher,
		s.mockHost,
		s.mockCommunication,
		s.sigChn,
		s.msgChan,
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
		LiquidityPool: "0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657",
		OrderHash:     "orderHash",
		DepositTxHash: "orderHash",
	}

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

	msgs := <-s.msgChan
	s.Len(msgs, 1)
	s.Equal(evmMessage.SignMessageType, msgs[0].Type)
	s.Equal(uint64(42161), msgs[0].Destination)
	req := msgs[0].Data.(*evmMessage.SignRequest)
	s.Equal(new(big.Int).SetUint64(2000001), req.BorrowAmount)
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
		LiquidityPool: "0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657",
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

func (s *LighterMessageHandlerTestSuite) Test_HandleMessage_InvalidAsset() {
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
		LiquidityPool: "0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657",
		OrderHash:     "orderHash",
		DepositTxHash: "orderHash",
	}
	s.mockTxFetcher.EXPECT().GetTx(ad.OrderHash).Return(&lighter.LighterTx{
		Type: lighter.TxTypeL2Transfer,
		Transfer: &lighter.Transfer{
			Amount:         2000001,
			AssetIndex:     2,
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
		LiquidityPool: "0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657",
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

func (s *LighterMessageHandlerTestSuite) Test_HandleMessage_InvalidOrderValue() {
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
		LiquidityPool: "0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657",
		OrderHash:     "orderHash",
		DepositTxHash: "orderHash",
	}
	s.mockTxFetcher.EXPECT().GetTx(ad.OrderHash).Return(&lighter.LighterTx{
		Type: lighter.TxTypeL2Transfer,
		Transfer: &lighter.Transfer{
			Amount:         200000001,
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
		LiquidityPool: "0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657",
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
