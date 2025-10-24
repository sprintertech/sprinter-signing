//nolint:gocognit
package lighter_test

import (
	"bytes"
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
