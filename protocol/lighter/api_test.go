//nolint:gocognit
package lighter_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/sprintertech/sprinter-signing/protocol/lighter"
	"github.com/sprintertech/sprinter-signing/protocol/lighter/mock"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func Test_LighterAPI_GetTx(t *testing.T) {
	tests := []struct {
		name         string
		id           string
		mockResponse []byte
		statusCode   int
		mockError    error
		wantResult   []byte
		wantErr      bool
	}{
		{
			name:         "successful response",
			id:           "testhash",
			mockResponse: []byte(mock.LighterMockResponse),
			statusCode:   http.StatusOK,
			wantResult:   []byte(mock.ExpectedLighterResponse),
		},
		{
			name:      "HTTP error",
			id:        "errorhash",
			mockError: errors.New("connection refused"),
			wantErr:   true,
		},
		{
			name:         "non-200 status",
			id:           "badstatus",
			mockResponse: []byte("Not found"),
			statusCode:   http.StatusNotFound,
			wantErr:      true,
		},
		{
			name:         "invalid JSON",
			id:           "badjson",
			mockResponse: []byte("{invalid"),
			statusCode:   http.StatusOK,
			wantErr:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := lighter.NewLighterAPI()
			client.HTTPClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				expectedURL := fmt.Sprintf("%s/v1/tx?by=hash&value=%s", lighter.LIGHTER_URL, tc.id)
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

			got, err := client.GetTx(tc.id)

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

				var want *lighter.LighterTx
				err = json.Unmarshal(tc.wantResult, &want)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("expected %+v, got %+v", tc.wantResult, got)
				}

				if !reflect.DeepEqual(got, want) {
					t.Errorf("expected %+v, got %+v", tc.wantResult, got)
				}
			} else if got != nil {
				t.Errorf("expected nil result, got %+v", got)
			}
		})
	}
}

func Test_LighterCheckRetry(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		inputError   error
		ctxCanceled  bool
		wantRetry    bool
		wantErr      bool
	}{
		{
			name:         "retry on TX_NOT_FOUND_ERROR_CODE",
			statusCode:   http.StatusOK,
			responseBody: `{"code": 21500, "message": "transaction not found"}`,
			wantRetry:    true,
			wantErr:      false,
		},
		{
			name:         "retry on code 200 with missing info",
			statusCode:   http.StatusOK,
			responseBody: `{"code": 200, "hash": "0xabc", "type": 12}`,
			wantRetry:    true,
			wantErr:      false,
		},
		{
			name:         "retry on code 200 with empty info",
			statusCode:   http.StatusOK,
			responseBody: `{"code": 200, "hash": "0xabc", "type": 12, "info": ""}`,
			wantRetry:    true,
			wantErr:      false,
		},
		{
			name:         "no retry on code 200 with valid info",
			statusCode:   http.StatusOK,
			responseBody: `{"code": 200, "hash": "0xabc", "type": 12, "info": "{\"USDCAmount\": 1000}"}`,
			wantRetry:    false,
			wantErr:      false,
		},
		{
			name:         "no retry on different error code",
			statusCode:   http.StatusOK,
			responseBody: `{"code": 500, "message": "internal server error"}`,
			wantRetry:    false,
			wantErr:      false,
		},
		{
			name:         "no retry on non-200 status",
			statusCode:   http.StatusNotFound,
			responseBody: `{"error": "not found"}`,
			wantRetry:    false,
			wantErr:      false,
		},
		{
			name:       "no retry on input error",
			inputError: errors.New("connection error"),
			wantRetry:  false,
			wantErr:    true,
		},
		{
			name:        "error on context cancellation",
			ctxCanceled: true,
			wantRetry:   false,
			wantErr:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			if tc.ctxCanceled {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			var resp *http.Response
			if tc.statusCode != 0 {
				resp = &http.Response{
					StatusCode: tc.statusCode,
					Body:       io.NopCloser(bytes.NewReader([]byte(tc.responseBody))),
					Header:     make(http.Header),
				}
			}

			shouldRetry, err := lighter.LighterCheckRetry(ctx, resp, tc.inputError)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			if shouldRetry != tc.wantRetry {
				t.Errorf("expected retry=%v, got retry=%v", tc.wantRetry, shouldRetry)
			}
		})
	}
}
