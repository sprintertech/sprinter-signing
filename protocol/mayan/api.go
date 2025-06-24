package mayan

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	MAYAN_EXPLORER_URL = "https://explorer-api.mayan.finance"
	WORMHOLE_DECIMALS  = 8
)

type MayanSwap struct {
	OrderHash          string
	RandomKey          string
	MayanBps           uint8
	AuctionMode        uint8
	RedeemRelayerFee   string
	RedeemRelayerFee64 uint64
	RefundRelayerFee   string
	RefundRelayerFee64 uint64
	Trader             string
	MinAmountOut64     string
	SourceTxHash       string
}

type MayanExplorer struct {
	HTTPClient *http.Client
}

func NewMayanExplorer() *MayanExplorer {
	return &MayanExplorer{
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *MayanExplorer) GetSwap(hash string) (*MayanSwap, error) {
	url := fmt.Sprintf("%s/v3/swap/order-id/SWIFT_%s", MAYAN_EXPLORER_URL, hash)
	resp, err := c.HTTPClient.Get(url)
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

	s := new(MayanSwap)
	if err := json.Unmarshal(body, s); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return s, nil
}
