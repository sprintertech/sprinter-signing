package message

import (
	"context"
	"fmt"
	"maps"
	"math/big"
	"slices"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"github.com/sprintertech/sprinter-signing/config"
)

type TokenPricer interface {
	TokenPrice(symbol string) (float64, error)
}

type Watcher struct {
	client        EventFilterer
	tokenStore    config.TokenStore
	confirmations map[uint64]uint64
	blocktime     time.Duration
	tokenPricer   TokenPricer
}

func NewWatcher(
	client EventFilterer,
	tokenPricer TokenPricer,
	tokenStore config.TokenStore,
	confirmations map[uint64]uint64,
	blocktime time.Duration,
) *Watcher {
	return &Watcher{
		client:        client,
		tokenStore:    tokenStore,
		confirmations: confirmations,
		blocktime:     blocktime,
		tokenPricer:   tokenPricer,
	}
}

// WaitForTokenConfirmations blocks until the transaction hash has enough on-chain confirmations.
func (w *Watcher) WaitForTokenConfirmations(
	ctx context.Context,
	chainID uint64,
	txHash common.Hash,
	token common.Address,
	amount *big.Int,
) error {
	ctx, cancel := context.WithTimeout(ctx, TIMEOUT)
	defer cancel()

	orderValue, err := w.orderValue(chainID, token, amount)
	if err != nil {
		return err
	}

	requiredConfirmations, err := w.minimalConfirmations(orderValue)
	if err != nil {
		return err
	}

	return w.wait(ctx, txHash, requiredConfirmations)
}

// WaitForConfirmations blocks until the transaction hash has enough on-chain confirmations.
func (w *Watcher) WaitForOrderConfirmations(
	ctx context.Context,
	chainID uint64,
	txHash common.Hash,
	orderValue float64,
) error {
	ctx, cancel := context.WithTimeout(ctx, TIMEOUT)
	defer cancel()

	orderValueInt, _ := big.NewFloat(orderValue).Int(nil)
	requiredConfirmations, err := w.minimalConfirmations(orderValueInt)
	if err != nil {
		return err
	}

	return w.wait(ctx, txHash, requiredConfirmations)
}

func (w *Watcher) wait(ctx context.Context, txHash common.Hash, requiredConfirmations uint64) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for confirmations")
		default:
			txReceipt, err := w.client.TransactionReceipt(ctx, txHash)
			if err != nil {
				log.Warn().Msgf("Error fetching transaction receipt: %v\n", err)
				time.Sleep(w.blocktime)
				continue
			}

			if txReceipt == nil {
				time.Sleep(w.blocktime)
				continue
			}

			currentBlock, err := w.client.LatestBlock()
			if err != nil {
				log.Warn().Msgf("Error fetching current block: %v\n", err)
				time.Sleep(w.blocktime)
				continue
			}

			confirmations := new(big.Int).Sub(currentBlock, txReceipt.BlockNumber)
			if confirmations.Cmp(new(big.Int).SetUint64(requiredConfirmations)) != -1 {
				return nil
			}

			// nolint:gosec
			duration := time.Duration(uint64(w.blocktime) * (requiredConfirmations - confirmations.Uint64()))
			log.Debug().Msgf("Waiting for tx %s for %s", txHash, duration)
			time.Sleep(duration)
		}
	}
}

// minimalConfirmations calculates the minimal confirmations needed to wait for execution
// of an order based on order size
func (w *Watcher) minimalConfirmations(orderValue *big.Int) (uint64, error) {
	buckets := slices.Collect(maps.Keys(w.confirmations))
	slices.Sort(buckets)
	for _, bucket := range buckets {
		if orderValue.Cmp(new(big.Int).SetUint64(bucket)) < 0 {
			return w.confirmations[bucket], nil
		}
	}

	return 0, fmt.Errorf("order value %f exceeds confirmation buckets", orderValue)
}

func (w *Watcher) orderValue(chainID uint64, token common.Address, amount *big.Int) (*big.Int, error) {
	symbol, c, err := w.tokenStore.ConfigByAddress(chainID, token)
	if err != nil {
		return nil, err
	}

	price, err := w.tokenPricer.TokenPrice(symbol)
	if err != nil {
		return nil, err
	}

	orderValueInt := new(big.Int)
	orderValueInt, _ = new(big.Float).Quo(
		new(big.Float).Mul(big.NewFloat(price), new(big.Float).SetInt(amount)),
		new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(c.Decimals)), nil)),
	).Int(orderValueInt)
	return orderValueInt, nil
}
