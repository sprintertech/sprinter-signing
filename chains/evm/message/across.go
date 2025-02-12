package message

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/events"
	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sprintertech/sprinter-signing/tss"
	"github.com/sprintertech/sprinter-signing/tss/ecdsa/signing"
	tssMessage "github.com/sprintertech/sprinter-signing/tss/message"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
)

const (
	AcrossMessage = "AcrossMessage"
)

type EventFilterer interface {
	FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error)
}

type AcrossData struct {
	depositId   *big.Int
	coordinator peer.ID
}

func NewAcrossMessage(source uint8, destination uint8, acrossData AcrossData) *message.Message {
	return &message.Message{
		Source:      source,
		Destination: destination,
		Data:        acrossData,
		Type:        AcrossMessage,
		Timestamp:   time.Now(),
	}
}

type AcrossMessageHandler struct {
	client EventFilterer

	across common.Address
	abi    abi.ABI

	coordinator *tss.Coordinator
	host        host.Host
	comm        comm.Communication
	fetcher     signing.SaveDataFetcher

	sigChn chan interface{}
}

func (h *AcrossMessageHandler) Listen(ctx context.Context) {
	msgChn := make(chan *comm.WrappedMessage)
	subID := h.comm.Subscribe(comm.AcrossSessionID, comm.AcrossMsg, msgChn)

	for {
		select {
		case wMsg := <-msgChn:
			{
				acrossMsg, err := tssMessage.UnmarshalAcrossMessage(wMsg.Payload)
				if err != nil {
					log.Warn().Msgf("Failed unmarshaling across message: %s", err)
					continue
				}

				msg := NewAcrossMessage(acrossMsg.Source, acrossMsg.Destination, AcrossData{
					depositId:   acrossMsg.DepositId,
					coordinator: wMsg.From,
				})
				_, err = h.HandleMessage(msg)
				if err != nil {
					log.Err(err).Msgf("Failed handling across message %+v because of: %s", acrossMsg, err)
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

// HandleMessage finds the Across deposit with the according deposit ID and starts
// the MPC signature process for it. The result will be saved into the signature
// cache through the result channel.
func (h *AcrossMessageHandler) HandleMessage(m *message.Message) (*proposal.Proposal, error) {
	data := m.Data.(AcrossData)
	err := h.notify(m, data)
	if err != nil {
		return nil, err
	}

	d, err := h.deposit(data.depositId)
	if err != nil {
		return nil, err
	}

	signing, err := signing.NewSigning(
		d.DepositId,
		data.depositId.Text(16),
		data.depositId.Text(16),
		h.host,
		h.comm,
		h.fetcher)
	if err != nil {
		return nil, err
	}

	err = h.coordinator.Execute(context.Background(), []tss.TssProcess{signing}, h.sigChn, data.coordinator)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (h *AcrossMessageHandler) notify(m *message.Message, data AcrossData) error {
	msgBytes, err := tssMessage.MarshalAcrossMessage(data.depositId, m.Source, m.Destination)
	if err != nil {
		return err
	}

	return h.comm.Broadcast(h.host.Peerstore().Peers(), msgBytes, comm.AcrossMsg, comm.AcrossSessionID)
}

func (h *AcrossMessageHandler) deposit(depositId *big.Int) (*events.AcrossDeposit, error) {
	q := ethereum.FilterQuery{
		Addresses: []common.Address{
			h.across,
		},
		Topics: [][]common.Hash{
			{events.AcrossDepositSig.GetTopic()},
			nil,
			{common.HexToHash(depositId.Text(16))},
		},
	}
	logs, err := h.client.FilterLogs(context.Background(), q)
	if err != nil {
		return nil, err
	}

	if len(logs) == 0 {
		return nil, fmt.Errorf("no deposit found with ID: %s", depositId)
	}

	return h.parseDeposit(logs[0])
}

func (h *AcrossMessageHandler) parseDeposit(l types.Log) (*events.AcrossDeposit, error) {
	var d *events.AcrossDeposit
	err := h.abi.UnpackIntoInterface(&d, "V3FundsDeposited", l.Data)
	return d, err
}
