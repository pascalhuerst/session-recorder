package grpc

import (
	"fmt"
	"net"
	"net/http"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/rs/zerolog/log"

	"github.com/pascalhuerst/session-recorder/mdns"
	"google.golang.org/grpc"
)

// ProtocolServer is an interface that all the grpc servers must implement
type ProtocolServer interface {
	registerGrpcServer(*grpc.Server)
	serverOptions() []grpc.ServerOption
	announcement() [][]byte
}

func StartProtocolServer(server ProtocolServer, mdnsServer *mdns.Server, mdnsName string, port uint16, webWrapper bool) (uint16, error) {
	host := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", host)
	if err != nil {
		return 0, fmt.Errorf("cannot listen on %s: %v", host, err)
	}

	port = uint16(listener.Addr().(*net.TCPAddr).Port)

	grpcServer := grpc.NewServer(server.serverOptions()...)

	server.registerGrpcServer(grpcServer)

	if webWrapper {
		wrappedGrpc := grpcweb.WrapServer(grpcServer,
			grpcweb.WithOriginFunc(func(s string) bool {
				return true
			}),
			grpcweb.WithWebsocketOriginFunc(func(req *http.Request) bool {
				return true
			}),
		)

		http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
			wrappedGrpc.ServeHTTP(resp, req)
		})

		go func() {
			err := http.Serve(listener, nil)
			if err != nil {
				log.Err(err).Uint16("port", port).Bool("web-wrapper", webWrapper).Msg("unable to start protocol server")
			}
		}()
	} else {
		go func() {
			err := grpcServer.Serve(listener)
			if err != nil {
				log.Err(err).Uint16("port", port).Bool("web-wrapper", webWrapper).Msg("unable to start protocol server")
			}
		}()
	}

	if mdnsServer != nil {
		_, err = mdnsServer.PublishRecord(mdnsServer.Hostname(), mdnsName, "", port, server.announcement())
		if err != nil {
			return 0, fmt.Errorf("unable to publish mDNS record %s: %v", mdnsName, err)
		}
	}

	log.Info().Msgf("Protocol %s is now being served on port %d", mdnsName, port)

	return port, nil
}
