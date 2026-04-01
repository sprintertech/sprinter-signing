// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package chains

import (
	"fmt"
	"math/big"
)

// CalculateStartingBlock returns first block number (smaller or equal) that is dividable with block confirmations
func CalculateStartingBlock(startBlock *big.Int, blockConfirmations *big.Int) (*big.Int, error) {
	if startBlock == nil || blockConfirmations == nil {
		return nil, fmt.Errorf("startBlock or blockConfirmations can not be nill when calculating CalculateStartingBlock")
	}
	mod := big.NewInt(0).Mod(startBlock, blockConfirmations)
	startBlock.Sub(startBlock, mod)
	return startBlock, nil
}

// ScaleTokenAmount scales amount from srcDecimals to dstDecimals.
// Formula: amount / 10^(srcDecimals-dstDecimals)
// When src > dst (e.g. BSC USDC 18 -> Base USDC 6): divides by 10^12.
// When src < dst: exponent is negative, so effectively multiplies.
func ScaleTokenAmount(amount *big.Int, srcDecimals, dstDecimals int64) *big.Int {
	if srcDecimals == dstDecimals {
		return amount
	}
	if srcDecimals > dstDecimals {
		scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(srcDecimals-dstDecimals), nil)
		return new(big.Int).Div(amount, scale)
	}
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(dstDecimals-srcDecimals), nil)
	return new(big.Int).Mul(amount, scale)
}
