package lifi

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/sprintertech/lifi-solver/pkg/protocols/lifi"
	contracts "github.com/sprintertech/lifi-solver/pkg/protocols/lifi/contracts"
)

const (
	TRANSACTION_TIMEOUT = 30 * time.Second
	OpenEventTopic      = "0x9ff74bd56d00785b881ef9fa3f03d7b598686a39a9bcff89a6008db588b18a7b"
)

type ReceiptFetcher interface {
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

type LifiEventFetcher struct {
	clients      map[uint64]ReceiptFetcher
	inputSettler common.Address
}

func NewLifiEventFetcher(
	clients map[uint64]ReceiptFetcher,
	inputSettler common.Address,
) *LifiEventFetcher {
	return &LifiEventFetcher{
		clients:      clients,
		inputSettler: inputSettler,
	}
}

func (h *LifiEventFetcher) Order(ctx context.Context, sourceChainID uint64, hash common.Hash, orderID common.Hash) (*lifi.LifiOrder, error) {
	client, ok := h.clients[sourceChainID]
	if !ok {
		return nil, fmt.Errorf("no client configured for source chain %d", sourceChainID)
	}

	ctx, cancel := context.WithTimeout(ctx, TRANSACTION_TIMEOUT)
	defer cancel()

	log, err := h.fetchOpenEvent(ctx, client, hash, orderID)
	if err != nil {
		return nil, err
	}

	return contracts.ParseOpenEvent(log, h.inputSettler.Hex())
}

func (h *LifiEventFetcher) fetchOpenEvent(ctx context.Context, client ReceiptFetcher, hash common.Hash, orderID common.Hash) (*types.Log, error) {
	receipt, err := client.TransactionReceipt(ctx, hash)
	if err != nil {
		return nil, err
	}

	for _, l := range receipt.Logs {
		if l.Removed {
			continue
		}

		if len(l.Topics) < 2 {
			continue
		}

		if l.Address != h.inputSettler {
			continue
		}

		if l.Topics[0] != common.HexToHash(OpenEventTopic) {
			continue
		}

		if l.Topics[1] != orderID {
			continue
		}

		return l, nil
	}

	return nil, fmt.Errorf("order with id %s not found", orderID.Hex())
}
