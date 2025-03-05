// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package health

import (
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// StartHealthEndpoint starts /health endpoint on provided port that returns ok on invocation
func StartHealthEndpoint(port uint16) {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}

	err := srv.ListenAndServe()
	if err != nil {
		log.Err(err).Msgf("Failed starting health server")
		return
	}

	log.Info().Msgf("started /health endpoint on port %d", port)
}
