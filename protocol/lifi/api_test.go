//nolint:gocognit
package lifi_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"testing"

	lifiProtocol "github.com/sprintertech/lifi-solver/pkg/protocols/lifi"
	"github.com/sprintertech/sprinter-signing/protocol/lifi"
	"github.com/sprintertech/sprinter-signing/protocol/lifi/mock"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func Test_LifiAPI_GetOrder(t *testing.T) {
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
			mockResponse: []byte(mock.LifiMockResponse),
			statusCode:   http.StatusOK,
			wantResult:   []byte(mock.ExpectedLifiResponse),
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
			client := lifi.NewLifiAPI()
			client.HTTPClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				expectedURL := fmt.Sprintf("%s/orders/status?onChainOrderId=%s", lifi.LIFI_URL, tc.id)
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

			got, err := client.GetOrder(tc.id)

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

				var want *lifiProtocol.LifiOrder
				err = json.Unmarshal(tc.wantResult, &want)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
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
