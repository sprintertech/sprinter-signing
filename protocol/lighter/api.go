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
	"github.com/rs/zerolog/log"
)

const (
	LIGHTER_URL             = "https://mainnet.zklighter.elliot.ai/api"
	TX_NOT_FOUND_RETRIES    = 3
	TX_NOT_FOUND_RETRY_WAIT = 500 * time.Millisecond
	TX_NOT_FOUND_ERROR_CODE = 21500
)

type TxType uint64

const (
	TxTypeL2Transfer TxType = 12
	TxTypeL2Withdraw TxType = 13
)

type Transfer struct {
	USDCAmount       uint64
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

func (e *LighterError) Error() error {
	return fmt.Errorf("lighter error: code %d, message: %s", e.Code, e.Message)
}

func (tx *LighterTx) UnmarshalJSON(data []byte) error {
	type t LighterTx
	if err := json.Unmarshal(data, (*t)(tx)); err != nil {
		return err
	}

	if tx.Type == TxTypeL2Transfer {
		var t *Transfer
		if err := json.Unmarshal([]byte(tx.Info), &t); err != nil {
			return err
		}
		tx.Transfer = t
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
	retryClient.CheckRetry = lighterCheckRetry
	retryClient.Logger = log.Logger

	return &LighterAPI{
		HTTPClient: retryClient.StandardClient(),
	}
}

// lighterCheckRetry checks if we should retry based on Lighter API error codes
func lighterCheckRetry(ctx context.Context, resp *http.Response, err error) (bool, error) {
	// Don't retry on context cancellation or client errors
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	if err != nil {
		return false, err
	}

	// Only retry on 200 OK with transaction not found error
	if resp.StatusCode == http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return false, err
		}
		resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(body))

		var lighterErr LighterError
		if err := json.Unmarshal(body, &lighterErr); err == nil {
			if lighterErr.Code == TX_NOT_FOUND_ERROR_CODE {
				return true, nil
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
		e := new(LighterError)
		if err := json.Unmarshal(body, e); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response body: %s, with error: %w", string(body), err)
		}
		return nil, e.Error()
	}

	return s, nil
}
