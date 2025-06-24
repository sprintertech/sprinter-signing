package rhinestone_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"testing"

	"github.com/sprintertech/sprinter-signing/protocol/rhinestone"
)

// roundTripperFunc allows mocking HTTP transport
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func Test_RhinestoneOrchestrator_GetBundle(t *testing.T) {
	mockResponse, _ := os.ReadFile("./mock/response.json")
	bundleID, _ := new(big.Int).SetString("1566632349852817507569732921128369748446911234273639335443882769857984331776", 10)

	tests := []struct {
		name         string
		bundleID     *big.Int
		mockResponse []byte
		statusCode   int
		mockError    error
		wantResult   *rhinestone.Bundle
		wantErr      bool
	}{
		{
			name:         "successful response",
			bundleID:     bundleID,
			mockResponse: mockResponse,
			statusCode:   http.StatusOK,
			wantResult: &rhinestone.Bundle{
				Status:        rhinestone.StatusCompleted,
				TargetChainId: uint64(8453),
				UserAddress:   "0x6Dc7354cEA1b225B299Fe06b97aC12ac5066B899",
				BundleData: rhinestone.BundleData{
					Nonce:   "1566632349852817507569732921128369748446911234273639335443882769857984331776",
					Expires: "1782292973",
				},
				BundleEvent: rhinestone.BundleEvent{
					BundleId: bundleID.String(),
					AcrossDepositEvents: []rhinestone.AcrossDepositEvent{
						{
							Message:             "0x",
							DepositId:           "72508795011696122011199505727503554569122957656552787329084847407262805528117",
							Depositor:           "0x6Dc7354cEA1b225B299Fe06b97aC12ac5066B899",
							Recipient:           "0x6Dc7354cEA1b225B299Fe06b97aC12ac5066B899",
							InputToken:          "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
							InputAmount:         "10000000",
							OutputToken:         "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
							FillDeadline:        "1750757273",
							OutputAmount:        "10000000",
							QuoteTimestamp:      1750756973,
							ExclusiveRelayer:    "0x000000000060f6e853447881951574CDd0663530",
							DestinationChainId:  uint64(8453),
							ExclusivityDeadline: "1750757273",
						},
					},
				},
			},
		},
		{
			name:      "HTTP error",
			bundleID:  bundleID,
			mockError: errors.New("connection refused"),
			wantErr:   true,
		},
		{
			name:         "non-200 status",
			bundleID:     bundleID,
			mockResponse: []byte("Not found"),
			statusCode:   http.StatusNotFound,
			wantErr:      true,
		},
		{
			name:         "invalid JSON",
			bundleID:     bundleID,
			mockResponse: []byte("{invalid"),
			statusCode:   http.StatusOK,
			wantErr:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			orchestrator := rhinestone.NewRhinestoneOrchestrator("api-key")
			orchestrator.Client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				// Verify URL construction
				expectedURL := fmt.Sprintf("%s/bundles/%s", rhinestone.RHINESTONE_ORCHESTRATOR_URL, tc.bundleID)
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

			got, err := orchestrator.GetBundle(context.Background(), tc.bundleID)

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
				if got.BundleData != tc.wantResult.BundleData {
					t.Errorf("expected %+v, got %+v", tc.wantResult, got)
				}
				if got.Status != tc.wantResult.Status {
					t.Errorf("expected %+v, got %+v", tc.wantResult, got)
				}
				if got.BundleEvent.AcrossDepositEvents[0] != tc.wantResult.BundleEvent.AcrossDepositEvents[0] {
					t.Errorf("expected %+v, got %+v", tc.wantResult, got)
				}
			} else if got != nil {
				t.Errorf("expected nil result, got %+v", got)
			}
		})
	}
}
