package lighter

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	solverConfig "github.com/sprintertech/solver-config/go/config"
)

var (
	ARBITRUM_CHAIN_ID = big.NewInt(42161)
	LIGHTER_CAIP      = "lighter:1"
	ARBITRUM_CAIP     = "eip155:42161"
	USDC              = "usdc"
)

type LighterConfig struct {
	WithdrawalAddress common.Address
	UsdcAddress       common.Address
	RepaymentAddress  string
}

func NewLighterConfig(solverConfig solverConfig.SolverConfig) (*LighterConfig, error) {
	arbitrumConfig, ok := solverConfig.Chains[ARBITRUM_CAIP]
	if !ok {
		return nil, fmt.Errorf("no solver config for id %s", ARBITRUM_CAIP)
	}

	usdcConfig, ok := arbitrumConfig.Tokens[USDC]
	if !ok {
		return nil, fmt.Errorf("usdc not configured")
	}

	withdrawalAddress, ok := solverConfig.ProtocolsMetadata.Lighter.FastWithdrawalContract[ARBITRUM_CAIP]
	if !ok {
		return nil, fmt.Errorf("withdrawal address not configured")
	}

	return &LighterConfig{
		WithdrawalAddress: common.HexToAddress(withdrawalAddress),
		RepaymentAddress:  solverConfig.ProtocolsMetadata.Lighter.RepaymentAddress,
		UsdcAddress:       common.HexToAddress(usdcConfig.Address),
	}, nil
}
