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
//
// Each stream is mapped to a specific session, by sessionID
type StreamManager struct {
	streamsBySessionID map[string]map[peer.ID]network.Stream
	streamLocker       *sync.Mutex
	host               host.Host
}

// NewStreamManager creates new StreamManager
func NewStreamManager(host host.Host) *StreamManager {
	return &StreamManager{
		streamsBySessionID: make(map[string]map[peer.ID]network.Stream),
		streamLocker:       &sync.Mutex{},
		host:               host,
	}
}

// ReleaseStream removes reference on streams mapped to provided sessionID and closes them
func (sm *StreamManager) ReleaseStreams(sessionID string) {
	sm.streamLocker.Lock()
	defer sm.streamLocker.Unlock()

	streams, ok := sm.streamsBySessionID[sessionID]
	if !ok {
		return
	}

	for peer, stream := range streams {
		if stream.Conn() != nil {
			_ = stream.Conn().Close()
		}

		err := stream.Close()
		if err != nil {
			log.Debug().Msgf("Cannot close stream to peer %s, err: %s", peer.String(), err.Error())
		}
	}

	delete(sm.streamsBySessionID, sessionID)
}

// AddStream saves and maps provided stream to sessionID
func (sm *StreamManager) AddStream(sessionID string, peerID peer.ID, stream network.Stream) {
	sm.streamLocker.Lock()
	defer sm.streamLocker.Unlock()

	_, ok := sm.streamsBySessionID[sessionID]
	if !ok {
		sm.streamsBySessionID[sessionID] = make(map[peer.ID]network.Stream)
	}

	_, ok = sm.streamsBySessionID[sessionID][peerID]
	if ok {
		return
	}

	sm.streamsBySessionID[sessionID][peerID] = stream
}

// Stream fetches stream by peer and session ID
func (sm *StreamManager) Stream(sessionID string, peerID peer.ID, protocolID protocol.ID) (network.Stream, error) {
	sm.streamLocker.Lock()

	stream, ok := sm.streamsBySessionID[sessionID][peerID]
	if !ok {
		stream, err := sm.host.NewStream(context.TODO(), peerID, protocolID)
		if err != nil {
			return nil, err
		}

		sm.streamLocker.Unlock()
		sm.AddStream(sessionID, peerID, stream)
		return stream, nil
	}

	sm.streamLocker.Unlock()
	return stream, nil
}
