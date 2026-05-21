package grpc

import (
	"fmt"
	"net"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/pascalhuerst/session-recorder/mdns"
	"google.golang.org/grpc"
)

// ShutdownServer attempts GracefulStop with a deadline. If the graceful
// stop doesn't complete in time — typically because a long-lived
// server-streaming handler is still blocked on its per-RPC context — it
// falls back to Stop() to force-cancel those handlers so the process can
// exit. GracefulStop on its own will not cancel handler contexts.
func ShutdownServer(srv *grpc.Server, name string, timeout time.Duration) {
	done := make(chan struct{})
	go func() {
		srv.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
		log.Info().Str("server", name).Msg("gRPC graceful stop complete")
	case <-time.After(timeout):
		log.Warn().Str("server", name).Dur("timeout", timeout).Msg("gRPC graceful stop timed out, forcing")
		srv.Stop()
		<-done
	}
}

// ProtocolServer is an interface that all the grpc servers must implement
type ProtocolServer interface {
	registerGrpcServer(*grpc.Server)
	serverOptions() []grpc.ServerOption
	announcement() [][]byte
}

// StartProtocolServer creates a gRPC server, starts serving it in a goroutine,
// and publishes an mDNS record for it. Returns the gRPC server, the published
// mDNS record (nil if mdnsServer was nil), and the resolved port. The caller
// owns the lifecycle: close the PublishedService and GracefulStop the gRPC
// server on shutdown.
func StartProtocolServer(server ProtocolServer, mdnsServer *mdns.Server, mdnsName string, port uint16) (*grpc.Server, *mdns.PublishedService, uint16, error) {
	host := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp4", host)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("cannot listen on %s: %v", host, err)
	}

	port = uint16(listener.Addr().(*net.TCPAddr).Port)

	grpcServer := grpc.NewServer(server.serverOptions()...)

	server.registerGrpcServer(grpcServer)

	go func() {
		err := grpcServer.Serve(listener)
		if err != nil {
			log.Err(err).Msgf("unable to serve %s on port %d", mdnsName, port)
		}
	}()

	var published *mdns.PublishedService
	if mdnsServer != nil {
		published, err = mdnsServer.PublishRecord(mdnsServer.Hostname(), mdnsName, "", port, server.announcement())
		if err != nil {
			return nil, nil, 0, fmt.Errorf("unable to publish mDNS record %s: %v", mdnsName, err)
		}
	}

	log.Info().Msgf("Protocol %s is now being served on port %d", mdnsName, port)

	return grpcServer, published, port, nil
}
