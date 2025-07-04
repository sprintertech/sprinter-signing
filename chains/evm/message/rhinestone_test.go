package message_test

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

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
	"github.com/sprintertech/sprinter-signing/protocol/rhinestone"
	mock_tss "github.com/sprintertech/sprinter-signing/tss/ecdsa/common/mock"
	"github.com/stretchr/testify/suite"
	coreMessage "github.com/sygmaprotocol/sygma-core/relayer/message"
	"go.uber.org/mock/gomock"
)

type RhinestoneMessageHandlerTestSuite struct {
	suite.Suite

	mockCommunication *mock_communication.MockCommunication
	mockCoordinator   *mock_message.MockCoordinator
	mockHost          *mock_host.MockHost
	mockFetcher       *mock_tss.MockSaveDataFetcher
	sigChn            chan interface{}

	mockBundleFetcher *mock_message.MockBundleFetcher
	mockBundle        *rhinestone.Bundle
	handler           *message.RhinestoneMessageHandler
}

func TestRunRhinestoneMessageHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(RhinestoneMessageHandlerTestSuite))
}

func (s *RhinestoneMessageHandlerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())

	s.mockCommunication = mock_communication.NewMockCommunication(ctrl)
	s.mockCoordinator = mock_message.NewMockCoordinator(ctrl)

	s.mockHost = mock_host.NewMockHost(ctrl)
	s.mockHost.EXPECT().ID().Return(peer.ID("")).AnyTimes()

	s.mockFetcher = mock_tss.NewMockSaveDataFetcher(ctrl)
	s.mockFetcher.EXPECT().UnlockKeyshare().AnyTimes()
	s.mockFetcher.EXPECT().LockKeyshare().AnyTimes()
	s.mockFetcher.EXPECT().GetKeyshare().AnyTimes().Return(keyshare.ECDSAKeyshare{}, nil)

	s.mockBundleFetcher = mock_message.NewMockBundleFetcher(ctrl)
	b := new(rhinestone.Bundle)
	if err := json.Unmarshal([]byte(mock_message.MockBundleJSON), b); err != nil {
		panic(err)
	}
	s.mockBundle = b

	rhinestoneContract := contracts.NewRhinestoneContract()

	s.sigChn = make(chan interface{}, 1)

	tokens := make(map[uint64]map[string]config.TokenConfig)
	tokens[42161] = make(map[string]config.TokenConfig)
	tokens[42161]["WETH"] = config.TokenConfig{
		Address:  common.HexToAddress("0x0000000000000000000000000000000000000000"),
		Decimals: 18,
	}
	tokens[8453] = make(map[string]config.TokenConfig)
	tokens[8453]["WETH"] = config.TokenConfig{
		Address:  common.HexToAddress("0x4200000000000000000000000000000000000006"),
		Decimals: 18,
	}
	tokenStore := config.TokenStore{
		Tokens: tokens,
	}
	confirmations := make(map[uint64]uint64)
	confirmations[1000] = 100
	confirmations[2000] = 200

	s.handler = message.NewRhinestoneMessageHandler(
		8453,
		s.mockCoordinator,
		s.mockHost,
		s.mockCommunication,
		s.mockFetcher,
		tokenStore,
		*rhinestoneContract,
		s.mockBundleFetcher,
		s.sigChn,
	)
}

func (s *RhinestoneMessageHandlerTestSuite) Test_HandleMessage_ValidMessage() {
	s.mockCommunication.EXPECT().Broadcast(
		gomock.Any(),
		gomock.Any(),
		comm.RhinestoneMsg,
		fmt.Sprintf("%d-%s", 8453, comm.RhinestoneSessionID),
	).Return(nil)
	p, _ := pstoremem.NewPeerstore()
	s.mockHost.EXPECT().Peerstore().Return(p)

	errChn := make(chan error, 1)
	ad := &message.RhinestoneData{
		ErrChn:        errChn,
		Nonce:         big.NewInt(101),
		LiquidityPool: common.HexToAddress("0xe59aaf21c4D9Cf92d9eD4537f4404BA031f83b23"),
		Caller:        common.HexToAddress("0xde526bA5d1ad94cC59D7A79d99A59F607d31A657"),
		BorrowAmount:  big.NewInt(2493365192379644),
		BundleID:      "bundleID",
	}

	s.mockBundleFetcher.EXPECT().GetBundle("bundleID").Return(s.mockBundle, nil)
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

func (s *RhinestoneMessageHandlerTestSuite) Test_HandleMessage_FetchingBundleFails() {
	s.mockCommunication.EXPECT().Broadcast(
		gomock.Any(),
		gomock.Any(),
		comm.RhinestoneMsg,
		fmt.Sprintf("%d-%s", 8453, comm.RhinestoneSessionID),
	).Return(nil)
	p, _ := pstoremem.NewPeerstore()
	s.mockHost.EXPECT().Peerstore().Return(p)

	errChn := make(chan error, 1)
	ad := &message.RhinestoneData{
		ErrChn:        errChn,
		Nonce:         big.NewInt(101),
		LiquidityPool: common.HexToAddress("0xe59aaf21c4D9Cf92d9eD4537f4404BA031f83b23"),
		Caller:        common.HexToAddress("0xde526bA5d1ad94cC59D7A79d99A59F607d31A657"),
		BorrowAmount:  big.NewInt(3493365192379644),
		BundleID:      "bundleID",
	}

	s.mockBundleFetcher.EXPECT().GetBundle("bundleID").Return(s.mockBundle, fmt.Errorf("error"))

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

func (s *RhinestoneMessageHandlerTestSuite) Test_HandleMessage_InvalidBorrowAmount() {
	s.mockCommunication.EXPECT().Broadcast(
		gomock.Any(),
		gomock.Any(),
		comm.RhinestoneMsg,
		fmt.Sprintf("%d-%s", 8453, comm.RhinestoneSessionID),
	).Return(nil)
	p, _ := pstoremem.NewPeerstore()
	s.mockHost.EXPECT().Peerstore().Return(p)

	errChn := make(chan error, 1)
	ad := &message.RhinestoneData{
		ErrChn:        errChn,
		Nonce:         big.NewInt(101),
		LiquidityPool: common.HexToAddress("0xe59aaf21c4D9Cf92d9eD4537f4404BA031f83b23"),
		Caller:        common.HexToAddress("0xde526bA5d1ad94cC59D7A79d99A59F607d31A657"),
		BorrowAmount:  big.NewInt(3493365192379644),
		BundleID:      "bundleID",
	}

	s.mockBundleFetcher.EXPECT().GetBundle("bundleID").Return(s.mockBundle, nil)

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

func (s *RhinestoneMessageHandlerTestSuite) Test_HandleMessage_InvalidRepaymentAddresses() {
	s.mockCommunication.EXPECT().Broadcast(
		gomock.Any(),
		gomock.Any(),
		comm.RhinestoneMsg,
		fmt.Sprintf("%d-%s", 8453, comm.RhinestoneSessionID),
	).Return(nil)
	p, _ := pstoremem.NewPeerstore()
	s.mockHost.EXPECT().Peerstore().Return(p)

	errChn := make(chan error, 1)
	ad := &message.RhinestoneData{
		ErrChn:        errChn,
		Nonce:         big.NewInt(101),
		LiquidityPool: common.HexToAddress("0xinvalid"),
		Caller:        common.HexToAddress("0xde526bA5d1ad94cC59D7A79d99A59F607d31A657"),
		BorrowAmount:  big.NewInt(2493365192379644),
		BundleID:      "bundleID",
	}

	s.mockBundleFetcher.EXPECT().GetBundle("bundleID").Return(s.mockBundle, nil)

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
