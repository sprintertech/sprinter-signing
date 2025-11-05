package lighter

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	LIGHTER_URL = "https://mainnet.zklighter.elliot.ai/api"
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
	return &LighterAPI{
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
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
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return s, nil
}
