package message_test

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
	"github.com/sprintertech/sprinter-signing/chains/evm"
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

type AcrossMessageHandlerTestSuite struct {
	suite.Suite

	mockCommunication *mock_communication.MockCommunication
	mockCoordinator   *mock_message.MockCoordinator
	mockEventFilterer *mock_message.MockEventFilterer
	mockHost          *mock_host.MockHost
	mockFetcher       *mock_tss.MockSaveDataFetcher
	mockPricer        *mock_message.MockTokenPricer

	handler *message.AcrossMessageHandler
	sigChn  chan interface{}

	validLog []byte
}

func TestRunAcrossMessageHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(AcrossMessageHandlerTestSuite))
}

func (s *AcrossMessageHandlerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())

	s.mockCommunication = mock_communication.NewMockCommunication(ctrl)
	s.mockCoordinator = mock_message.NewMockCoordinator(ctrl)
	s.mockEventFilterer = mock_message.NewMockEventFilterer(ctrl)

	s.mockHost = mock_host.NewMockHost(ctrl)
	s.mockHost.EXPECT().ID().Return(peer.ID("")).AnyTimes()

	s.mockFetcher = mock_tss.NewMockSaveDataFetcher(ctrl)
	s.mockFetcher.EXPECT().UnlockKeyshare().AnyTimes()
	s.mockFetcher.EXPECT().LockKeyshare().AnyTimes()
	s.mockFetcher.EXPECT().GetKeyshare().AnyTimes().Return(keyshare.ECDSAKeyshare{}, nil)

	s.mockPricer = mock_message.NewMockTokenPricer(ctrl)

	pool := common.HexToAddress("0x5c7BCd6E7De5423a257D81B442095A1a6ced35C5")

	s.sigChn = make(chan interface{}, 1)

	// Ethereum: 0x93a9d5e32f5c81cbd17ceb842edc65002e3a79da4efbdc9f1e1f7e97fbcd669b
	s.validLog, _ = hex.DecodeString("000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc200000000000000000000000082af49447d8a07e3bd95bd0d56f35241523fbab100000000000000000000000000000000000000000000000000119baee0ab0400000000000000000000000000000000000000000000000000001199073ea3008d0000000000000000000000000000000000000000000000000000000067bc6e3f0000000000000000000000000000000000000000000000000000000067bc927b00000000000000000000000000000000000000000000000000000000000000000000000000000000000000001886a1eb051c10f20c7386576a6a0716b20b2734000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001400000000000000000000000000000000000000000000000000000000000000000")

	tokens := make(map[string]evm.TokenConfig)
	tokens["ETH"] = evm.TokenConfig{
		Address:  common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"),
		Decimals: 18,
	}
	confirmations := make(map[uint64]uint64)
	confirmations[1000] = 100

	s.handler = message.NewAcrossMessageHandler(
		1,
		s.mockEventFilterer,
		pool,
		s.mockCoordinator,
		s.mockHost,
		s.mockCommunication,
		s.mockFetcher,
		s.mockPricer,
		s.sigChn,
		tokens,
		confirmations,
		time.Millisecond,
	)
}

func (s *AcrossMessageHandlerTestSuite) Test_HandleMessage_FailedLogQuery() {
	s.mockCommunication.EXPECT().Broadcast(
		gomock.Any(),
		gomock.Any(),
		comm.AcrossMsg,
		fmt.Sprintf("%d-%s", 1, comm.AcrossSessionID),
	).Return(nil)
	p, _ := pstoremem.NewPeerstore()
	s.mockHost.EXPECT().Peerstore().Return(p)
	s.mockEventFilterer.EXPECT().LatestBlock().Return(big.NewInt(100), nil)
	s.mockEventFilterer.EXPECT().FilterLogs(gomock.Any(), gomock.Any()).Return([]types.Log{}, fmt.Errorf("error"))

	errChn := make(chan error, 1)
	ad := message.AcrossData{
		ErrChn:        errChn,
		DepositId:     big.NewInt(100),
		LiquidityPool: common.HexToAddress("0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657"),
		Caller:        common.HexToAddress("0xde526bA5d1ad94cC59D7A79d99A59F607d31A657"),
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

func (s *AcrossMessageHandlerTestSuite) Test_HandleMessage_LogMissing() {
	s.mockCommunication.EXPECT().Broadcast(
		gomock.Any(),
		gomock.Any(),
		comm.AcrossMsg,
		fmt.Sprintf("%d-%s", 1, comm.AcrossSessionID),
	).Return(nil)
	p, _ := pstoremem.NewPeerstore()
	s.mockHost.EXPECT().Peerstore().Return(p)
	s.mockEventFilterer.EXPECT().FilterLogs(gomock.Any(), gomock.Any()).Return([]types.Log{}, nil)
	s.mockEventFilterer.EXPECT().LatestBlock().Return(big.NewInt(100), nil)

	errChn := make(chan error, 1)
	ad := message.AcrossData{
		ErrChn:        errChn,
		DepositId:     big.NewInt(100),
		LiquidityPool: common.HexToAddress("0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657"),
		Caller:        common.HexToAddress("0xde526bA5d1ad94cC59D7A79d99A59F607d31A657"),
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

func (s *AcrossMessageHandlerTestSuite) Test_HandleMessage_IgnoreRemovedLogs() {
	s.mockCommunication.EXPECT().Broadcast(
		gomock.Any(),
		gomock.Any(),
		comm.AcrossMsg,
		fmt.Sprintf("%d-%s", 1, comm.AcrossSessionID),
	).Return(nil)
	p, _ := pstoremem.NewPeerstore()
	s.mockHost.EXPECT().Peerstore().Return(p)
	s.mockEventFilterer.EXPECT().LatestBlock().Return(big.NewInt(100), nil)
	s.mockEventFilterer.EXPECT().FilterLogs(gomock.Any(), gomock.Any()).Return([]types.Log{
		{
			Removed: true,
			Data:    s.validLog,
		},
	}, nil)

	errChn := make(chan error, 1)
	ad := message.AcrossData{
		ErrChn:        errChn,
		DepositId:     big.NewInt(100),
		LiquidityPool: common.HexToAddress("0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657"),
		Caller:        common.HexToAddress("0xde526bA5d1ad94cC59D7A79d99A59F607d31A657"),
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

func (s *AcrossMessageHandlerTestSuite) Test_HandleMessage_ValidLog() {
	s.mockCommunication.EXPECT().Broadcast(
		gomock.Any(),
		gomock.Any(),
		comm.AcrossMsg,
		fmt.Sprintf("%d-%s", 1, comm.AcrossSessionID),
	).Return(nil)
	p, _ := pstoremem.NewPeerstore()
	s.mockHost.EXPECT().Peerstore().Return(p)

	s.mockEventFilterer.EXPECT().TransactionReceipt(
		gomock.Any(),
		gomock.Any(),
	).Return(&types.Receipt{}, fmt.Errorf("missing transaction receipt"))
	s.mockEventFilterer.EXPECT().TransactionReceipt(gomock.Any(), gomock.Any()).Return(&types.Receipt{
		BlockNumber: big.NewInt(100),
	}, nil)

	s.mockEventFilterer.EXPECT().LatestBlock().Return(big.NewInt(200), nil).AnyTimes()
	s.mockEventFilterer.EXPECT().FilterLogs(gomock.Any(), gomock.Any()).Return([]types.Log{
		{
			Removed: false,
			Data:    s.validLog,
			Topics: []common.Hash{
				{},
				{},
				{},
				{},
			},
		},
	}, nil)
	s.mockCoordinator.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	s.mockPricer.EXPECT().TokenPrice("ETH").Return(2200.15, nil)

	errChn := make(chan error, 1)
	ad := message.AcrossData{
		ErrChn:        errChn,
		DepositId:     big.NewInt(100),
		LiquidityPool: common.HexToAddress("0xbe526bA5d1ad94cC59D7A79d99A59F607d31A657"),
		Caller:        common.HexToAddress("0x5ECF7351930e4A251193aA022Ef06249C6cBfa27"),
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
}
