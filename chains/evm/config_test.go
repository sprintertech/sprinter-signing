// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package evm_test

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	solverConfig "github.com/sprintertech/solver-config/go/config"
	"github.com/sprintertech/sprinter-signing/chains/evm"
	"github.com/sprintertech/sprinter-signing/config"
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
	}, solverConfig.SolverConfig{})

	s.NotNil(err)
}

func (s *NewEVMConfigTestSuite) Test_FailedGeneralConfigValidation() {
	_, err := evm.NewEVMConfig(map[string]interface{}{}, solverConfig.SolverConfig{})

	s.NotNil(err)
}

func (s *NewEVMConfigTestSuite) Test_FailedEVMConfigValidation() {
	_, err := evm.NewEVMConfig(map[string]interface{}{
		"endpoint": "ws://domain.com",
		"name":     "evm1",
		"from":     "address",
	}, solverConfig.SolverConfig{})

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
		"confirmationsByValue": map[string]string{
			"1000": "0",
		},
	}

	_, err := evm.NewEVMConfig(rawConfig, solverConfig.SolverConfig{})

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

	solverChains := make(map[string]solverConfig.Chain)
	solverChains["eip155:1"] = solverConfig.Chain{
		Tokens:        make(map[string]solverConfig.Token),
		Confirmations: make([]solverConfig.Confirmations, 0),
	}

	actualConfig, err := evm.NewEVMConfig(rawConfig, solverConfig.SolverConfig{
		Chains: solverChains,
		ProtocolsMetadata: solverConfig.ProtocolsMetadata{
			Across: &solverConfig.AcrossMetadata{
				SpokePools: map[string]string{
					"eip155:1": "acrossPool",
				},
				HubPools: map[string]string{
					"eip155:1": "acrossHubPool",
				},
			},
			Mayan: &solverConfig.MayanMetadata{
				SwiftContracts: map[string]string{
					"eip155:1": "mayanSwift",
				},
			},
			Sprinter: &solverConfig.SprinterMetadata{
				Repayer: map[string]string{
					"eip155:1": "repayer",
				},
			},
		},
	})

	id := new(uint64)
	*id = 1
	s.Nil(err)
	s.Equal(*actualConfig, evm.EVMConfig{
		GeneralChainConfig: chain.GeneralChainConfig{
			Name:               "evm1",
			Endpoint:           "ws://domain.com",
			Id:                 id,
			BlockConfirmations: 5,
			Blocktime:          12,
		},
		BlockInterval:        big.NewInt(5),
		BlockRetryInterval:   time.Duration(5) * time.Second,
		Admin:                "adminAddress",
		Repayer:              "repayer",
		AcrossPool:           "acrossPool",
		AcrossHubPool:        "acrossHubPool",
		MayanSwift:           "mayanSwift",
		ConfirmationsByValue: make(map[uint64]uint64),
		Tokens:               make(map[string]config.TokenConfig),
	})
}

func (s *NewEVMConfigTestSuite) Test_ValidConfigWithCustomTxParams() {
	rawConfig := map[string]interface{}{
		"id":                    1,
		"endpoint":              "ws://domain.com",
		"name":                  "evm1",
		"from":                  "address",
		"admin":                 "adminAddress",
		"acrossPool":            "acrossPool",
		"acrossHubPool":         "hubPool",
		"mayanSwift":            "mayanSwift",
		"maxGasPrice":           1000,
		"gasMultiplier":         1000,
		"gasIncreasePercentage": 20,
		"gasLimit":              1000,
		"transferGas":           300000,
		"startBlock":            1000,
		"blockRetryInterval":    10,
		"blockInterval":         2,
		"blocktime":             10,
		"confirmationsByValue": map[string]string{
			"1000": "5",
			"2000": "10",
		},
		"tokens": map[string]interface{}{
			"usdc": map[string]interface{}{
				"address":  "0xdBBE3D8c2d2b22A2611c5A94A9a12C2fCD49Eb29",
				"decimals": "8",
			},
		},
	}

	expectedBlockConfirmations := make(map[uint64]uint64)
	expectedBlockConfirmations[1000] = 5
	expectedBlockConfirmations[2000] = 10

	expectedTokens := make(map[string]config.TokenConfig)
	expectedTokens["usdc"] = config.TokenConfig{
		Address:  common.HexToAddress("0xdBBE3D8c2d2b22A2611c5A94A9a12C2fCD49Eb29"),
		Decimals: 8,
	}

	solverChains := make(map[string]solverConfig.Chain)
	solverChains["eip155:1"] = solverConfig.Chain{
		Tokens: map[string]solverConfig.Token{
			"usdc": {
				Address:  "0xdBBE3D8c2d2b22A2611c5A94A9a12C2fCD49Eb29",
				Decimals: 8,
			},
		},
		Confirmations: []solverConfig.Confirmations{
			{
				Confirmations: 5,
				MaxAmountUSD:  1000,
			},
			{
				Confirmations: 10,
				MaxAmountUSD:  2000,
			},
		},
	}

	actualConfig, err := evm.NewEVMConfig(rawConfig, solverConfig.SolverConfig{
		Chains: solverChains,
		ProtocolsMetadata: solverConfig.ProtocolsMetadata{
			Across: &solverConfig.AcrossMetadata{
				SpokePools: map[string]string{
					"eip155:1": "acrossPool",
				},
				HubPools: map[string]string{
					"eip155:1": "acrossHubPool",
				},
			},
			Mayan: &solverConfig.MayanMetadata{
				SwiftContracts: map[string]string{
					"eip155:1": "mayanSwift",
				},
			},
			Sprinter: &solverConfig.SprinterMetadata{
				Repayer: map[string]string{
					"eip155:1": "repayer",
				},
			},
		},
	})

	id := new(uint64)
	*id = 1
	s.Nil(err)
	s.Equal(*actualConfig, evm.EVMConfig{
		GeneralChainConfig: chain.GeneralChainConfig{
			Name:               "evm1",
			Endpoint:           "ws://domain.com",
			Id:                 id,
			BlockConfirmations: 5,
			Blocktime:          10,
		},
		BlockInterval:        big.NewInt(2),
		BlockRetryInterval:   time.Duration(10) * time.Second,
		Admin:                "adminAddress",
		AcrossPool:           "acrossPool",
		AcrossHubPool:        "acrossHubPool",
		MayanSwift:           "mayanSwift",
		Repayer:              "repayer",
		ConfirmationsByValue: expectedBlockConfirmations,
		Tokens:               expectedTokens,
	})
}
