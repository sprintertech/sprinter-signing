// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package evm

import (
	"fmt"
	"math/big"
	"time"

	"github.com/creasty/defaults"
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
	BlockConfirmations *big.Int
	BlockInterval      *big.Int
	BlockRetryInterval time.Duration
}

type RawEVMConfig struct {
	chain.GeneralChainConfig `mapstructure:",squash"`
	Admin                    string `mapstructure:"admin"`
	LiqudityPool             string `mapstructure:"liquidityPool"`
	AcrossPool               string `mapstructure:"acrossPool"`
	BlockConfirmations       int64  `mapstructure:"blockConfirmations" default:"10"`
	BlockInterval            int64  `mapstructure:"blockInterval" default:"5"`
	BlockRetryInterval       uint64 `mapstructure:"blockRetryInterval" default:"5"`
}

func (c *RawEVMConfig) Validate() error {
	if err := c.GeneralChainConfig.Validate(); err != nil {
		return err
	}
	if c.BlockConfirmations < 1 {
		return fmt.Errorf("blockConfirmations has to be >=1")
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

	c.GeneralChainConfig.ParseFlags()
	config := &EVMConfig{
		GeneralChainConfig: c.GeneralChainConfig,
		Admin:              c.Admin,
		LiqudityPool:       c.LiqudityPool,
		AcrossPool:         c.AcrossPool,
		// nolint:gosec
		BlockRetryInterval: time.Duration(c.BlockRetryInterval) * time.Second,
		BlockConfirmations: big.NewInt(c.BlockConfirmations),
		BlockInterval:      big.NewInt(c.BlockInterval),
	}

	return config, nil
}
