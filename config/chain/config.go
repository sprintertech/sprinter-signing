// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package chain

import (
	"fmt"

	"github.com/spf13/viper"
	"github.com/sprintertech/sprinter-signing/config"
)

type GeneralChainConfig struct {
	Name               string  `mapstructure:"name"`
	Id                 *uint64 `mapstructure:"id"`
	Endpoint           string  `mapstructure:"endpoint"`
	Type               string  `mapstructure:"type"`
	BlockstorePath     string  `mapstructure:"blockstorePath"`
	Blocktime          uint64  `mapstructure:"blocktime" default:"12"`
	BlockConfirmations uint64  `default:"5"`
	Key                string
	Insecure           bool
}

func (c *GeneralChainConfig) Validate() error {
	// viper defaults to 0 for not specified ints
	if c.Id == nil {
		return fmt.Errorf("required field domain.Id empty for chain %v", c.Id)
	}
	if c.Endpoint == "" {
		return fmt.Errorf("required field chain.Endpoint empty for chain %v", *c.Id)
	}
	if c.Name == "" {
		return fmt.Errorf("required field chain.Name empty for chain %v", *c.Id)
	}
	return nil
}

func (c *GeneralChainConfig) ParseFlags() {
	blockstore := viper.GetString(config.BlockstoreFlagName)
	if blockstore != "" {
		c.BlockstorePath = blockstore
	}
}
