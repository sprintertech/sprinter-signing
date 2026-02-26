// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package p2p

import (
	"context"
	"sync"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/rs/zerolog/log"
)

// StreamManager manages instances of network.Stream
type StreamManager struct {
	streamsByPeer map[peer.ID]network.Stream
	streamLocker  *sync.Mutex
	host          host.Host
	protocolID    protocol.ID
}

// NewStreamManager creates new StreamManager
func NewStreamManager(host host.Host, protocolID protocol.ID) *StreamManager {
	return &StreamManager{
		streamsByPeer: make(map[peer.ID]network.Stream),
		streamLocker:  &sync.Mutex{},
		host:          host,
		protocolID:    protocolID,
	}
}

// CloseStream closes stream to the peer
func (sm *StreamManager) CloseStream(peerID peer.ID) {
	sm.streamLocker.Lock()
	stream, ok := sm.streamsByPeer[peerID]
	if !ok {
		sm.streamLocker.Unlock()
		return
	}

	delete(sm.streamsByPeer, peerID)
	sm.streamLocker.Unlock()

	err := stream.Close()
	if err != nil {
		log.Warn().Err(err).Msgf("Failed to close stream")
		return
	}
}

// Stream fetches stream by peer
func (sm *StreamManager) Stream(peerID peer.ID) (network.Stream, error) {
	sm.streamLocker.Lock()
	defer sm.streamLocker.Unlock()

	stream, ok := sm.streamsByPeer[peerID]
	if !ok {
		stream, err := sm.host.NewStream(context.TODO(), peerID, sm.protocolID)
		if err != nil {
			return nil, err
		}

		sm.streamsByPeer[peerID] = stream
		return stream, nil
	}

	return stream, nil
}
