// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v3.21.12
// source: pb/upgradeable-consensus.proto

package pb

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

const (
	P2P_Send_FullMethodName          = "/pb.P2P/Send"
	P2P_Request_FullMethodName       = "/pb.P2P/Request"
	P2P_PoTresRequest_FullMethodName = "/pb.P2P/PoTresRequest"
)

// P2PClient is the client API for P2P service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type P2PClient interface {
	Send(ctx context.Context, in *Packet, opts ...grpc.CallOption) (*Empty, error)
	Request(ctx context.Context, in *HeaderRequest, opts ...grpc.CallOption) (*Header, error)
	PoTresRequest(ctx context.Context, in *PoTRequest, opts ...grpc.CallOption) (*PoTResponse, error)
}

type p2PClient struct {
	cc grpc.ClientConnInterface
}

func NewP2PClient(cc grpc.ClientConnInterface) P2PClient {
	return &p2PClient{cc}
}

func (c *p2PClient) Send(ctx context.Context, in *Packet, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, P2P_Send_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *p2PClient) Request(ctx context.Context, in *HeaderRequest, opts ...grpc.CallOption) (*Header, error) {
	out := new(Header)
	err := c.cc.Invoke(ctx, P2P_Request_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *p2PClient) PoTresRequest(ctx context.Context, in *PoTRequest, opts ...grpc.CallOption) (*PoTResponse, error) {
	out := new(PoTResponse)
	err := c.cc.Invoke(ctx, P2P_PoTresRequest_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// P2PServer is the server API for P2P service.
// All implementations must embed UnimplementedP2PServer
// for forward compatibility
type P2PServer interface {
	Send(context.Context, *Packet) (*Empty, error)
	Request(context.Context, *HeaderRequest) (*Header, error)
	PoTresRequest(context.Context, *PoTRequest) (*PoTResponse, error)
	mustEmbedUnimplementedP2PServer()
}

// UnimplementedP2PServer must be embedded to have forward compatible implementations.
type UnimplementedP2PServer struct {
}

func (UnimplementedP2PServer) Send(context.Context, *Packet) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Send not implemented")
}
func (UnimplementedP2PServer) Request(context.Context, *HeaderRequest) (*Header, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Request not implemented")
}
func (UnimplementedP2PServer) PoTresRequest(context.Context, *PoTRequest) (*PoTResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PoTresRequest not implemented")
}
func (UnimplementedP2PServer) mustEmbedUnimplementedP2PServer() {}

// UnsafeP2PServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to P2PServer will
// result in compilation errors.
type UnsafeP2PServer interface {
	mustEmbedUnimplementedP2PServer()
}

func RegisterP2PServer(s grpc.ServiceRegistrar, srv P2PServer) {
	s.RegisterService(&P2P_ServiceDesc, srv)
}

func _P2P_Send_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Packet)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(P2PServer).Send(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: P2P_Send_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(P2PServer).Send(ctx, req.(*Packet))
	}
	return interceptor(ctx, in, info, handler)
}

func _P2P_Request_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(HeaderRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(P2PServer).Request(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: P2P_Request_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(P2PServer).Request(ctx, req.(*HeaderRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _P2P_PoTresRequest_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PoTRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(P2PServer).PoTresRequest(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: P2P_PoTresRequest_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(P2PServer).PoTresRequest(ctx, req.(*PoTRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// P2P_ServiceDesc is the grpc.ServiceDesc for P2P service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var P2P_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "pb.P2P",
	HandlerType: (*P2PServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Send",
			Handler:    _P2P_Send_Handler,
		},
		{
			MethodName: "Request",
			Handler:    _P2P_Request_Handler,
		},
		{
			MethodName: "PoTresRequest",
			Handler:    _P2P_PoTresRequest_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "pb/upgradeable-consensus.proto",
}
