// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package signing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sourcegraph/conc/pool"

	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sprintertech/sprinter-signing/keyshare"
	tsserrors "github.com/sprintertech/sprinter-signing/tss"
	"github.com/sprintertech/sprinter-signing/tss/cggmp21"
	"github.com/sprintertech/sprinter-signing/tss/ecdsa/common"
	"github.com/sprintertech/sprinter-signing/tss/message"
	"github.com/sprintertech/sprinter-signing/tss/util"
)

type SaveDataFetcher interface {
	GetKeyshare() (keyshare.ECDSAKeyshare, error)
	LockKeyshare()
	UnlockKeyshare()
}

type EcdsaSignature struct {
	Signature []byte
	ID        string
}

// Signing drives a single cggmp21 threshold-ECDSA signing session for one
// party. It satisfies the tss.TssProcess interface; the underlying MPC is
// driven via tss/cggmp21 (Rust FFI), with this party-set/messaging plumbing
// kept in Go.
type Signing struct {
	host          host.Host
	communication comm.Communication
	log           zerolog.Logger
	// TssTimeout bounds how long the protocol may stall waiting for a new
	// inbound message before erroring out. Exported so tests can shorten it.
	TssTimeout time.Duration

	sid string
	msg *big.Int
	key keyshare.ECDSAKeyshare

	mux            sync.Mutex
	started        bool
	cancel         context.CancelFunc
	coordinator    bool
	resultChn      chan interface{}
	subscriptionID comm.SubscriptionID
}

func SessionID(chainID uint64, depositID string) string {
	return fmt.Sprintf("%d-%s", chainID, depositID)
}

func NewSigning(
	msg *big.Int,
	messageID string,
	sessionID string,
	host host.Host,
	communication comm.Communication,
	fetcher SaveDataFetcher,
) (*Signing, error) {
	fetcher.LockKeyshare()
	defer fetcher.UnlockKeyshare()
	key, err := fetcher.GetKeyshare()
	if err != nil {
		return nil, err
	}

	return &Signing{
		host:          host,
		communication: communication,
		sid:           sessionID,
		msg:           msg,
		key:           key,
		log:           log.With().Str("SessionID", sessionID).Str("messageID", messageID).Str("Process", "signing").Logger(),
		cancel:        func() {},
		TssTimeout:    8 * time.Second,
	}, nil
}

// SessionID returns the signing session identifier.
func (s *Signing) SessionID() string { return s.sid }

// Timeout returns the configured timeout for this signing process.
func (s *Signing) Timeout() time.Duration { return s.TssTimeout }

// Retryable signals that signing can be safely retried on failure.
func (s *Signing) Retryable() bool { return true }

// ValidCoordinators returns peers that hold a valid key share and can act as
// the signing coordinator for this process.
func (s *Signing) ValidCoordinators() []peer.ID { return s.key.Peers }

// Ready reports whether enough peers are ready to start signing (threshold+1).
func (s *Signing) Ready(readyPeers []peer.ID, _ []peer.ID) (bool, error) {
	return len(s.readyParticipants(readyPeers)) == s.key.Threshold+1, nil
}

// StartParams chooses threshold+1 peers from the ready set deterministically
// (sorted by hash(peer || session)) and returns the JSON-encoded subset.
func (s *Signing) StartParams(readyPeers []peer.ID) []byte {
	readyPeers = s.readyParticipants(readyPeers)
	sorted := util.SortPeersForSession(readyPeers, s.sid)
	subset := make([]peer.ID, 0, s.key.Threshold+1)
	for _, p := range sorted {
		subset = append(subset, p.ID)
		if len(subset) == s.key.Threshold+1 {
			break
		}
	}
	b, _ := json.Marshal(subset)
	return b
}

// Stop tears down communication subscriptions and cancels the running protocol.
func (s *Signing) Stop() {
	s.log.Info().Msgf("Stopping tss process.")
	s.communication.UnSubscribe(s.subscriptionID)
	s.cancel()
}

// Run starts the signing process. params carries the coordinator-chosen peer
// subset that this party must use.
func (s *Signing) Run(
	ctx context.Context,
	coordinator bool,
	resultChn chan interface{},
	params []byte,
) error {
	s.mux.Lock()
	if s.started {
		s.mux.Unlock()
		s.log.Warn().Msgf("Signing already started")
		return common.ErrProcessStarted
	}
	s.started = true
	s.mux.Unlock()

	s.coordinator = coordinator
	s.resultChn = resultChn
	ctx, s.cancel = context.WithCancel(ctx)

	// Subscribe FIRST — before any FFI work — so we never miss inbound messages
	// from peers that may start signing slightly ahead of us. The buffered
	// channel lets pre-Setup messages queue without blocking the sender.
	msgChn := make(chan *comm.WrappedMessage, 64)
	s.subscriptionID = s.communication.Subscribe(s.sid, comm.TssKeySignMsg, msgChn)

	signerPeers, err := unmarshallStartParams(params)
	if err != nil {
		return err
	}
	if !util.IsParticipant(s.host.ID(), signerPeers) {
		return &tsserrors.SubsetError{Peer: s.host.ID()}
	}

	// Establish the deterministic peer ↔ index mappings shared by every signer.
	signerPeers = sortPeersByPartyKey(signerPeers)
	peerToSignIdx := make(map[peer.ID]uint16, len(signerPeers))
	for i, p := range signerPeers {
		peerToSignIdx[p] = uint16(i)
	}
	keygenSigners := make([]uint16, len(signerPeers))
	for i, p := range signerPeers {
		ki, err := peerKeygenIndex(p, s.key.Peers)
		if err != nil {
			return fmt.Errorf("signing: %w", err)
		}
		keygenSigners[i] = ki
	}
	mySignIdx := peerToSignIdx[s.host.ID()]

	shareJSON, err := s.key.CggmpShare()
	if err != nil {
		return fmt.Errorf("signing: convert keyshare to cggmp21: %w", err)
	}

	hash := make([]byte, 32)
	s.msg.FillBytes(hash)

	session, err := cggmp21.NewSigningSession(shareJSON, []byte(s.sid), mySignIdx, keygenSigners, hash)
	if err != nil {
		return fmt.Errorf("signing: %w", err)
	}
	defer session.Close()

	s.log.Info().Msgf("Started signing process for message %s", s.msg.Text(16))

	p := pool.New().WithContext(ctx).WithCancelOnError()
	p.Go(func(ctx context.Context) error {
		return s.drive(ctx, session, signerPeers, peerToSignIdx, msgChn)
	})
	return p.Wait()
}

// drive owns the cggmp21 session: drains outgoing messages, delivers inbound
// messages, and finalises the signature. Single-threaded by design — the
// session is not safe for concurrent use.
func (s *Signing) drive(
	ctx context.Context,
	session *cggmp21.SigningSession,
	signerPeers []peer.ID,
	peerToSignIdx map[peer.ID]uint16,
	msgChn chan *comm.WrappedMessage,
) error {
	defer s.cancel()

	send := func() error {
		for {
			msg, ok, err := session.NextOutgoing()
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}
			wire, err := message.MarshalTssMessage(msg.Payload, msg.IsBroadcast())
			if err != nil {
				s.log.Error().Err(err).Msg("marshal outgoing tss message")
				continue
			}
			var dests []peer.ID
			if msg.IsBroadcast() {
				for _, p := range signerPeers {
					if p != s.host.ID() {
						dests = append(dests, p)
					}
				}
			} else {
				if int(msg.Recipient) >= len(signerPeers) {
					return fmt.Errorf("signing: outgoing recipient %d out of range", msg.Recipient)
				}
				dests = []peer.ID{signerPeers[msg.Recipient]}
			}
			if err := s.communication.Broadcast(dests, wire, comm.TssKeySignMsg, s.sid); err != nil {
				return fmt.Errorf("signing: broadcast: %w", err)
			}
		}
	}

	// Drain anything the session emitted at construction time.
	if err := send(); err != nil {
		return err
	}

	for {
		if session.Done() {
			sig, err := session.Signature()
			if err != nil {
				return fmt.Errorf("signing: %w", err)
			}
			s.log.Info().Msg("Successfully generated signature")
			s.resultChn <- EcdsaSignature{Signature: sig, ID: s.sid}
			return s.distributeSignature(sig)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(s.TssTimeout):
			return &comm.CommunicationError{
				Err: fmt.Errorf("signing: timed out waiting for messages after %s", s.TssTimeout),
			}
		case wMsg := <-msgChn:
			tssMsg, err := message.UnmarshalTssMessage(wMsg.Payload)
			if err != nil {
				s.log.Error().Err(err).Msgf("unmarshal message from %s", wMsg.From)
				continue
			}
			senderIdx, ok := peerToSignIdx[wMsg.From]
			if !ok {
				s.log.Warn().Msgf("dropping message from non-signer %s", wMsg.From)
				continue
			}
			if err := session.Deliver(senderIdx, tssMsg.IsBroadcast, tssMsg.MsgBytes); err != nil {
				return fmt.Errorf("signing: deliver from %s: %w", wMsg.From, err)
			}
			if err := send(); err != nil {
				return err
			}
		}
	}
}

// readyParticipants returns the subset of readyPeers that also hold a valid
// share (i.e. participated in keygen).
func (s *Signing) readyParticipants(readyPeers []peer.ID) []peer.ID {
	out := make([]peer.ID, 0, len(readyPeers))
	for _, p := range readyPeers {
		if slices.Contains(s.key.Peers, p) {
			out = append(out, p)
		}
	}
	return out
}

// distributeSignature broadcasts the final signature to peers that aren't part
// of the signing subset (so they learn the result). Non-coordinators skip; the
// coordinator handles result distribution.
func (s *Signing) distributeSignature(sig []byte) error {
	if s.coordinator {
		return nil
	}
	sigMsg, err := message.MarshalSignatureMessage(s.sid, sig)
	if err != nil {
		return err
	}
	return s.communication.Broadcast(s.host.Peerstore().Peers(), sigMsg, comm.SignatureMsg, comm.SignatureSessionID)
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func unmarshallStartParams(b []byte) ([]peer.ID, error) {
	var peers []peer.ID
	if err := json.Unmarshal(b, &peers); err != nil {
		return nil, err
	}
	if len(peers) == 0 {
		return nil, errors.New("signing: empty peer subset in start params")
	}
	return peers, nil
}

// partyKey returns the big.Int derived from peer.ID's bytes that tss-lib /
// threshlib uses for ordering parties (see CreatePartyID in
// tss/ecdsa/common/utils.go). The keygen-time index is the position of the
// peer in the sort order produced by these keys.
func partyKey(p peer.ID) *big.Int {
	return new(big.Int).SetBytes([]byte(p.String()))
}

// sortPeersByPartyKey returns peers sorted ascending by their party key, which
// is the canonical signing-time ordering used to map peers to signing indexes.
func sortPeersByPartyKey(peers []peer.ID) []peer.ID {
	out := make([]peer.ID, len(peers))
	copy(out, peers)
	sort.Slice(out, func(i, j int) bool {
		return partyKey(out[i]).Cmp(partyKey(out[j])) < 0
	})
	return out
}

// peerKeygenIndex returns the keygen-time index of `target` within the full
// set of keygen peers — i.e. its position in the sorted-by-party-key list of
// all peers that participated in keygen. cggmp21 needs this to look up the
// corresponding VSS evaluation point and public share for each signer.
func peerKeygenIndex(target peer.ID, keygenPeers []peer.ID) (uint16, error) {
	sorted := sortPeersByPartyKey(keygenPeers)
	for i, p := range sorted {
		if p == target {
			return uint16(i), nil
		}
	}
	return 0, fmt.Errorf("peer %s not found among keygen peers", target)
}
