package across

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/consts"
	"github.com/sprintertech/sprinter-signing/chains/evm/calls/events"
	"github.com/sprintertech/sprinter-signing/config"
)

const (
	ZERO_HASH           = "0000000000000000000000000000000000000000000000000000000000000000"
	TRANSACTION_TIMEOUT = 30 * time.Second
)

type EventFilterer interface {
	FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error)
	LatestBlock() (*big.Int, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

type TokenMatcher interface {
	DestinationToken(destinationChainId *big.Int, symbol string) (common.Address, error)
}

type AcrossDepositFetcher struct {
	chainID      uint64
	tokenStore   config.TokenStore
	client       EventFilterer
	tokenMatcher TokenMatcher
}

func (h *AcrossDepositFetcher) Deposit(ctx context.Context, hash common.Hash, depositID *big.Int) (*events.AcrossDeposit, error) {
	ctx, cancel := context.WithTimeout(ctx, TRANSACTION_TIMEOUT)
	defer cancel()

	return h.fetchDepositByHash(ctx, hash, depositID)
}

func (h *AcrossDepositFetcher) fetchDepositByHash(ctx context.Context, hash common.Hash, depositID *big.Int) (*events.AcrossDeposit, error) {
	receipt, err := h.client.TransactionReceipt(ctx, hash)
	if err != nil {
		return nil, err
	}

	for _, l := range receipt.Logs {
		if l.Removed {
			continue
		}

		if l.Topics[0] != events.AcrossDepositSig.GetTopic() {
			continue
		}

		if l.Topics[2] != common.HexToHash(common.Bytes2Hex(common.LeftPadBytes(depositID.Bytes(), 32))) {
			continue
		}

		d, err := h.parseDeposit(*l)
		if err != nil {
			return nil, err
		}
		return d, nil
	}

	return nil, fmt.Errorf("deposit with id %s not found", depositID)
}

func (h *AcrossDepositFetcher) parseDeposit(l types.Log) (*events.AcrossDeposit, error) {
	d := &events.AcrossDeposit{}
	err := consts.SpokePoolABI.UnpackIntoInterface(d, "FundsDeposited", l.Data)
	if err != nil {
		return nil, err
	}

	if len(l.Topics) < 4 {
		return nil, fmt.Errorf("across deposit missing topics")
	}

	d.DestinationChainId = new(big.Int).SetBytes(l.Topics[1].Bytes())
	d.DepositId = new(big.Int).SetBytes(l.Topics[2].Bytes())
	copy(d.Depositor[:], l.Topics[3].Bytes())

	if common.Bytes2Hex(d.OutputToken[:]) == ZERO_HASH {
		symbol, _, err := h.tokenStore.ConfigByAddress(h.chainID, common.BytesToAddress(d.InputToken[12:]))
		if err != nil {
			return nil, err
		}

		address, err := h.tokenMatcher.DestinationToken(d.DestinationChainId, symbol)
		if err != nil {
			return nil, err
		}

		d.OutputToken = common.BytesToHash(address.Bytes())
	}

	return d, err
}
