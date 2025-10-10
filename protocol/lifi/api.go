package lifi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sprintertech/lifi-solver/pkg/protocols/lifi"
)

const (
	LIFI_URL = "https://order.li.fi"
)

type Response struct {
	Data []lifi.LifiOrder `json:"data"`
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
func (a *LifiAPI) GetOrder(orderID string) (*lifi.LifiOrder, error) {
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
