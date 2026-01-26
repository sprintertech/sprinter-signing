package lifi

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/sprintertech/lifi-solver/pkg/protocols/lifi"
	contracts "github.com/sprintertech/lifi-solver/pkg/protocols/lifi/contracts"
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/events"
)

const (
	TRANSACTION_TIMEOUT = 30 * time.Second
)

type ReceiptFetcher interface {
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

type LifiEventFetcher struct {
	chainID      uint64
	client       ReceiptFetcher
	inputSettler common.Address
}

func NewLifiEventFetcher(
	chainID uint64,
	client ReceiptFetcher,
	inputSettler common.Address,
) *LifiEventFetcher {
	return &LifiEventFetcher{
		chainID:      chainID,
		client:       client,
		inputSettler: inputSettler,
	}
}

func (h *LifiEventFetcher) Order(ctx context.Context, hash common.Hash, orderID common.Hash) (*lifi.LifiOrder, error) {
	ctx, cancel := context.WithTimeout(ctx, TRANSACTION_TIMEOUT)
	defer cancel()

	log, err := h.fetchOpenEvent(ctx, hash, orderID)
	if err != nil {
		return nil, err
	}

	return contracts.ParseOpenEvent(log, h.inputSettler.Hex())
}

func (h *LifiEventFetcher) fetchOpenEvent(ctx context.Context, hash common.Hash, orderID common.Hash) (*types.Log, error) {
	receipt, err := h.client.TransactionReceipt(ctx, hash)
	if err != nil {
		return nil, err
	}

	for _, l := range receipt.Logs {
		if l.Removed {
			continue
		}

		if len(l.Topics) < 3 {
			continue
		}

		if l.Topics[0] != events.LifiOpenSig.GetTopic() {
			continue
		}

		if l.Topics[1] != orderID {
			continue
		}

		return l, nil
	}

	return nil, fmt.Errorf("order with id %s not found", orderID.Hex())
}
