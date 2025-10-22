package message

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sprintertech/sprinter-signing/tss"
	"github.com/sprintertech/sprinter-signing/tss/ecdsa/signing"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
)

type Coordinator interface {
	Execute(ctx context.Context, tssProcesses []tss.TssProcess, resultChn chan interface{}, coordinator peer.ID) error
}

type LighterMessageHandler struct {
	coordinator Coordinator
	host        host.Host
	comm        comm.Communication
	fetcher     signing.SaveDataFetcher

	sigChn chan any
}

// HandleMessage finds the Mayan deposit with the according deposit ID and starts
// the MPC signature process for it. The result will be saved into the signature
// cache through the result channel.
func (h *LighterMessageHandler) HandleMessage(m *message.Message) (*proposal.Proposal, error) {
	data := m.Data.(*LighterData)

	err := h.notify(data)
	if err != nil {
		log.Warn().Msgf("Failed to notify relayers because of %s", err)
	}

	unlockHash := []byte{}
	sessionID := fmt.Sprintf("lighter-%s", "")
	signing, err := signing.NewSigning(
		new(big.Int).SetBytes(unlockHash),
		sessionID,
		sessionID,
		h.host,
		h.comm,
		h.fetcher)
	if err != nil {
		return nil, err
	}

	err = h.coordinator.Execute(context.Background(), []tss.TssProcess{signing}, h.sigChn, data.Coordinator)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (h *LighterMessageHandler) verifyOrder() error {
	return nil
}

func (h *LighterMessageHandler) Listen(ctx context.Context) {
	msgChn := make(chan *comm.WrappedMessage)
	subID := h.comm.Subscribe(comm.LighterSessionID, comm.LighterMsg, msgChn)

	for {
		select {
		case wMsg := <-msgChn:
			{
				d := &LighterData{}
				err := json.Unmarshal(wMsg.Payload, d)
				if err != nil {
					log.Warn().Msgf("Failed unmarshaling Mayan message: %s", err)
					continue
				}

				d.ErrChn = make(chan error, 1)
				msg := NewLighterMessage(d.Source, d.Destination, d)
				_, err = h.HandleMessage(msg)
				if err != nil {
					log.Err(err).Msgf("Failed handling Mayan message %+v because of: %s", msg, err)
				}
			}
		case <-ctx.Done():
			{
				h.comm.UnSubscribe(subID)
				return
			}
		}
	}
}

func (h *LighterMessageHandler) notify(data *LighterData) error {
	if data.Coordinator != peer.ID("") {
		return nil
	}

	data.Coordinator = h.host.ID()
	msgBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return h.comm.Broadcast(h.host.Peerstore().Peers(), msgBytes, comm.LighterMsg, comm.LighterSessionID)
}
