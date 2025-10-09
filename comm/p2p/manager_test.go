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
	streamManager := p2p.NewStreamManager(s.mockHost)

	mockConn := mock_network.NewMockConn(s.mockController)
	stream1 := mock_network.NewMockStream(s.mockController)
	stream1.EXPECT().Conn().Return(mockConn).AnyTimes()
	stream2 := mock_network.NewMockStream(s.mockController)
	stream2.EXPECT().Conn().Return(mockConn).AnyTimes()
	stream3 := mock_network.NewMockStream(s.mockController)
	stream3.EXPECT().Conn().Return(mockConn).AnyTimes()

	peerID1, _ := peer.Decode("QmcW3oMdSqoEcjbyd51auqC23vhKX6BqfcZcY2HJ3sKAZR")
	peerID2, _ := peer.Decode("QmZHPnN3CKiTAp8VaJqszbf8m7v4mPh15M421KpVdYHF54")

	streamManager.AddStream("1", peerID1, stream1)
	streamManager.AddStream("1", peerID1, stream1)
	streamManager.AddStream("1", peerID2, stream2)
	streamManager.AddStream("2", peerID1, stream3)

	stream1.EXPECT().Close().Times(1).Return(nil)
	stream2.EXPECT().Close().Times(1).Return(nil)

	streamManager.ReleaseStreams("1")
}

func (s *StreamManagerTestSuite) Test_FetchStream_NoStream() {
	streamManager := p2p.NewStreamManager(s.mockHost)

	expectedStream := mock_network.NewMockStream(s.mockController)
	s.mockHost.EXPECT().NewStream(gomock.Any(), gomock.Any(), gomock.Any()).Return(expectedStream, nil)

	stream, err := streamManager.Stream("1", peer.ID(""), protocol.ID(""))

	s.Nil(err)
	s.Equal(stream, expectedStream)
}

func (s *StreamManagerTestSuite) Test_FetchStream_ValidStream() {
	streamManager := p2p.NewStreamManager(s.mockHost)

	stream := mock_network.NewMockStream(s.mockController)
	peerID1, _ := peer.Decode("QmcW3oMdSqoEcjbyd51auqC23vhKX6BqfcZcY2HJ3sKAZR")
	streamManager.AddStream("1", peerID1, stream)

	expectedStream, err := streamManager.Stream("1", peerID1, protocol.ID(""))

	s.Nil(err)
	s.Equal(stream, expectedStream)
}

func (s *StreamManagerTestSuite) Test_AddStream_IgnoresExistingPeer() {
	streamManager := p2p.NewStreamManager(s.mockHost)

	stream1 := mock_network.NewMockStream(s.mockController)
	stream2 := mock_network.NewMockStream(s.mockController)
	peerID1, _ := peer.Decode("QmcW3oMdSqoEcjbyd51auqC23vhKX6BqfcZcY2HJ3sKAZR")
	streamManager.AddStream("1", peerID1, stream1)
	streamManager.AddStream("1", peerID1, stream2)

	expectedStream, err := streamManager.Stream("1", peerID1, protocol.ID(""))

	s.Nil(err)
	s.Equal(stream1, expectedStream)
}
