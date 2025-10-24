// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package lighter_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	solverConfig "github.com/sprintertech/solver-config/go/config"
	"github.com/sprintertech/sprinter-signing/chains/lighter"
	"github.com/stretchr/testify/suite"
)

type NewLighterConfigTestSuite struct {
	suite.Suite
}

func TestRunNewLighterConfigTestSuite(t *testing.T) {
	suite.Run(t, new(NewLighterConfigTestSuite))
}

func (s *NewLighterConfigTestSuite) Test_ArbitrumNotConfigured() {
	solverChains := make(map[string]solverConfig.Chain)
	_, err := lighter.NewLighterConfig(solverConfig.SolverConfig{
		Chains:            solverChains,
		ProtocolsMetadata: solverConfig.ProtocolsMetadata{},
	})

	s.NotNil(err)
}

func (s *NewLighterConfigTestSuite) Test_UsdcNotConfigured() {
	solverChains := make(map[string]solverConfig.Chain)
	solverChains["eip155:42161"] = solverConfig.Chain{}

	_, err := lighter.NewLighterConfig(solverConfig.SolverConfig{
		Chains:            solverChains,
		ProtocolsMetadata: solverConfig.ProtocolsMetadata{},
	})

	s.NotNil(err)
}

func (s *NewLighterConfigTestSuite) Test_WithdrawalAddressNotConfigured() {
	tokens := make(map[string]solverConfig.Token)
	tokens["usdc"] = solverConfig.Token{
		Address:  "address",
		Decimals: 6,
	}

	solverChains := make(map[string]solverConfig.Chain)
	solverChains["eip155:42161"] = solverConfig.Chain{
		Tokens: tokens,
	}

	_, err := lighter.NewLighterConfig(solverConfig.SolverConfig{
		Chains: solverChains,
		ProtocolsMetadata: solverConfig.ProtocolsMetadata{
			Lighter: &solverConfig.Lighter{},
		},
	})

	s.NotNil(err)
}

func (s *NewLighterConfigTestSuite) Test_ValidConfig() {
	tokens := make(map[string]solverConfig.Token)
	tokens["usdc"] = solverConfig.Token{
		Address:  "usdc",
		Decimals: 6,
	}

	solverChains := make(map[string]solverConfig.Chain)
	solverChains["eip155:42161"] = solverConfig.Chain{
		Tokens: tokens,
	}

	config, err := lighter.NewLighterConfig(solverConfig.SolverConfig{
		Chains: solverChains,
		ProtocolsMetadata: solverConfig.ProtocolsMetadata{
			Lighter: &solverConfig.Lighter{
				FastWithdrawalContract: map[string]string{
					"eip155:42161": "withdrawal",
				},
			},
		},
	})

	s.Nil(err)
	s.Equal(config, &lighter.LighterConfig{
		WithdrawalAddress: common.HexToAddress("withdrawal"),
		UsdcAddress:       common.HexToAddress("usdc"),
	})
}
