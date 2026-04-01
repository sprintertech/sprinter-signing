// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package chains

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/suite"
)

type UtilTestSuite struct {
	suite.Suite
}

func TestRunNewEVMConfigTestSuite(t *testing.T) {
	suite.Run(t, new(UtilTestSuite))
}

func (s *UtilTestSuite) Test_CalculateStartingBlock_ProperAdjustment() {
	res, err := CalculateStartingBlock(big.NewInt(104), big.NewInt(5))
	s.Equal(big.NewInt(100), res)
	s.Nil(err)
}

func (s *UtilTestSuite) Test_CalculateStartingBlock_NoAdjustment() {
	res, err := CalculateStartingBlock(big.NewInt(200), big.NewInt(5))
	s.Equal(big.NewInt(200), res)
	s.Nil(err)
}

func (s *UtilTestSuite) Test_CalculateStartingBlock_Nil() {
	res, err := CalculateStartingBlock(nil, nil)
	s.Nil(res)
	s.NotNil(err)
}

func (s *UtilTestSuite) TestScaleTokenAmount() {
	tests := []struct {
		name        string
		amount      *big.Int
		srcDecimals int64
		dstDecimals int64
		want        *big.Int
	}{
		{
			name:        "same decimals — no scaling",
			amount:      big.NewInt(1_000_000),
			srcDecimals: 6,
			dstDecimals: 6,
			want:        big.NewInt(1_000_000),
		},
		{
			name:        "18 to 6 decimals",
			amount:      big.NewInt(1_000_000_000_000_000_000),
			srcDecimals: 18,
			dstDecimals: 6,
			want:        big.NewInt(1_000_000),
		},
		{
			name:        "6 to 18 decimals",
			amount:      big.NewInt(1_000_000),
			srcDecimals: 6,
			dstDecimals: 18,
			want:        big.NewInt(1_000_000_000_000_000_000),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := ScaleTokenAmount(tt.amount, tt.srcDecimals, tt.dstDecimals)
			s.Equal(tt.want, got)
		})
	}
}
