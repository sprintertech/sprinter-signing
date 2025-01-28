// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package resharing_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/sourcegraph/conc/pool"
	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sprintertech/sprinter-signing/comm/elector"
	"github.com/sprintertech/sprinter-signing/keyshare"
	"github.com/sprintertech/sprinter-signing/tss"
	"github.com/sprintertech/sprinter-signing/tss/frost/resharing"
	tsstest "github.com/sprintertech/sprinter-signing/tss/test"
	"github.com/stretchr/testify/suite"
)

type ResharingTestSuite struct {
	tsstest.CoordinatorTestSuite
}

func TestRunResharingTestSuite(t *testing.T) {
	suite.Run(t, new(ResharingTestSuite))
}

func (s *ResharingTestSuite) Test_ValidResharingProcess_OldAndNewSubset() {
	communicationMap := make(map[peer.ID]*tsstest.TestCommunication)
	coordinators := []*tss.Coordinator{}
	processes := []tss.TssProcess{}

	hosts := []host.Host{}
	for i := 0; i < s.PartyNumber+1; i++ {
		host, _ := tsstest.NewHost(i)
		hosts = append(hosts, host)
	}
	for _, host := range hosts {
		for _, peer := range hosts {
			host.Peerstore().AddAddr(peer.ID(), peer.Addrs()[0], peerstore.PermanentAddrTTL)
		}
	}

	for i, host := range hosts {
		communication := tsstest.TestCommunication{
			Host:          host,
			Subscriptions: make(map[comm.SubscriptionID]chan *comm.WrappedMessage),
		}
		communicationMap[host.ID()] = &communication
		storer := keyshare.NewFrostKeyshareStore(fmt.Sprintf("../../test/keyshares/%d-frost.keyshare", i))
		share, err := storer.GetKeyshare()
		s.MockFrostStorer.EXPECT().LockKeyshare()
		s.MockFrostStorer.EXPECT().UnlockKeyshare()
		s.MockFrostStorer.EXPECT().GetKeyshare().Return(share, err)
		s.MockFrostStorer.EXPECT().StoreKeyshare(gomock.Any()).Return(nil)
		resharing := resharing.NewResharing("resharing2", 1, host, &communication, s.MockFrostStorer)
		electorFactory := elector.NewCoordinatorElectorFactory(host, s.BullyConfig)
		coordinators = append(coordinators, tss.NewCoordinator(host, &communication, electorFactory))
		processes = append(processes, resharing)
	}
	tsstest.SetupCommunication(communicationMap)

	resultChn := make(chan interface{})
	pool := pool.New().WithContext(context.Background()).WithCancelOnError()
	for i, coordinator := range coordinators {
		pool.Go(func(ctx context.Context) error {
			return coordinator.Execute(ctx, []tss.TssProcess{processes[i]}, resultChn)
		})
	}

	err := pool.Wait()
	s.Nil(err)
}

func (s *ResharingTestSuite) Test_ValidResharingProcess_RemovePeer() {
	communicationMap := make(map[peer.ID]*tsstest.TestCommunication)
	coordinators := []*tss.Coordinator{}
	processes := []tss.TssProcess{}

	hosts := []host.Host{}
	for i := 0; i < s.PartyNumber-1; i++ {
		host, _ := tsstest.NewHost(i)
		hosts = append(hosts, host)
	}
	for _, host := range hosts {
		for _, peer := range hosts {
			host.Peerstore().AddAddr(peer.ID(), peer.Addrs()[0], peerstore.PermanentAddrTTL)
		}
	}

	for i, host := range hosts {
		communication := tsstest.TestCommunication{
			Host:          host,
			Subscriptions: make(map[comm.SubscriptionID]chan *comm.WrappedMessage),
		}
		communicationMap[host.ID()] = &communication
		storer := keyshare.NewFrostKeyshareStore(fmt.Sprintf("../../test/keyshares/%d-frost.keyshare", i))
		share, err := storer.GetKeyshare()
		s.MockFrostStorer.EXPECT().LockKeyshare()
		s.MockFrostStorer.EXPECT().UnlockKeyshare()
		s.MockFrostStorer.EXPECT().GetKeyshare().Return(share, err)
		s.MockFrostStorer.EXPECT().StoreKeyshare(gomock.Any()).Return(nil)
		resharing := resharing.NewResharing("resharing2", 1, host, &communication, s.MockFrostStorer)
		electorFactory := elector.NewCoordinatorElectorFactory(host, s.BullyConfig)
		coordinators = append(coordinators, tss.NewCoordinator(host, &communication, electorFactory))
		processes = append(processes, resharing)
	}
	tsstest.SetupCommunication(communicationMap)

	resultChn := make(chan interface{})
	pool := pool.New().WithContext(context.Background()).WithCancelOnError()
	for i, coordinator := range coordinators {
		pool.Go(func(ctx context.Context) error {
			return coordinator.Execute(ctx, []tss.TssProcess{processes[i]}, resultChn)
		})
	}

	err := pool.Wait()
	s.Nil(err)
}
