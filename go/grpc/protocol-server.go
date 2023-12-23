package grpc

import (
	"fmt"
	"net"

	"github.com/rs/zerolog/log"

	"github.com/pascalhuerst/session-recorder/mdns"
	"google.golang.org/grpc"
)

type ProtocolServerContext struct {
}

func MakeProtocolServerContext() *ProtocolServerContext {
	return &ProtocolServerContext{}
}

type AudioSourceServer struct {
	protocolServerCtx *ProtocolServerContext
}

// ProtocolServer is an interface that all the grpc servers must implement
type ProtocolServer interface {
	registerGrpcServer(*grpc.Server)
	serverOptions() []grpc.ServerOption
}

// StartProtocolServer asynchronously starts listening for grpc connections from clients
func StartProtocolServer(server ProtocolServer, mdnsServer *mdns.Server, mdnsName string, port uint16, announcement [][]byte) (uint16, error) {

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return 0, fmt.Errorf("cannot listen on port: %v", err)
	}

	port = uint16(listener.Addr().(*net.TCPAddr).Port)

	const MAX_MSG_SIZE = 1024 * 1024
	grpcServer := grpc.NewServer(grpc.MaxRecvMsgSize(MAX_MSG_SIZE), grpc.MaxSendMsgSize(MAX_MSG_SIZE))

	go func() {
		err := grpcServer.Serve(listener)
		if err != nil {
			log.Err(err).Msgf("unable to serve %s on port %d", mdnsName, port)
		}
	}()

	if mdnsServer != nil {
		_, err = mdnsServer.PublishRecord(mdnsServer.Hostname(), mdnsName, "", port, announcement)
		if err != nil {
			return 0, fmt.Errorf("unable to publish mDNS record %s: %v", mdnsName, err)
		}
	}

	log.Info().Msgf("Protocol %s is now being served on port %d", mdnsName, port)

	return port, nil
}
