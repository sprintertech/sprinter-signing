package config

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	solverConfig "github.com/sprintertech/solver-config/go/config"
)

const (
	ConfigWatcherInterval = time.Minute * 1
)

// StartConfigWatcher starts a goroutine that periodically checks for config changes.
// Panics to induce a restart if the config has changed
func StartConfigWatcher(ctx context.Context, config *solverConfig.SolverConfig, opts []solverConfig.Option) (<-chan struct{}, error) {
	configChanged := make(chan struct{}, 1)
	configHash, err := calculateConfigHash(config)
	if err != nil {
		return configChanged, err
	}

	go func() {
		ticker := time.NewTicker(ConfigWatcherInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				hasChanged, err := hasConfigChanged(ctx, configHash, opts)
				if err != nil {
					log.Warn().Msgf("Failed checking has config changed: %s", err)
					continue
				}

				if hasChanged {
					configChanged <- struct{}{}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return configChanged, nil
}

// hasConfigChanged fetches config and returns true if it has changed
func hasConfigChanged(ctx context.Context, initialHash string, opts []solverConfig.Option) (bool, error) {
	solverConfig, err := solverConfig.FetchSolverConfig(ctx, opts...)
	if err != nil {
		return false, err
	}

	newHash, err := calculateConfigHash(solverConfig)
	if err != nil {
		return false, err
	}

	if initialHash == newHash {
		return false, nil
	}

	return true, nil
}

func calculateConfigHash(config *solverConfig.SolverConfig) (string, error) {
	jsonBytes, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}

	hash := sha256.Sum256(jsonBytes)
	return fmt.Sprintf("%x", hash), nil
}
