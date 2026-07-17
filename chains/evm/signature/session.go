package signature

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// BorrowSessionID keys the signing session and cache by the digest fields not already fixed by the deposit.
func BorrowSessionID(
	chainID uint64, depositID string, deadline uint64, caller common.Address,
	borrowAmount *big.Int, liquidityPool common.Address, repaymentChainID uint64,
) string {
	return fmt.Sprintf("%d-%s-%d-%s-%s-%s-%d",
		chainID, depositID, deadline, caller.Hex(),
		borrowAmount.String(), liquidityPool.Hex(), repaymentChainID)
}
