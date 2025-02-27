package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"github.com/sprintertech/sprinter-signing/api/handlers"
)

func Serve(
	ctx context.Context,
	addr string,
	signingHandler *handlers.SigningHandler,
) {
	r := mux.NewRouter()
	r.HandleFunc("POST /v1/chains/{chainId:[0-9]+}/signatures", signingHandler.HandleSigning)
	http.Handle("/", r)

	server := &http.Server{
		Addr: addr,
	}
	go func() {
		log.Info().Msgf("Starting server on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := server.Shutdown(shutdownCtx)
	if err != nil {
		log.Err(err).Msgf("Error shutting down server")
	} else {
		log.Info().Msgf("Server shut down gracefully.")
	}
}
