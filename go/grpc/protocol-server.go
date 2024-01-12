package grpc

import (
	"fmt"
	"net"

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

func StartProtocolServer(server ProtocolServer, mdnsServer *mdns.Server, mdnsName string, port uint16) (uint16, error) {
	host := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp4", host)
	if err != nil {
		return 0, fmt.Errorf("cannot listen on %s: %v", host, err)
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

	if mdnsServer != nil {
		_, err = mdnsServer.PublishRecord(mdnsServer.Hostname(), mdnsName, "", port, server.announcement())
		if err != nil {
			return 0, fmt.Errorf("unable to publish mDNS record %s: %v", mdnsName, err)
		}
	}

	log.Info().Msgf("Protocol %s is now being served on port %d", mdnsName, port)

	return port, nil
}
