// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package evm_test

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
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
		"endpoint": "ws://domain.com",
		"name":     "evm1",
		"from":     "address",
	})

	s.NotNil(err)
}

func (s *NewEVMConfigTestSuite) Test_InvalidConfirmations() {
	rawConfig := map[string]interface{}{
		"id":          1,
		"endpoint":    "ws://domain.com",
		"name":        "evm1",
		"from":        "address",
		"bridge":      "bridgeAddress",
		"admin":       "adminAddress",
		"frostKeygen": "frostKeygen",
		"acrossPool":  "acrossPool",
		"blockConfirmations": map[string]uint64{
			"1000": 0,
		},
	}

	_, err := evm.NewEVMConfig(rawConfig)

	s.NotNil(err)
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
		"acrossPool":  "acrossPool",
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
		BlockInterval:      big.NewInt(5),
		BlockRetryInterval: time.Duration(5) * time.Second,
		Admin:              "adminAddress",
		AcrossPool:         "acrossPool",
		BlockConfirmations: make(map[uint64]uint64),
		Tokens:             make(map[string]common.Address),
	})
}

func (s *NewEVMConfigTestSuite) Test_ValidConfigWithCustomTxParams() {
	rawConfig := map[string]interface{}{
		"id":            1,
		"endpoint":      "ws://domain.com",
		"name":          "evm1",
		"from":          "address",
		"admin":         "adminAddress",
		"liquidityPool": "pool",
		"acrossPool":    "acrossPool",

		"maxGasPrice":           1000,
		"gasMultiplier":         1000,
		"gasIncreasePercentage": 20,
		"gasLimit":              1000,
		"transferGas":           300000,
		"startBlock":            1000,
		"blockRetryInterval":    10,
		"blockInterval":         2,
		"blockConfirmations": map[string]uint64{
			"1000": 5,
			"2000": 10,
		},
		"tokens": map[string]string{
			"usdc": "0xdBBE3D8c2d2b22A2611c5A94A9a12C2fCD49Eb29",
		},
	}

	expectedBlockConfirmations := make(map[uint64]uint64)
	expectedBlockConfirmations[1000] = 5
	expectedBlockConfirmations[2000] = 10

	expectedTokens := make(map[string]common.Address)
	expectedTokens["usdc"] = common.HexToAddress("0xdBBE3D8c2d2b22A2611c5A94A9a12C2fCD49Eb29")

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
		BlockInterval:      big.NewInt(2),
		BlockRetryInterval: time.Duration(10) * time.Second,
		Admin:              "adminAddress",
		LiqudityPool:       "pool",
		AcrossPool:         "acrossPool",
		BlockConfirmations: expectedBlockConfirmations,
		Tokens:             expectedTokens,
	})
}
