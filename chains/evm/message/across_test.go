package message_test

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
	"github.com/sprintertech/sprinter-signing/chains/evm/message"
	mock_message "github.com/sprintertech/sprinter-signing/chains/evm/message/mock"
	"github.com/sprintertech/sprinter-signing/comm"
	mock_communication "github.com/sprintertech/sprinter-signing/comm/mock"
	mock_host "github.com/sprintertech/sprinter-signing/comm/p2p/mock/host"
	"github.com/sprintertech/sprinter-signing/config"
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
	mockWatcher       *mock_message.MockConfirmationWatcher
	mockMatcher       *mock_message.MockTokenMatcher

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

	s.mockWatcher = mock_message.NewMockConfirmationWatcher(ctrl)
	s.mockMatcher = mock_message.NewMockTokenMatcher(ctrl)

	pools := make(map[uint64]common.Address)
	pools[2] = common.HexToAddress("0x5c7BCd6E7De5423a257D81B442095A1a6ced35C5")

	s.sigChn = make(chan interface{}, 1)

	// Ethereum: 0x93a9d5e32f5c81cbd17ceb842edc65002e3a79da4efbdc9f1e1f7e97fbcd669b
	s.validLog, _ = hex.DecodeString("000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc200000000000000000000000082af49447d8a07e3bd95bd0d56f35241523fbab100000000000000000000000000000000000000000000000000119baee0ab0400000000000000000000000000000000000000000000000000001199073ea3008d0000000000000000000000000000000000000000000000000000000067bc6e3f0000000000000000000000000000000000000000000000000000000067bc927b00000000000000000000000000000000000000000000000000000000000000000000000000000000000000001886a1eb051c10f20c7386576a6a0716b20b2734000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001400000000000000000000000000000000000000000000000000000000000000000")

	tokens := make(map[uint64]map[string]config.TokenConfig)
	tokens[1] = make(map[string]config.TokenConfig)
	tokens[1]["ETH"] = config.TokenConfig{
		Address:  common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"),
		Decimals: 18,
	}
	tokens[1]["USDC"] = config.TokenConfig{
		Address:  common.HexToAddress("0x3355df6d4c9c3035724fd0e3914de96a5a83aaf4"),
		Decimals: 6,
	}
	tokenStore := config.TokenStore{
		Tokens: tokens,
	}
	confirmations := make(map[uint64]uint64)
	confirmations[1000] = 100
	confirmations[2000] = 200

	s.handler = message.NewAcrossMessageHandler(
		1,
		s.mockEventFilterer,
		pools,
		s.mockCoordinator,
		s.mockHost,
		s.mockCommunication,
		s.mockFetcher,
		s.mockMatcher,
		tokenStore,
		s.mockWatcher,
		s.sigChn,
	)
}

func (s *AcrossMessageHandlerTestSuite) Test_HandleMessage_FailedTransactionQuery() {
	s.mockCommunication.EXPECT().Broadcast(
		gomock.Any(),
		gomock.Any(),
		comm.AcrossMsg,
		fmt.Sprintf("%d-%s", 1, comm.AcrossSessionID),
	).Return(nil)
	p, _ := pstoremem.NewPeerstore()
	s.mockHost.EXPECT().Peerstore().Return(p)
	s.mockEventFilterer.EXPECT().TransactionReceipt(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error"))

	errChn := make(chan error, 1)
	ad := &message.AcrossData{
		ErrChn:        errChn,
		DepositId:     big.NewInt(100),
		Nonce:         big.NewInt(101),
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
	s.mockEventFilterer.EXPECT().TransactionReceipt(gomock.Any(), gomock.Any()).Return(&types.Receipt{
		Logs: []*types.Log{},
	}, nil)

	errChn := make(chan error, 1)
	ad := &message.AcrossData{
		ErrChn:        errChn,
		DepositId:     big.NewInt(100),
		Nonce:         big.NewInt(101),
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
	s.mockEventFilterer.EXPECT().TransactionReceipt(gomock.Any(), gomock.Any()).Return(&types.Receipt{
		Logs: []*types.Log{
			{
				Removed: true,
				Data:    s.validLog,
			},
		},
	}, nil)

	errChn := make(chan error, 1)
	ad := &message.AcrossData{
		ErrChn:        errChn,
		DepositId:     big.NewInt(100),
		Nonce:         big.NewInt(101),
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

	s.mockEventFilterer.EXPECT().TransactionReceipt(gomock.Any(), gomock.Any()).Return(&types.Receipt{
		Logs: []*types.Log{
			{
				Removed: false,
				Data:    s.validLog,
				Topics: []common.Hash{
					common.HexToHash("0x32ed1a409ef04c7b0227189c3a103dc5ac10e775a15b785dcc510201f7c25ad3"),
					{},
					common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000279995"),
					{},
				},
			},
		},
	}, nil)

	s.mockWatcher.EXPECT().WaitForConfirmations(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	s.mockCoordinator.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	errChn := make(chan error, 1)
	ad := &message.AcrossData{
		ErrChn:        errChn,
		DepositId:     big.NewInt(2595221),
		Nonce:         big.NewInt(101),
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

func (s *AcrossMessageHandlerTestSuite) Test_HandleMessage_ZeroOutputToken() {
	s.mockCommunication.EXPECT().Broadcast(
		gomock.Any(),
		gomock.Any(),
		comm.AcrossMsg,
		fmt.Sprintf("%d-%s", 1, comm.AcrossSessionID),
	).Return(nil)
	p, _ := pstoremem.NewPeerstore()
	s.mockHost.EXPECT().Peerstore().Return(p)

	log, _ := hex.DecodeString("0000000000000000000000003355df6d4c9c3035724fd0e3914de96a5a83aaf40000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000006d7cbe22000000000000000000000000000000000000000000000000000000006d789ac90000000000000000000000000000000000000000000000000000000067ce09230000000000000000000000000000000000000000000000000000000067ce5ea7000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000051d55999c7cd91b17af7276cbecd647dbc000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001400000000000000000000000000000000000000000000000000000000000000000")
	s.mockEventFilterer.EXPECT().TransactionReceipt(gomock.Any(), gomock.Any()).Return(&types.Receipt{
		Logs: []*types.Log{
			{
				Removed: false,
				Data:    log,
				Topics: []common.Hash{
					common.HexToHash("0x32ed1a409ef04c7b0227189c3a103dc5ac10e775a15b785dcc510201f7c25ad3"),
					{},
					common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000279995"),
					{},
				},
			},
		},
	}, nil)
	s.mockWatcher.EXPECT().WaitForConfirmations(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	s.mockMatcher.EXPECT().DestinationToken(gomock.Any(), "USDC").Return(common.Address{}, nil)
	s.mockCoordinator.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	errChn := make(chan error, 1)
	ad := &message.AcrossData{
		ErrChn:        errChn,
		DepositId:     big.NewInt(2595221),
		Nonce:         big.NewInt(101),
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
