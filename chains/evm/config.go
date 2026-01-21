// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package evm

import (
	"fmt"
	"math/big"
	"time"

	"github.com/creasty/defaults"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mitchellh/mapstructure"

	solverConfig "github.com/sprintertech/solver-config/go/config"
	"github.com/sprintertech/sprinter-signing/config"
	"github.com/sprintertech/sprinter-signing/config/chain"
)

type EVMConfig struct {
	GeneralChainConfig chain.GeneralChainConfig
	Admin              string

	AcrossPool        string
	AcrossHubPool     string
	MayanSwift        string
	LifiOutputSettler string
	Repayer           string
	// Liquidator contract per token address
	Liquidators map[common.Address]common.Address

	Tokens map[string]config.TokenConfig
	// usd bucket -> confirmations
	ConfirmationsByValue map[uint64]uint64

	BlockInterval      *big.Int
	BlockRetryInterval time.Duration
}

type RawEVMConfig struct {
	chain.GeneralChainConfig `mapstructure:",squash"`
	Admin                    string `mapstructure:"admin"`
	Repayer                  string `mapstructure:"repayer"`

	BlockInterval      int64  `mapstructure:"blockInterval" default:"5"`
	BlockRetryInterval uint64 `mapstructure:"blockRetryInterval" default:"5"`
}

func (c *RawEVMConfig) Validate() error {
	if err := c.GeneralChainConfig.Validate(); err != nil {
		return err
	}
	return nil
}

// NewEVMConfig decodes and validates an instance of an EVMConfig from
// raw chain config
func NewEVMConfig(chainConfig map[string]interface{}, solverConfig solverConfig.SolverConfig) (*EVMConfig, error) {
	var c RawEVMConfig
	err := mapstructure.Decode(chainConfig, &c)
	if err != nil {
		return nil, err
	}

	err = defaults.Set(&c)
	if err != nil {
		return nil, err
	}

	err = c.Validate()
	if err != nil {
		return nil, err
	}

	id := fmt.Sprintf("eip155:%d", *c.Id)
	sc, ok := solverConfig.Chains[id]
	if !ok {
		return nil, fmt.Errorf("no solver config for id %s", id)
	}

	tokens := make(map[string]config.TokenConfig)
	for s, c := range sc.Tokens {
		tc := config.TokenConfig{
			Address: common.HexToAddress(c.Address),
			// nolint:gosec
			Decimals: uint8(c.Decimals),
		}
		tokens[s] = tc
	}

	confirmations := make(map[uint64]uint64)
	for _, confirmation := range sc.Confirmations {
		// nolint:gosec
		confirmations[uint64(confirmation.MaxAmountUSD)] = uint64(confirmation.Confirmations)
	}

	c.ParseFlags()
	config := &EVMConfig{
		GeneralChainConfig: c.GeneralChainConfig,
		Admin:              c.Admin,
		Repayer:            solverConfig.ProtocolsMetadata.Sprinter.Repayer[id],

		AcrossPool:    solverConfig.ProtocolsMetadata.Across.SpokePools[id],
		AcrossHubPool: solverConfig.ProtocolsMetadata.Across.HubPools[id],

		MayanSwift: solverConfig.ProtocolsMetadata.Mayan.SwiftContracts[id],

		LifiOutputSettler: solverConfig.ProtocolsMetadata.Lifi.OutputSettler,

		//TODO: fill liquidators when solver config is ready

		// nolint:gosec
		BlockRetryInterval: time.Duration(c.BlockRetryInterval) * time.Second,
		BlockInterval:      big.NewInt(c.BlockInterval),

		ConfirmationsByValue: confirmations,
		Tokens:               tokens,
	}

	return config, nil
}
