// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package elector_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sprintertech/sprinter-signing/comm/elector"
	"github.com/sprintertech/sprinter-signing/tss/util"

	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/sprintertech/sprinter-signing/comm/p2p"
	"github.com/sprintertech/sprinter-signing/config/relayer"
	"github.com/sprintertech/sprinter-signing/topology"
	"github.com/stretchr/testify/suite"
)

type BullyTestSuite struct {
	suite.Suite
	mockController *gomock.Controller
	testProtocolID protocol.ID
	testSessionID  string
	portOffset     uint16
}

type RelayerTestDescriber struct {
	name         string
	index        int
	initialDelay time.Duration
}

type BullyTestCase struct {
	name           string
	isLeaderActive bool
	testRelayers   []RelayerTestDescriber
}

func TestRunCommunicationIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(BullyTestSuite))
}

func (s *BullyTestSuite) SetupSuite() {
	s.testProtocolID = "/sygma/coordinator/1.0.0"
	s.testSessionID = "1"
	s.portOffset = 0
}
func (s *BullyTestSuite) TearDownSuite() {}
func (s *BullyTestSuite) SetupTest()     {}

func (s *BullyTestSuite) SetupIndividualTest(c BullyTestCase) ([]elector.CoordinatorElector, peer.ID, peer.ID, []host.Host, peer.IDSlice) {
	s.mockController = gomock.NewController(s.T())
	var testHosts []host.Host
	allowedPeers := peer.IDSlice{}
	var testBullyCoordinators []elector.CoordinatorElector

	numberOfTestHosts := uint16(len(c.testRelayers))

	topology := &topology.NetworkTopology{
		Peers: []*peer.AddrInfo{},
	}
	privateKeys := []crypto.PrivKey{}
	for i := range numberOfTestHosts {
		privKeyForHost, _, _ := crypto.GenerateKeyPair(crypto.ECDSA, 1)
		privateKeys = append(privateKeys, privKeyForHost)
		peerID, _ := peer.IDFromPrivateKey(privKeyForHost)
		addrInfoForHost, _ := peer.AddrInfoFromString(fmt.Sprintf(
			"/ip4/127.0.0.1/tcp/%d/p2p/%s", 4000+s.portOffset+i, peerID.String(),
		))
		topology.Peers = append(topology.Peers, addrInfoForHost)
	}

	// create test hosts
	for i := range numberOfTestHosts {
		connectionGate := p2p.NewConnectionGate(topology)
		newHost, _ := p2p.NewHost(privateKeys[i], topology, connectionGate, 4000+s.portOffset+i)
		testHosts = append(testHosts, newHost)
		allowedPeers = append(allowedPeers, newHost.ID())
	}

	sortedPeers := util.SortPeersForSession(allowedPeers, s.testSessionID)
	initialCoordinator := sortedPeers[0].ID
	var finalCoordinator peer.ID
	if !c.isLeaderActive {
		finalCoordinator = sortedPeers[1].ID
	} else {
		finalCoordinator = initialCoordinator
	}

	s.portOffset += numberOfTestHosts
	for i := range numberOfTestHosts {
		com := p2p.NewCommunication(
			testHosts[i],
			s.testProtocolID,
		)

		if !c.isLeaderActive && testHosts[i].ID() == initialCoordinator {
			testBullyCoordinators = append(testBullyCoordinators, nil)
		} else {
			b := elector.NewBullyCoordinatorElector(s.testSessionID, testHosts[i], relayer.BullyConfig{
				PingWaitTime:     1 * time.Second,
				PingBackOff:      1 * time.Second,
				PingInterval:     1 * time.Second,
				ElectionWaitTime: 2 * time.Second,
				BullyWaitTime:    10 * time.Second,
			}, com)
			testBullyCoordinators = append(testBullyCoordinators, b)
		}
	}

	return testBullyCoordinators, initialCoordinator, finalCoordinator, testHosts, allowedPeers
}
func (s *BullyTestSuite) TearDownTest() {}

func (s *BullyTestSuite) TestBully_GetCoordinator_OneDelay() {
	testCases := []BullyTestCase{
		{
			name:           "five relayers bully coordination - multiple lags on relayers",
			isLeaderActive: true,
			testRelayers: []RelayerTestDescriber{
				{
					name:         "R1",
					index:        0,
					initialDelay: 1 * time.Second,
				},
				{
					name:         "R2",
					index:        1,
					initialDelay: 2 * time.Second,
				},
				{
					name:         "R3",
					index:        2,
					initialDelay: 3 * time.Second,
				},
				{
					name:         "R4",
					index:        3,
					initialDelay: 2 * time.Second,
				},
				{
					name:         "R5",
					index:        4,
					initialDelay: 0,
				},
			},
		},
		{
			name:           "five relayers bully coordination - leader not active",
			isLeaderActive: false,
			testRelayers: []RelayerTestDescriber{
				{
					name:         "R1",
					index:        0,
					initialDelay: 0,
				},
				{
					name:         "R2",
					index:        1,
					initialDelay: 0,
				},
				{
					name:         "R3",
					index:        2,
					initialDelay: 0,
				},
				{
					name:         "R4",
					index:        3,
					initialDelay: 0,
				},
				{
					name:         "R5",
					index:        4,
					initialDelay: 0,
				},
			},
		},
	}

	for _, t := range testCases {
		testBullyCoordinators, initialCoordinator, finalCoordinator, testHosts, allowedPeers := s.SetupIndividualTest(t)

		s.Run(t.name, func() {
			resultChan := make(chan peer.ID)
			for _, r := range t.testRelayers {
				rDescriber := r
				if !t.isLeaderActive && testHosts[rDescriber.index].ID() == initialCoordinator {
					// in case leader is not active
				} else {
					go func() {
						if rDescriber.initialDelay > 0 {
							time.Sleep(rDescriber.initialDelay)
						}
						c, err := testBullyCoordinators[rDescriber.index].Coordinator(context.Background(), allowedPeers)

						s.Nil(err)
						resultChan <- c
					}()
				}
			}

			numberOfResults := len(t.testRelayers)
			if !t.isLeaderActive {
				numberOfResults -= 1
			}
			for i := 0; i < numberOfResults; i++ {
				c := <-resultChan

				s.Equal(finalCoordinator.String(), c.String())
			}
		})
	}
}
