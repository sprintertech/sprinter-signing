package lighter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

const (
	LIGHTER_URL             = "https://mainnet.zklighter.elliot.ai/api"
	TX_NOT_FOUND_RETRIES    = 3
	TX_NOT_FOUND_RETRY_WAIT = 500 * time.Millisecond
	TX_NOT_FOUND_ERROR_CODE = 21500
	TX_FOUND_STATUS_CODE    = 200
)

type TxType uint64

const (
	TxTypeL2Transfer TxType = 12
	TxTypeL2Withdraw TxType = 13
)

type Transfer struct {
	Amount           uint64
	AssetIndex       uint64
	FromAccountIndex uint64
	ToAccountIndex   int
	Fee              uint64
	Memo             []byte
}

type LighterTx struct {
	Code      uint64 `json:"code"`
	Hash      string `json:"hash"`
	Type      TxType `json:"type"`
	Info      string `json:"info"`
	L1Address string `json:"l1_address"`
	Transfer  *Transfer
}

type LighterError struct {
	Code    uint64 `json:"code"`
	Message string `json:"message"`
}

func (e *LighterError) Error() string {
	return fmt.Sprintf("lighter error: code %d, message: %s", e.Code, e.Message)
}

func (tx *LighterTx) UnmarshalJSON(data []byte) error {
	type t LighterTx
	if err := json.Unmarshal(data, (*t)(tx)); err != nil {
		return err
	}

	if tx.Type == TxTypeL2Transfer {
		var t *Transfer
		if err := json.Unmarshal([]byte(tx.Info), &t); err != nil {
			return fmt.Errorf("failed to unmarshal info: %w", err)
		}
		tx.Transfer = t
	} else {
		return fmt.Errorf("unsupported transaction type: %d", tx.Type)
	}

	return nil
}

type LighterAPI struct {
	HTTPClient *http.Client
}

func NewLighterAPI() *LighterAPI {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = TX_NOT_FOUND_RETRIES - 1 // RetryMax is a number of retries after an initial attempt
	retryClient.RetryWaitMin = TX_NOT_FOUND_RETRY_WAIT
	retryClient.RetryWaitMax = TX_NOT_FOUND_RETRY_WAIT
	retryClient.CheckRetry = LighterCheckRetry
	retryClient.Logger = nil

	return &LighterAPI{
		HTTPClient: retryClient.StandardClient(),
	}
}

// LighterCheckRetry checks if we should retry the request.
// Retries when: error code is TX_NOT_FOUND_ERROR_CODE (21500) or response has code 200 but missing/empty info.
func LighterCheckRetry(ctx context.Context, resp *http.Response, err error) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	if err != nil {
		return false, err
	}

	if resp.StatusCode == http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return false, err
		}
		resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(body))

		// First try to unmarshal as LighterError
		e := new(LighterError)
		if err := json.Unmarshal(body, e); err == nil && e.Code == TX_NOT_FOUND_ERROR_CODE {
			return true, nil
		}

		// Check if it's a LighterTx with code 200 but missing info
		var raw map[string]interface{}
		if err := json.Unmarshal(body, &raw); err == nil {
			if code, ok := raw["code"].(float64); ok && code == TX_FOUND_STATUS_CODE {
				info, hasInfo := raw["info"].(string)
				if !hasInfo || info == "" {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// GetTx fetches transaction from the lighter API
func (a *LighterAPI) GetTx(hash string) (*LighterTx, error) {
	url := fmt.Sprintf("%s/v1/tx?by=hash&value=%s", LIGHTER_URL, hash)
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

	s := new(LighterTx)
	if err := json.Unmarshal(body, s); err != nil {
		return nil, err
	}

	return s, nil
}
