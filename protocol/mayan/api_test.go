package mayan_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/sprintertech/sprinter-signing/protocol/mayan"
)

// roundTripperFunc allows mocking HTTP transport
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func Test_MayanExplorer_GetSwap(t *testing.T) {
	tests := []struct {
		name         string
		hash         string
		mockResponse []byte
		statusCode   int
		mockError    error
		wantResult   *mayan.MayanSwap
		wantErr      bool
	}{
		{
			name: "successful response",
			hash: "testhash",
			mockResponse: []byte(`{
                "orderHash": "0x123",
                "randomKey": "key",
                "mayanBps": 10,
                "auctionMode": 1,
                "redeemRelayerFee": "0.1",
                "refundRelayerFee": "0.05",
                "trader": "0xTrader",
                "minAmountOut64": "100"
            }`),
			statusCode: http.StatusOK,
			wantResult: &mayan.MayanSwap{
				OrderHash:        "0x123",
				RandomKey:        "key",
				MayanBps:         10,
				AuctionMode:      1,
				RedeemRelayerFee: "0.1",
				RefundRelayerFee: "0.05",
				Trader:           "0xTrader",
				MinAmountOut64:   "100",
			},
		},
		{
			name:      "HTTP error",
			hash:      "errorhash",
			mockError: errors.New("connection refused"),
			wantErr:   true,
		},
		{
			name:         "non-200 status",
			hash:         "badstatus",
			mockResponse: []byte("Not found"),
			statusCode:   http.StatusNotFound,
			wantErr:      true,
		},
		{
			name:         "invalid JSON",
			hash:         "badjson",
			mockResponse: []byte("{invalid"),
			statusCode:   http.StatusOK,
			wantErr:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := mayan.NewMayanExplorer()
			client.HTTPClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				// Verify URL construction
				expectedURL := fmt.Sprintf("%s/v3/swap/trx/%s", mayan.MAYAN_EXPLORER_URL, tc.hash)
				if req.URL.String() != expectedURL {
					return nil, fmt.Errorf("unexpected URL: got %s, want %s", req.URL.String(), expectedURL)
				}

				if tc.mockError != nil {
					return nil, tc.mockError
				}

				return &http.Response{
					StatusCode: tc.statusCode,
					Body:       io.NopCloser(bytes.NewReader(tc.mockResponse)),
					Header:     make(http.Header),
				}, nil
			})

			got, err := client.GetSwap(tc.hash)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error got %v", err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.wantResult != nil {
				if got == nil {
					t.Fatal("expected non-nil result, got nil")
				}
				if *got != *tc.wantResult {
					t.Errorf("expected %+v, got %+v", tc.wantResult, got)
				}
			} else if got != nil {
				t.Errorf("expected nil result, got %+v", got)
			}
		})
	}
}
