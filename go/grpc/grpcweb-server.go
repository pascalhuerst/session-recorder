package grpc

import (
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

// StartGrpcWebServer exposes an existing *grpc.Server as gRPC-Web on the given
// HTTP port. The HTTP listener runs in its own goroutine. The returned
// *http.Server can be Shutdown by the caller for a graceful exit.
func StartGrpcWebServer(grpcServer *grpc.Server, port uint16) (*http.Server, error) {
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

	// Bind IPv4 only. http.Server.ListenAndServe would listen on "tcp" which
	// resolves ":port" to the IPv6 wildcard (":::port", dual-stack); the rest of
	// the system (StartProtocolServer, the recorder, the UI) is IPv4-only, so
	// listen on tcp4 explicitly.
	listener, err := net.Listen("tcp4", addr)
	if err != nil {
		return nil, fmt.Errorf("cannot listen on %s: %w", addr, err)
	}

	go func() {
		log.Info().Str("addr", listener.Addr().String()).Msg("gRPC-Web server listening")
		if err := httpServer.Serve(listener); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			log.Err(err).Msg("gRPC-Web server failed")
		}
	}()

	return httpServer, nil
}
