package mayan

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const MAYAN_EXPLORER_URL = "https://explorer-api.mayan.finance/v3/swap/trx"

type MayanSwap struct {
	OrderHash        string
	RandomKey        string
	MayanBps         uint8
	AuctionMode      uint8
	RedeemRelayerFee string
	RefundRelayerFee string
	Trader           string
}

type MayanSwapClient struct {
	HTTPClient *http.Client
}

func NewMayanClient() *MayanSwapClient {
	return &MayanSwapClient{
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *MayanSwapClient) GetSwap(hash string) (*MayanSwap, error) {
	fullURL := fmt.Sprintf("%s/%s", MAYAN_EXPLORER_URL, hash)

	resp, err := c.HTTPClient.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var s *MayanSwap
	if err := json.Unmarshal(body, &s); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return s, nil
}
