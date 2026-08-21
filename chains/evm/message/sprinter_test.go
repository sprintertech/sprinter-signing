package message_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
	"github.com/sprintertech/sprinter-signing/chains/evm/message"
	"github.com/sprintertech/sprinter-signing/comm"
	mock_communication "github.com/sprintertech/sprinter-signing/comm/mock"
	mock_host "github.com/sprintertech/sprinter-signing/comm/p2p/mock/host"
	"github.com/stretchr/testify/suite"
	coreMessage "github.com/sygmaprotocol/sygma-core/relayer/message"
	"go.uber.org/mock/gomock"
)

type SprinterCreditMessageHandlerTestSuite struct {
	suite.Suite

	mockCommunication *mock_communication.MockCommunication
	mockHost          *mock_host.MockHost

	handler *message.SprinterCreditMessageHandler
	sigChn  chan interface{}
	msgChan chan []*coreMessage.Message
}

func TestRunSprinterCreditMessageHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(SprinterCreditMessageHandlerTestSuite))
}

func (s *SprinterCreditMessageHandlerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())

	s.mockCommunication = mock_communication.NewMockCommunication(ctrl)
	s.mockHost = mock_host.NewMockHost(ctrl)
	s.mockHost.EXPECT().ID().Return(peer.ID("")).AnyTimes()
	s.sigChn = make(chan interface{}, 1)
	s.msgChan = make(chan []*coreMessage.Message, 1)

	liquidators := make(map[common.Address]common.Address)
	token := common.HexToAddress("0x0000000000000000000000000000000000000001")
	liquidator := common.HexToAddress("0x0000000000000000000000000000000000000002")
	liquidators[token] = liquidator

	s.handler = message.NewSprinterCreditMessageHandler(
		1,
		liquidators,
		s.mockHost,
		s.mockCommunication,
		s.sigChn,
		s.msgChan,
	)
}

func (s *SprinterCreditMessageHandlerTestSuite) Test_HandleMessage_InvalidToken() {
	s.mockCommunication.EXPECT().Broadcast(
		gomock.Any(),
		gomock.Any(),
		comm.SprinterCreditMsg,
		fmt.Sprintf("%d-%s", 1, comm.SprinterCreditSessionID),
	).Return(nil)
	p, _ := pstoremem.NewPeerstore()
	s.mockHost.EXPECT().Peerstore().Return(p)

	errChn := make(chan error, 1)
	ad := &message.SprinterCreditData{
		ErrChn:        errChn,
		Nonce:         big.NewInt(101),
		LiquidityPool: "0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657",
		Caller:        "0x5ECF7351930e4A251193aA022Ef06249C6cBfa27",
		BorrowAmount:  big.NewInt(150),
		TokenOut:      "0x0000000000000000000000000000000000000002",
	}
	m := &coreMessage.Message{
		Data:        ad,
		Source:      1,
		Destination: 2,
	}

	prop, err := s.handler.HandleMessage(m)

	s.Nil(prop)
	s.NotNil(err)

	err = <-errChn
	s.NotNil(err)
}

func (s *SprinterCreditMessageHandlerTestSuite) Test_HandleMessage_ValidLiquidation() {
	s.mockCommunication.EXPECT().Broadcast(
		gomock.Any(),
		gomock.Any(),
		comm.SprinterCreditMsg,
		fmt.Sprintf("%d-%s", 1, comm.SprinterCreditSessionID),
	).Return(nil)
	p, _ := pstoremem.NewPeerstore()
	s.mockHost.EXPECT().Peerstore().Return(p)

	errChn := make(chan error, 1)
	ad := &message.SprinterCreditData{
		ErrChn:        errChn,
		Nonce:         big.NewInt(101),
		LiquidityPool: "0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657",
		Caller:        "0x5ECF7351930e4A251193aA022Ef06249C6cBfa27",
		BorrowAmount:  big.NewInt(150),
		TokenOut:      "0x0000000000000000000000000000000000000001",
	}
	m := &coreMessage.Message{
		Data:        ad,
		Source:      1,
		Destination: 2,
	}

	prop, err := s.handler.HandleMessage(m)

	s.Nil(prop)
	s.Nil(err)

	err = <-errChn
	s.Nil(err)

	msgs := <-s.msgChan
	s.Len(msgs, 1)
	s.Equal(message.SignMessageType, msgs[0].Type)
	s.Equal(uint64(1), msgs[0].Destination)
	req := msgs[0].Data.(*message.SignRequest)
	s.Equal(common.HexToAddress("0x0000000000000000000000000000000000000002").Hex(), req.Target)
}
