package message

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sprintertech/sprinter-signing/tss/ecdsa/signing"
	"github.com/sygmaprotocol/sygma-core/relayer/message"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
)

type SprinterCreditMessageHandler struct {
	chainID uint64

	liquidators map[common.Address]common.Address

	host    host.Host
	comm    comm.Communication
	sigChn  chan any
	msgChan chan []*message.Message
}

func NewSprinterCreditMessageHandler(
	chainID uint64,
	liquidators map[common.Address]common.Address,
	host host.Host,
	comm comm.Communication,
	sigChn chan any,
	msgChan chan []*message.Message,
) *SprinterCreditMessageHandler {
	return &SprinterCreditMessageHandler{
		chainID:     chainID,
		liquidators: liquidators,
		host:        host,
		comm:        comm,
		sigChn:      sigChn,
		msgChan:     msgChan,
	}
}

// HandleMessage signs the liquidation request if the transaction
// is going to the Liquidator contract.
func (h *SprinterCreditMessageHandler) HandleMessage(m *message.Message) (*proposal.Proposal, error) {
	data := m.Data.(*SprinterCreditData)

	log.Info().Msgf("Handling sprinter remote collateral message %+v", data)

	err := h.notify(data)
	if err != nil {
		log.Warn().Msgf("Failed to notify relayers because of %s", err)
	}

	calldata, err := hex.DecodeString(data.Calldata)
	if err != nil {
		data.ErrChn <- err
		return nil, err
	}

	token := common.HexToAddress(data.TokenOut)
	liquidator, ok := h.liquidators[token]
	if !ok {
		err := fmt.Errorf("no liquidator for token %s", data.TokenOut)
		data.ErrChn <- err
		return nil, err
	}

	data.ErrChn <- nil

	sessionID := signing.SessionID(h.chainID, data.DepositID)
	h.msgChan <- []*message.Message{NewSignMessage(0, h.chainID, &SignRequest{
		Calldata:      calldata,
		BorrowAmount:  data.BorrowAmount,
		BorrowToken:   token.Hex(),
		Target:        liquidator.Hex(),
		Deadline:      data.Deadline,
		Caller:        data.Caller,
		LiquidityPool: data.LiquidityPool,
		Nonce:         data.Nonce,
		SessionID:     sessionID,
		Coordinator:   data.Coordinator,
		ResultChn:     h.sigChn,
	})}
	return nil, nil
}

func (h *SprinterCreditMessageHandler) Listen(ctx context.Context) {
	msgChn := make(chan *comm.WrappedMessage)
	subID := h.comm.Subscribe(
		fmt.Sprintf("%d-%s", h.chainID, comm.SprinterCreditSessionID),
		comm.SprinterCreditMsg,
		msgChn)

	for {
		select {
		case wMsg := <-msgChn:
			{
				go func(wMsg *comm.WrappedMessage) {
					d := &SprinterCreditData{}
					err := json.Unmarshal(wMsg.Payload, d)
					if err != nil {
						log.Warn().Msgf("Failed unmarshaling across message: %s", err)
						return
					}

					d.ErrChn = make(chan error, 1)
					msg := NewSprinterCreditMessage(d.Source, d.Destination, d)
					_, err = h.HandleMessage(msg)
					if err != nil {
						log.Err(err).Msgf("Failed handling across message %+v because of: %s", msg, err)
					}
				}(wMsg)
			}
		case <-ctx.Done():
			{
				h.comm.UnSubscribe(subID)
				return
			}
		}
	}
}

func (h *SprinterCreditMessageHandler) notify(data *SprinterCreditData) error {
	if data.Coordinator != peer.ID("") {
		return nil
	}

	data.Coordinator = h.host.ID()
	msgBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return h.comm.Broadcast(
		h.host.Peerstore().Peers(),
		msgBytes,
		comm.SprinterCreditMsg,
		fmt.Sprintf("%d-%s", h.chainID, comm.SprinterCreditSessionID))
}
