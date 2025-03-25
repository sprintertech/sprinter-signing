// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package keygen_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/sourcegraph/conc/pool"
	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sprintertech/sprinter-signing/comm/elector"
	"github.com/sprintertech/sprinter-signing/tss"
	"github.com/sprintertech/sprinter-signing/tss/ecdsa/keygen"
	tsstest "github.com/sprintertech/sprinter-signing/tss/test"
	"github.com/stretchr/testify/suite"
)

type KeygenTestSuite struct {
	tsstest.CoordinatorTestSuite
}

func TestRunKeygenTestSuite(t *testing.T) {
	suite.Run(t, new(KeygenTestSuite))
}

func (s *KeygenTestSuite) Test_ValidKeygenProcess() {
	communicationMap := make(map[peer.ID]*tsstest.TestCommunication)
	coordinators := []*tss.Coordinator{}
	processes := []tss.TssProcess{}

	for _, host := range s.Hosts {
		communication := tsstest.TestCommunication{
			Host:          host,
			Subscriptions: make(map[comm.SubscriptionID]chan *comm.WrappedMessage),
		}
		communicationMap[host.ID()] = &communication
		keygen := keygen.NewKeygen("keygen", s.Threshold, host, &communication, s.MockECDSAStorer)
		electorFactory := elector.NewCoordinatorElectorFactory(host, s.BullyConfig)
		coordinator := tss.NewCoordinator(host, &communication, electorFactory)
		coordinators = append(coordinators, coordinator)
		processes = append(processes, keygen)
	}
	tsstest.SetupCommunication(communicationMap)

	s.MockECDSAStorer.EXPECT().LockKeyshare().Times(3)
	s.MockECDSAStorer.EXPECT().UnlockKeyshare().Times(3)
	s.MockECDSAStorer.EXPECT().StoreKeyshare(gomock.Any()).Times(3)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool := pool.New().WithContext(ctx)
	for i, coordinator := range coordinators {
		pool.Go(func(ctx context.Context) error {
			return coordinator.Execute(ctx, []tss.TssProcess{processes[i]}, nil, peer.ID(""))
		})
	}

	err := pool.Wait()
	s.Nil(err)
}

func (s *KeygenTestSuite) Test_KeygenTimeout() {
	communicationMap := make(map[peer.ID]*tsstest.TestCommunication)
	coordinators := []*tss.Coordinator{}
	processes := []tss.TssProcess{}
	for _, host := range s.Hosts {
		communication := tsstest.TestCommunication{
			Host:          host,
			Subscriptions: make(map[comm.SubscriptionID]chan *comm.WrappedMessage),
		}
		communicationMap[host.ID()] = &communication
		keygen := keygen.NewKeygen("keygen2", s.Threshold, host, &communication, s.MockECDSAStorer)
		electorFactory := elector.NewCoordinatorElectorFactory(host, s.BullyConfig)
		coordinator := tss.NewCoordinator(host, &communication, electorFactory)
		coordinator.TssTimeout = time.Millisecond
		coordinators = append(coordinators, coordinator)
		processes = append(processes, keygen)
	}
	tsstest.SetupCommunication(communicationMap)

	s.MockECDSAStorer.EXPECT().LockKeyshare().AnyTimes()
	s.MockECDSAStorer.EXPECT().UnlockKeyshare().AnyTimes()
	s.MockECDSAStorer.EXPECT().StoreKeyshare(gomock.Any()).Times(0)
	pool := pool.New().WithContext(context.Background())
	for i, coordinator := range coordinators {
		pool.Go(func(ctx context.Context) error {
			return coordinator.Execute(ctx, []tss.TssProcess{processes[i]}, nil, peer.ID(""))
		})
	}

	err := pool.Wait()

	s.Nil(err)
}
