package tron

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	tronaddress "github.com/fbsobreira/gotron-sdk/pkg/address"
)

func ToCommonAddress(addr string) (common.Address, error) {
	isHex := common.IsHexAddress(addr)
	if isHex {
		return common.HexToAddress(addr), nil
	}

	decoded, err := tronaddress.Base58ToAddress(addr)
	if err != nil {
		return common.Address{}, fmt.Errorf("invalid tron address %q: %w", addr, err)
	}

	return common.BytesToAddress(decoded[1:]), nil
}
