// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package common

import (
	"context"
	"fmt"
	"math/big"
	"runtime/debug"
	"time"

	"github.com/binance-chain/tss-lib/tss"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog"
	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sprintertech/sprinter-signing/tss/message"
)

type Party interface {
	UpdateFromBytes(wireBytes []byte, from *tss.PartyID, isBroadcast bool, sessionID *big.Int) (bool, *tss.Error)
	Start() *tss.Error
	WaitingFor() []*tss.PartyID
}

// BaseTss contains common variables and methods to
// all tss processes.
type BaseTss struct {
	Host          host.Host
	SID           string
	Party         Party
	PartyStore    map[string]*tss.PartyID
	Communication comm.Communication
	Peers         []peer.ID
	Log           zerolog.Logger
	TssTimeout    time.Duration

	Cancel context.CancelFunc
}

// PopulatePartyStore populates party store map with sorted parties for
// mapping message senders to parties
func (b *BaseTss) PopulatePartyStore(parties tss.SortedPartyIDs) {
	for _, party := range parties {
		b.PartyStore[party.Id] = party
	}
}

// ProcessInboundMessages processes messages from tss parties and updates local party accordingly.
func (b *BaseTss) ProcessInboundMessages(ctx context.Context, msgChan chan *comm.WrappedMessage) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s", string(debug.Stack()))
		}
	}()

	for {
		select {
		case wMsg := <-msgChan:
			{
				go func(wMsg *comm.WrappedMessage) {
					b.Log.Debug().Msgf("Processed inbound message from %s", wMsg.From)

					msg, err := message.UnmarshalTssMessage(wMsg.Payload)
					if err != nil {
						b.Log.Error().Err(err).Msgf("Failed unmarshaling message from %s", wMsg.From)
						return
					}

					ok, err := b.Party.UpdateFromBytes(
						msg.MsgBytes,
						b.PartyStore[wMsg.From.String()],
						msg.IsBroadcast,
						new(big.Int).SetBytes([]byte(b.SID)))
					if !ok {
						b.Log.Error().Err(err).Msgf("Failed updating party with message from %s", wMsg.From)
						return
					}
					b.Log.Debug().Msgf("Updated party with message from %s", wMsg.From)
					return
				}(wMsg)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

// ProcessOutboundMessages sends messages received from tss out channel to target peers.
// On context cancel stops listening to channel and exits.
func (b *BaseTss) ProcessOutboundMessages(ctx context.Context, outChn chan tss.Message, messageType comm.MessageType) error {
	for {
		select {
		case msg := <-outChn:
			{
				go func(msg tss.Message) {
					b.Log.Debug().Msg(msg.String())
					wireBytes, routing, err := msg.WireBytes()
					if err != nil {
						b.Log.Error().Err(err).Msgf("Failed getting wire bytes")
						return
					}

					msgBytes, err := message.MarshalTssMessage(wireBytes, routing.IsBroadcast)
					if err != nil {
						b.Log.Error().Err(err).Msgf("Failed marshaling message")
						return
					}

					peers, err := b.BroadcastPeers(msg)
					if err != nil {
						b.Log.Error().Err(err).Msgf("Failed getting broadcast peers")
						return
					}

					b.Log.Debug().Msgf("Sending message to %s", peers)
					err = b.Communication.Broadcast(peers, msgBytes, messageType, b.SessionID())
					if err != nil {
						b.Log.Error().Err(err).Msgf("Failed broadcasting message")
						return
					}
				}(msg)
			}
		case <-ctx.Done():
			{
				return nil
			}
		}
	}
}

// BroccastPeers returns peers that should receive the tss message
func (b *BaseTss) BroadcastPeers(msg tss.Message) ([]peer.ID, error) {
	if msg.IsBroadcast() {
		return b.Peers, nil
	} else {
		return PeersFromParties(msg.GetTo())
	}
}

func (b *BaseTss) SessionID() string {
	return b.SID
}

func (b *BaseTss) Timeout() time.Duration {
	return b.TssTimeout
}
