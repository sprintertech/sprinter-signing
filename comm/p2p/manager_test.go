// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package p2p_test

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/sprintertech/sprinter-signing/comm/p2p"

	mock_host "github.com/sprintertech/sprinter-signing/comm/p2p/mock/host"
	mock_network "github.com/sprintertech/sprinter-signing/comm/p2p/mock/stream"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type StreamManagerTestSuite struct {
	suite.Suite
	mockController *gomock.Controller
	mockHost       *mock_host.MockHost
}

func TestRunStreamManagerTestSuite(t *testing.T) {
	suite.Run(t, new(StreamManagerTestSuite))
}

func (s *StreamManagerTestSuite) SetupTest() {
	s.mockController = gomock.NewController(s.T())
	s.mockHost = mock_host.NewMockHost(s.mockController)
}

func (s *StreamManagerTestSuite) Test_ManagingSubscriptions_Success() {
	streamManager := p2p.NewStreamManager(s.mockHost, protocol.ID("1"))

	mockConn := mock_network.NewMockConn(s.mockController)
	stream1 := mock_network.NewMockStream(s.mockController)
	stream1.EXPECT().Conn().Return(mockConn).AnyTimes()
	s.mockHost.EXPECT().NewStream(gomock.Any(), gomock.Any(), gomock.Any()).Return(stream1, nil)

	peerID1, _ := peer.Decode("QmcW3oMdSqoEcjbyd51auqC23vhKX6BqfcZcY2HJ3sKAZR")

	s1, err := streamManager.Stream(peerID1)
	s.Nil(err)
	s2, err := streamManager.Stream(peerID1)
	s.Nil(err)

	s.Equal(s1, s2)

	stream1.EXPECT().Close().Times(1).Return(nil)
	streamManager.CloseStream(peerID1)
}
