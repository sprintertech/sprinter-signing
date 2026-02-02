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
	client       ReceiptFetcher
	inputSettler common.Address
}

func NewLifiEventFetcher(
	client ReceiptFetcher,
	inputSettler common.Address,
) *LifiEventFetcher {
	return &LifiEventFetcher{
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
		return nil, fmt.Errorf("no receipt found for hash %s, %w", hash.Hex(), err)
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
