package lifi

import (
	"fmt"
	"math/big"
	"strings"
	"time"
)

type BigInt struct {
	*big.Int
}

func (b *BigInt) UnmarshalJSON(data []byte) error {
	if b.Int == nil {
		b.Int = new(big.Int)
	}

	s := strings.Trim(string(data), "\"")
	_, ok := b.SetString(s, 10)
	if !ok {
		return fmt.Errorf("failed to parse big.Int from %s", s)
	}

	return nil
}

func (b *BigInt) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%s", b.String())), nil
}

type OrderItem struct {
	User          string          `json:"user"`
	Nonce         *BigInt         `json:"nonce"`
	Inputs        [][2]*BigInt    `json:"inputs"` // [token, amount]
	Expires       int64           `json:"expires"`
	Outputs       []MandateOutput `json:"outputs"`
	LocalOracle   string          `json:"localOracle"`
	FillDeadline  uint32          `json:"fillDeadline"`
	OriginChainID string          `json:"originChainId"`
}

type MandateOutput struct {
	Call      string  `json:"call"`
	Token     string  `json:"token"`
	Amount    *BigInt `json:"amount"`
	Oracle    string  `json:"oracle"`
	ChainID   string  `json:"chainId"`
	Context   string  `json:"context"`
	Settler   string  `json:"settler"`
	Recipient string  `json:"recipient"`
}

type Quote struct {
	ToAsset      string  `json:"toAsset"`
	ToPrice      *BigInt `json:"toPrice"`
	Discount     string  `json:"discount"`
	FromAsset    string  `json:"fromAsset"`
	FromPrice    *BigInt `json:"fromPrice"`
	Intermediary string  `json:"intermediary"`
}

type Meta struct {
	SubmitTime                    int64     `json:"submitTime"`
	OrderStatus                   string    `json:"orderStatus"`
	DestinationAddress            string    `json:"destinationAddress"`
	OrderIdentifier               string    `json:"orderIdentifier"`
	OnChainOrderID                string    `json:"onChainOrderId"`
	SignedAt                      time.Time `json:"signedAt"`
	ExpiredAt                     time.Time `json:"expiredAt"`
	LastCompactDepositBlockNumber *BigInt   `json:"lastCompactDepositBlockNumber"`
}

type LifiOrder struct {
	Order              OrderItem `json:"order"`
	Quote              Quote     `json:"quote"`
	SponsorSignature   string    `json:"sponsorSignature"`
	AllocatorSignature string    `json:"allocatorSignature"`
	InputSettler       string    `json:"inputSettler"`
	Meta               Meta      `json:"meta"`
}
