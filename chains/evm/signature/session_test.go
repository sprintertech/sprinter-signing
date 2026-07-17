package signature_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/sprintertech/sprinter-signing/chains/evm/signature"
)

func TestBorrowSessionID(t *testing.T) {
	caller := common.HexToAddress("0x1111111111111111111111111111111111111111")
	other := common.HexToAddress("0x2222222222222222222222222222222222222222")
	pool := common.HexToAddress("0x3333333333333333333333333333333333333333")
	otherPool := common.HexToAddress("0x4444444444444444444444444444444444444444")
	id := func(chainID uint64, deposit string, deadline uint64, c common.Address,
		amount *big.Int, lp common.Address, repay uint64) string {
		return signature.BorrowSessionID(chainID, deposit, deadline, c, amount, lp, repay)
	}
	base := id(1, "42", 1000, caller, big.NewInt(500), pool, 10)

	tests := []struct {
		name  string
		got   string
		equal bool
	}{
		{"identical inputs match", id(1, "42", 1000, caller, big.NewInt(500), pool, 10), true},
		{"different chain differs", id(2, "42", 1000, caller, big.NewInt(500), pool, 10), false},
		{"different deposit differs", id(1, "43", 1000, caller, big.NewInt(500), pool, 10), false},
		{"different deadline differs", id(1, "42", 1001, caller, big.NewInt(500), pool, 10), false},
		{"different caller differs", id(1, "42", 1000, other, big.NewInt(500), pool, 10), false},
		{"different borrow amount differs", id(1, "42", 1000, caller, big.NewInt(501), pool, 10), false},
		{"different liquidity pool differs", id(1, "42", 1000, caller, big.NewInt(500), otherPool, 10), false},
		{"different repayment chain differs", id(1, "42", 1000, caller, big.NewInt(500), pool, 11), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if (tt.got == base) != tt.equal {
				t.Fatalf("equal=%v, base=%q got=%q", tt.equal, base, tt.got)
			}
		})
	}

	want := "1-42-1000-" + caller.Hex() + "-500-" + pool.Hex() + "-10"
	if base != want {
		t.Fatalf("format: want %q got %q", want, base)
	}
}
