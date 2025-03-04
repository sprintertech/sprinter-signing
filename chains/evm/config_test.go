// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package evm_test

import (
	"math/big"
	"testing"
	"time"

	"github.com/sprintertech/sprinter-signing/chains/evm"
	"github.com/sprintertech/sprinter-signing/config/chain"
	"github.com/stretchr/testify/suite"
)

type NewEVMConfigTestSuite struct {
	suite.Suite
}

func TestRunNewEVMConfigTestSuite(t *testing.T) {
	suite.Run(t, new(NewEVMConfigTestSuite))
}

func (s *NewEVMConfigTestSuite) Test_FailedDecode() {
	_, err := evm.NewEVMConfig(map[string]interface{}{
		"gasLimit": "invalid",
	})

	s.NotNil(err)
}

func (s *NewEVMConfigTestSuite) Test_FailedGeneralConfigValidation() {
	_, err := evm.NewEVMConfig(map[string]interface{}{})

	s.NotNil(err)
}

func (s *NewEVMConfigTestSuite) Test_FailedEVMConfigValidation() {
	_, err := evm.NewEVMConfig(map[string]interface{}{
		"id":       1,
		"endpoint": "ws://domain.com",
		"name":     "evm1",
		"from":     "address",
	})

	s.NotNil(err)
}

func (s *NewEVMConfigTestSuite) Test_InvalidBlockConfirmation() {
	_, err := evm.NewEVMConfig(map[string]interface{}{
		"id":                 1,
		"endpoint":           "ws://domain.com",
		"name":               "evm1",
		"from":               "address",
		"admin":              "bridgeAddress",
		"blockConfirmations": -1,
	})

	s.NotNil(err)
	s.Equal(err.Error(), "blockConfirmations has to be >=1")
}

func (s *NewEVMConfigTestSuite) Test_ValidConfig() {
	rawConfig := map[string]interface{}{
		"id":          1,
		"endpoint":    "ws://domain.com",
		"name":        "evm1",
		"from":        "address",
		"bridge":      "bridgeAddress",
		"admin":       "adminAddress",
		"frostKeygen": "frostKeygen",
	}

	actualConfig, err := evm.NewEVMConfig(rawConfig)

	id := new(uint64)
	*id = 1
	s.Nil(err)
	s.Equal(*actualConfig, evm.EVMConfig{
		GeneralChainConfig: chain.GeneralChainConfig{
			Name:     "evm1",
			Endpoint: "ws://domain.com",
			Id:       id,
		},
		BlockConfirmations: big.NewInt(10),
		BlockInterval:      big.NewInt(5),
		BlockRetryInterval: time.Duration(5) * time.Second,
		Admin:              "adminAddress",
	})
}

func (s *NewEVMConfigTestSuite) Test_ValidConfigWithCustomTxParams() {
	rawConfig := map[string]interface{}{
		"id":            1,
		"endpoint":      "ws://domain.com",
		"name":          "evm1",
		"from":          "address",
		"bridge":        "bridgeAddress",
		"admin":         "adminAddress",
		"liquidityPool": "pool",
		"retry":         "retryAddress",
		"frostKeygen":   "frostKeygen",
		"handlers": []evm.HandlerConfig{
			{
				Type:    "erc20",
				Address: "address1",
			},
			{
				Type:    "erc721",
				Address: "address2",
			},
		},
		"maxGasPrice":           1000,
		"gasMultiplier":         1000,
		"gasIncreasePercentage": 20,
		"gasLimit":              1000,
		"transferGas":           300000,
		"startBlock":            1000,
		"blockConfirmations":    10,
		"blockRetryInterval":    10,
		"blockInterval":         2,
	}

	actualConfig, err := evm.NewEVMConfig(rawConfig)

	id := new(uint64)
	*id = 1
	s.Nil(err)
	s.Equal(*actualConfig, evm.EVMConfig{
		GeneralChainConfig: chain.GeneralChainConfig{
			Name:     "evm1",
			Endpoint: "ws://domain.com",
			Id:       id,
		},
		BlockConfirmations: big.NewInt(10),
		BlockInterval:      big.NewInt(2),
		BlockRetryInterval: time.Duration(10) * time.Second,
		Admin:              "adminAddress",
		LiqudityPool:       "pool",
	})
}
