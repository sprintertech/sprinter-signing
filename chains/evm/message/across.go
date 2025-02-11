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
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/events"
	"github.com/sprintertech/sprinter-signing/comm"
	"github.com/sprintertech/sprinter-signing/tss"
	"github.com/sprintertech/sprinter-signing/tss/ecdsa/signing"
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
	depositId *big.Int
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

	coordinator   *tss.Coordinator
	host          host.Host
	communication comm.Communication
	fetcher       signing.SaveDataFetcher

	resultChn chan interface{}
}

// HandleMessage finds the Across deposit with the according deposit ID and starts
// the MPC signature process for it. The result will be saved into the signature
// cache through the result channel.
func (h *AcrossMessageHandler) HandleMessage(m *message.Message) (*proposal.Proposal, error) {
	data := m.Data.(AcrossData)
	d, err := h.deposit(data.depositId)
	if err != nil {
		return nil, err
	}

	signing, err := signing.NewSigning(
		d.DepositId,
		data.depositId.Text(16),
		data.depositId.Text(16),
		h.host,
		h.communication,
		h.fetcher)
	if err != nil {
		return nil, err
	}

	err = h.coordinator.Execute(context.Background(), []tss.TssProcess{signing}, h.resultChn, h.host.ID())
	if err != nil {
		return nil, err
	}
	return nil, nil
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
