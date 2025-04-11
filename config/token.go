package config

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

type TokenConfig struct {
	Address  common.Address
	Decimals uint8
}

type TokenStore struct {
	Tokens map[uint64]map[string]TokenConfig
}

func (s *TokenStore) ConfigByAddress(chainID uint64, address common.Address) (string, TokenConfig, error) {
	tokens, ok := s.Tokens[chainID]
	if !ok {
		return "", TokenConfig{}, fmt.Errorf("no tokens for chain %d", chainID)
	}

	for symbol, c := range tokens {
		if c.Address == address {
			return symbol, c, nil
		}
	}

	return "", TokenConfig{}, fmt.Errorf("no symbol for address %s", address.Hex())
}

func (s *TokenStore) ConfigBySymbol(chainID uint64, symbol string) (TokenConfig, error) {
	tokens, ok := s.Tokens[chainID]
	if !ok {
		return TokenConfig{}, fmt.Errorf("no tokens for chain %d", chainID)
	}

	c, ok := tokens[symbol]
	if !ok {
		return TokenConfig{}, fmt.Errorf("no config for token %s", symbol)
	}

	return c, nil
}
