package rhinestone

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"
)

const (
	RHINESTONE_ORCHESTRATOR_URL = "https://orchestrator.rhinestone.dev"
)

type RhinestoneOrchestrator struct {
	apiKey string
	Client *http.Client
}

func NewRhinestoneOrchestrator(apiKey string) *RhinestoneOrchestrator {
	return &RhinestoneOrchestrator{
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
		apiKey: apiKey,
	}
}

func (o *RhinestoneOrchestrator) GetBundle(ctx context.Context, bundleID *big.Int) (*Bundle, error) {
	url := fmt.Sprintf("%s/bundles/%s", RHINESTONE_ORCHESTRATOR_URL, bundleID.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("x-api-key", o.apiKey)

	resp, err := o.Client.Do(req)
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

	b := new(Bundle)
	if err := json.Unmarshal(body, b); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return b, nil
}
