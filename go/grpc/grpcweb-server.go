package grpc

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

// StartGrpcWebServer exposes an existing *grpc.Server as gRPC-Web on the given
// HTTP port. The HTTP listener runs in its own goroutine.
func StartGrpcWebServer(grpcServer *grpc.Server, port uint16) error {
	wrapped := grpcweb.WrapServer(
		grpcServer,
		grpcweb.WithOriginFunc(func(string) bool { return true }),
		grpcweb.WithCorsForRegisteredEndpointsOnly(false),
	)

	addr := fmt.Sprintf(":%d", port)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if wrapped.IsGrpcWebRequest(r) ||
			wrapped.IsGrpcWebSocketRequest(r) ||
			wrapped.IsAcceptableGrpcCorsRequest(r) {
			wrapped.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	})

	httpServer := &http.Server{Addr: addr, Handler: handler}

	go func() {
		log.Info().Str("addr", addr).Msg("gRPC-Web server listening")
		if err := httpServer.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			log.Err(err).Msg("gRPC-Web server failed")
		}
	}()

	return nil
}
