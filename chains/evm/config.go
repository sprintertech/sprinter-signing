// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package evm

import (
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/creasty/defaults"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mitchellh/mapstructure"

	"github.com/sprintertech/sprinter-signing/config"
	"github.com/sprintertech/sprinter-signing/config/chain"
)

type EVMConfig struct {
	GeneralChainConfig chain.GeneralChainConfig
	Admin              string
	AcrossPool         string
	HubPool            string
	MayanSwift         string
	LiquidityPool      string
	Tokens             map[string]config.TokenConfig
	// usd bucket -> confirmations
	ConfirmationsByValue map[uint64]uint64
	BlockInterval        *big.Int
	BlockRetryInterval   time.Duration
}

type RawEVMConfig struct {
	chain.GeneralChainConfig `mapstructure:",squash"`
	Admin                    string                 `mapstructure:"admin"`
	LiquidityPool            string                 `mapstructure:"liquidityPool"`
	AcrossPool               string                 `mapstructure:"acrossPool"`
	MayanSwift               string                 `mapstructure:"mayanSwift"`
	HubPool                  string                 `mapstructure:"hubPool"`
	Tokens                   map[string]interface{} `mapstructure:"tokens"`
	ConfirmationsByValue     map[string]interface{} `mapstructure:"confirmationsByValue"`
	BlockInterval            int64                  `mapstructure:"blockInterval" default:"5"`
	BlockRetryInterval       uint64                 `mapstructure:"blockRetryInterval" default:"5"`
}

func (c *RawEVMConfig) Validate() error {
	if err := c.GeneralChainConfig.Validate(); err != nil {
		return err
	}
	return nil
}

// NewEVMConfig decodes and validates an instance of an EVMConfig from
// raw chain config
func NewEVMConfig(chainConfig map[string]interface{}) (*EVMConfig, error) {
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

	tokens := make(map[string]config.TokenConfig)
	for s, c := range c.Tokens {
		c := c.(map[string]interface{})

		decimals, err := strconv.ParseUint(c["decimals"].(string), 10, 8)
		if err != nil {
			return nil, err
		}

		tc := config.TokenConfig{
			Address:  common.HexToAddress(c["address"].(string)),
			Decimals: uint8(decimals),
		}
		tokens[s] = tc
	}

	confirmations := make(map[uint64]uint64)
	for usd, confirmation := range c.ConfirmationsByValue {
		usd, err := strconv.ParseUint(usd, 10, 64)
		if err != nil {
			return nil, err
		}

		confirmation, err := strconv.ParseUint(confirmation.(string), 10, 64)
		if err != nil {
			return nil, err
		}

		if confirmation < 1 {
			return nil, fmt.Errorf("confirmation cannot be lower than 1")
		}

		confirmations[usd] = confirmation
	}

	c.ParseFlags()
	config := &EVMConfig{
		GeneralChainConfig: c.GeneralChainConfig,
		Admin:              c.Admin,
		LiquidityPool:      c.LiquidityPool,
		AcrossPool:         c.AcrossPool,
		HubPool:            c.HubPool,
		MayanSwift:         c.MayanSwift,
		// nolint:gosec
		BlockRetryInterval: time.Duration(c.BlockRetryInterval) * time.Second,
		BlockInterval:      big.NewInt(c.BlockInterval),

		ConfirmationsByValue: confirmations,
		Tokens:               tokens,
	}

	return config, nil
}
