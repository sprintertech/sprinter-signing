package lifi

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"
)

const (
	LIFI_URL = "https://order-dev.li.fi"
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
	return []byte(b.String()), nil
}

type OrderItem struct {
	User          string       `json:"user"`
	Nonce         *BigInt      `json:"nonce"`
	Inputs        [][2]*BigInt `json:"inputs"` // [token, amount]
	Expires       int64        `json:"expires"`
	Outputs       []Output     `json:"outputs"`
	LocalOracle   string       `json:"localOracle"`
	FillDeadline  int64        `json:"fillDeadline"`
	OriginChainID string       `json:"originChainId"`
}

type Output struct {
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

type Response struct {
	Data []LifiOrder `json:"data"`
}

type LifiAPI struct {
	HTTPClient *http.Client
}

func NewLifiAPI() *LifiAPI {
	return &LifiAPI{
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetOrder fetches order from the LiFi API by its on-chain orderID
func (a *LifiAPI) GetOrder(orderID string) (*LifiOrder, error) {
	url := fmt.Sprintf("%s/orders?onChainOrderId=%s", LIFI_URL, orderID)
	resp, err := a.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	s := new(Response)
	if err := json.Unmarshal(body, s); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	if len(s.Data) == 0 {
		return nil, fmt.Errorf("no order found")
	}

	return &s.Data[0], nil
}
