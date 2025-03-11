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

	"github.com/sprintertech/sprinter-signing/config/chain"
)

type HandlerConfig struct {
	Address string
	Type    string
}

type EVMConfig struct {
	GeneralChainConfig chain.GeneralChainConfig
	Admin              string
	AcrossPool         string
	LiqudityPool       string
	Tokens             map[string]common.Address
	// usd bucket -> confirmations
	BlockConfirmations map[uint64]uint64
	BlockInterval      *big.Int
	BlockRetryInterval time.Duration
}

type RawEVMConfig struct {
	chain.GeneralChainConfig `mapstructure:",squash"`
	Admin                    string                 `mapstructure:"admin"`
	LiqudityPool             string                 `mapstructure:"liquidityPool"`
	AcrossPool               string                 `mapstructure:"acrossPool"`
	Tokens                   map[string]interface{} `mapstructure:"tokens"`
	BlockConfirmations       map[string]interface{} `mapstructure:"blockConfirmations"`
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

	tokens := make(map[string]common.Address)
	for s, a := range c.Tokens {
		tokens[s] = common.HexToAddress(a.(string))
	}

	confirmations := make(map[uint64]uint64)
	for usd, confirmation := range c.BlockConfirmations {
		usd, err := strconv.ParseUint(usd, 10, 64)
		if err != nil {
			return nil, err
		}

		confirmation := confirmation.(uint64)
		if confirmation < 1 {
			return nil, fmt.Errorf("confirmation cannot be lower than 1")
		}

		confirmations[usd] = confirmation
	}

	c.GeneralChainConfig.ParseFlags()
	config := &EVMConfig{
		GeneralChainConfig: c.GeneralChainConfig,
		Admin:              c.Admin,
		LiqudityPool:       c.LiqudityPool,
		AcrossPool:         c.AcrossPool,
		// nolint:gosec
		BlockRetryInterval: time.Duration(c.BlockRetryInterval) * time.Second,
		BlockInterval:      big.NewInt(c.BlockInterval),

		BlockConfirmations: confirmations,
		Tokens:             tokens,
	}

	return config, nil
}
