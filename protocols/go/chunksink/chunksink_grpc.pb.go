// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package chunksink

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// ChunkSinkClient is the client API for ChunkSink service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ChunkSinkClient interface {
	StreamChunkData(ctx context.Context, in *StreamChunkDataRequest, opts ...grpc.CallOption) (ChunkSink_StreamChunkDataClient, error)
	SetRecorderStatus(ctx context.Context, in *RecorderStatusRequest, opts ...grpc.CallOption) (*RecorderStatusReply, error)
}

type chunkSinkClient struct {
	cc grpc.ClientConnInterface
}

func NewChunkSinkClient(cc grpc.ClientConnInterface) ChunkSinkClient {
	return &chunkSinkClient{cc}
}

func (c *chunkSinkClient) StreamChunkData(ctx context.Context, in *StreamChunkDataRequest, opts ...grpc.CallOption) (ChunkSink_StreamChunkDataClient, error) {
	stream, err := c.cc.NewStream(ctx, &ChunkSink_ServiceDesc.Streams[0], "/chunksink.ChunkSink/StreamChunkData", opts...)
	if err != nil {
		return nil, err
	}
	x := &chunkSinkStreamChunkDataClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type ChunkSink_StreamChunkDataClient interface {
	Recv() (*ChunkData, error)
	grpc.ClientStream
}

type chunkSinkStreamChunkDataClient struct {
	grpc.ClientStream
}

func (x *chunkSinkStreamChunkDataClient) Recv() (*ChunkData, error) {
	m := new(ChunkData)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *chunkSinkClient) SetRecorderStatus(ctx context.Context, in *RecorderStatusRequest, opts ...grpc.CallOption) (*RecorderStatusReply, error) {
	out := new(RecorderStatusReply)
	err := c.cc.Invoke(ctx, "/chunksink.ChunkSink/SetRecorderStatus", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ChunkSinkServer is the server API for ChunkSink service.
// All implementations should embed UnimplementedChunkSinkServer
// for forward compatibility
type ChunkSinkServer interface {
	StreamChunkData(*StreamChunkDataRequest, ChunkSink_StreamChunkDataServer) error
	SetRecorderStatus(context.Context, *RecorderStatusRequest) (*RecorderStatusReply, error)
}

// UnimplementedChunkSinkServer should be embedded to have forward compatible implementations.
type UnimplementedChunkSinkServer struct {
}

func (UnimplementedChunkSinkServer) StreamChunkData(*StreamChunkDataRequest, ChunkSink_StreamChunkDataServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamChunkData not implemented")
}
func (UnimplementedChunkSinkServer) SetRecorderStatus(context.Context, *RecorderStatusRequest) (*RecorderStatusReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetRecorderStatus not implemented")
}

// UnsafeChunkSinkServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ChunkSinkServer will
// result in compilation errors.
type UnsafeChunkSinkServer interface {
	mustEmbedUnimplementedChunkSinkServer()
}

func RegisterChunkSinkServer(s grpc.ServiceRegistrar, srv ChunkSinkServer) {
	s.RegisterService(&ChunkSink_ServiceDesc, srv)
}

func _ChunkSink_StreamChunkData_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(StreamChunkDataRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ChunkSinkServer).StreamChunkData(m, &chunkSinkStreamChunkDataServer{stream})
}

type ChunkSink_StreamChunkDataServer interface {
	Send(*ChunkData) error
	grpc.ServerStream
}

type chunkSinkStreamChunkDataServer struct {
	grpc.ServerStream
}

func (x *chunkSinkStreamChunkDataServer) Send(m *ChunkData) error {
	return x.ServerStream.SendMsg(m)
}

func _ChunkSink_SetRecorderStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RecorderStatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ChunkSinkServer).SetRecorderStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/chunksink.ChunkSink/SetRecorderStatus",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ChunkSinkServer).SetRecorderStatus(ctx, req.(*RecorderStatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// ChunkSink_ServiceDesc is the grpc.ServiceDesc for ChunkSink service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ChunkSink_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "chunksink.ChunkSink",
	HandlerType: (*ChunkSinkServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SetRecorderStatus",
			Handler:    _ChunkSink_SetRecorderStatus_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StreamChunkData",
			Handler:       _ChunkSink_StreamChunkData_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "chunksink.proto",
}