package handlers

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
)

type BigInt struct {
	*big.Int
}

func (b *BigInt) UnmarshalJSON(data []byte) error {
	if b.Int == nil {
		b.Int = new(big.Int)
	}

	s := strings.Trim(string(data), "\"")
	_, ok := b.Int.SetString(s, 10)
	if !ok {
		return fmt.Errorf("failed to parse big.Int from %s", s)
	}

	return nil
}

func (b *BigInt) MarshalJSON() ([]byte, error) {
	return []byte(b.String()), nil
}

func JSONError(w http.ResponseWriter, err interface{}, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(err)
}
