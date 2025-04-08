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
	"github.com/sprintertech/sprinter-signing/chains/evm"
)

type TokenPricer interface {
	TokenPrice(symbol string) (float64, error)
}

type Watcher struct {
	// chainId -> symbol -> config
	tokens map[uint64]map[string]evm.TokenConfig
	// chainId -> order value usd -> confirmations
	confirmations map[uint64]map[uint64]uint64
	// chainId -> block time
	blocktime   map[uint64]time.Duration
	tokenPricer TokenPricer
}

// waitForConfirmations blocks until the transaction hash has enough on-chain confirmations.
func (w *Watcher) WaitForConfirmations(
	sourceChainId uint64,
	client EventFilterer,
	txHash common.Hash,
	token common.Address,
	amount *big.Int,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), TIMEOUT)
	defer cancel()

	requiredConfirmations, err := w.minimalConfirmations(sourceChainId, token, amount)
	if err != nil {
		return err
	}

	blocktime, ok := w.blocktime[sourceChainId]
	if !ok {
		return fmt.Errorf("no blocktime configured for chain %d", sourceChainId)
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for confirmations")
		default:
			txReceipt, err := client.TransactionReceipt(ctx, txHash)
			if err != nil {
				log.Warn().Msgf("Error fetching transaction receipt: %v\n", err)
				time.Sleep(blocktime)
				continue
			}

			if txReceipt == nil {
				time.Sleep(blocktime)
				continue
			}

			currentBlock, err := client.LatestBlock()
			if err != nil {
				log.Warn().Msgf("Error fetching current block: %v\n", err)
				time.Sleep(blocktime)
				continue
			}

			confirmations := new(big.Int).Sub(currentBlock, txReceipt.BlockNumber)
			if confirmations.Cmp(new(big.Int).SetUint64(requiredConfirmations)) != -1 {
				return nil
			}

			// nolint:gosec
			duration := time.Duration(uint64(blocktime) * (requiredConfirmations - confirmations.Uint64()))
			log.Debug().Msgf("Waiting for tx %s for %s", txHash, duration)
			time.Sleep(duration)
		}
	}
}

// TokenConfig fetches the token configuration and symbol for the given chain
func (w *Watcher) TokenConfig(chain uint64, token common.Address) (string, evm.TokenConfig, error) {
	tokens, ok := w.tokens[chain]
	if !ok {
		return "", evm.TokenConfig{}, fmt.Errorf("no token config for chain %d", chain)
	}

	for symbol, c := range tokens {
		if c.Address == token {
			return symbol, c, nil
		}
	}

	return "", evm.TokenConfig{}, fmt.Errorf("token %s not supported", token.Hex())
}

// minimalConfirmations calculates the minimal confirmations needed to wait for execution
// of an order based on order size
func (w *Watcher) minimalConfirmations(sourceChainId uint64, token common.Address, amount *big.Int) (uint64, error) {
	symbol, c, err := w.TokenConfig(sourceChainId, token)
	if err != nil {
		return 0, err
	}

	price, err := w.tokenPricer.TokenPrice(symbol)
	if err != nil {
		return 0, err
	}

	orderValueInt := new(big.Int)
	orderValueInt, _ = new(big.Float).Quo(
		new(big.Float).Mul(big.NewFloat(price), new(big.Float).SetInt(amount)),
		new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(c.Decimals)), nil)),
	).Int(orderValueInt)

	confirmations, ok := w.confirmations[sourceChainId]
	if !ok {
		return 0, fmt.Errorf("no confirmations for chain %d configured", sourceChainId)
	}
	buckets := slices.Collect(maps.Keys(confirmations))
	slices.Sort(buckets)
	for _, bucket := range buckets {
		if orderValueInt.Cmp(new(big.Int).SetUint64(bucket)) < 0 {
			return confirmations[bucket], nil
		}
	}

	return 0, fmt.Errorf("order value %f exceeds confirmation buckets", orderValueInt)
}
