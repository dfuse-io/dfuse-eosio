// Code generated by protoc-gen-go. DO NOT EDIT.
// source: dfuse/eosio/funnel/v1/funnel.proto

package pbfunnel

import (
	context "context"
	fmt "fmt"
	v1 "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	proto "github.com/golang/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type StreamBlockRequest struct {
	FromBlockNum         int64    `protobuf:"varint,2,opt,name=fromBlockNum,proto3" json:"fromBlockNum,omitempty"`
	IrreversibleOnly     bool     `protobuf:"varint,3,opt,name=irreversibleOnly,proto3" json:"irreversibleOnly,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *StreamBlockRequest) Reset()         { *m = StreamBlockRequest{} }
func (m *StreamBlockRequest) String() string { return proto.CompactTextString(m) }
func (*StreamBlockRequest) ProtoMessage()    {}
func (*StreamBlockRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_477a415e0c40d59c, []int{0}
}

func (m *StreamBlockRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StreamBlockRequest.Unmarshal(m, b)
}
func (m *StreamBlockRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StreamBlockRequest.Marshal(b, m, deterministic)
}
func (m *StreamBlockRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StreamBlockRequest.Merge(m, src)
}
func (m *StreamBlockRequest) XXX_Size() int {
	return xxx_messageInfo_StreamBlockRequest.Size(m)
}
func (m *StreamBlockRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_StreamBlockRequest.DiscardUnknown(m)
}

var xxx_messageInfo_StreamBlockRequest proto.InternalMessageInfo

func (m *StreamBlockRequest) GetFromBlockNum() int64 {
	if m != nil {
		return m.FromBlockNum
	}
	return 0
}

func (m *StreamBlockRequest) GetIrreversibleOnly() bool {
	if m != nil {
		return m.IrreversibleOnly
	}
	return false
}

type StreamBlockResponse struct {
	Undo                 bool      `protobuf:"varint,1,opt,name=undo,proto3" json:"undo,omitempty"`
	Block                *v1.Block `protobuf:"bytes,2,opt,name=block,proto3" json:"block,omitempty"`
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *StreamBlockResponse) Reset()         { *m = StreamBlockResponse{} }
func (m *StreamBlockResponse) String() string { return proto.CompactTextString(m) }
func (*StreamBlockResponse) ProtoMessage()    {}
func (*StreamBlockResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_477a415e0c40d59c, []int{1}
}

func (m *StreamBlockResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StreamBlockResponse.Unmarshal(m, b)
}
func (m *StreamBlockResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StreamBlockResponse.Marshal(b, m, deterministic)
}
func (m *StreamBlockResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StreamBlockResponse.Merge(m, src)
}
func (m *StreamBlockResponse) XXX_Size() int {
	return xxx_messageInfo_StreamBlockResponse.Size(m)
}
func (m *StreamBlockResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_StreamBlockResponse.DiscardUnknown(m)
}

var xxx_messageInfo_StreamBlockResponse proto.InternalMessageInfo

func (m *StreamBlockResponse) GetUndo() bool {
	if m != nil {
		return m.Undo
	}
	return false
}

func (m *StreamBlockResponse) GetBlock() *v1.Block {
	if m != nil {
		return m.Block
	}
	return nil
}

func init() {
	proto.RegisterType((*StreamBlockRequest)(nil), "dfuse.eosio.funnel.v1.StreamBlockRequest")
	proto.RegisterType((*StreamBlockResponse)(nil), "dfuse.eosio.funnel.v1.StreamBlockResponse")
}

func init() {
	proto.RegisterFile("dfuse/eosio/funnel/v1/funnel.proto", fileDescriptor_477a415e0c40d59c)
}

var fileDescriptor_477a415e0c40d59c = []byte{
	// 268 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x51, 0xb1, 0x4e, 0xc3, 0x30,
	0x10, 0x55, 0x28, 0x54, 0xc8, 0x74, 0x40, 0x46, 0x48, 0x51, 0x59, 0xa2, 0x4c, 0xa5, 0x12, 0x36,
	0x29, 0x23, 0x13, 0x45, 0x62, 0x04, 0x29, 0x6c, 0x88, 0x05, 0x27, 0x97, 0x62, 0x91, 0xf8, 0x52,
	0x3b, 0x8e, 0xc4, 0xdf, 0xa3, 0x9e, 0x33, 0xa4, 0x6a, 0x07, 0xb6, 0xa7, 0xe7, 0xf7, 0xfc, 0xee,
	0xdd, 0xb1, 0xb4, 0xac, 0xbc, 0x03, 0x09, 0xe8, 0x34, 0xca, 0xca, 0x1b, 0x03, 0xb5, 0xec, 0xb3,
	0x01, 0x89, 0xd6, 0x62, 0x87, 0xfc, 0x9a, 0x34, 0x82, 0x34, 0x62, 0x78, 0xe9, 0xb3, 0x79, 0x32,
	0xb6, 0x16, 0x58, 0x42, 0xb1, 0x73, 0x12, 0x08, 0xc6, 0xb4, 0x64, 0xfc, 0xbd, 0xb3, 0xf0, 0xd5,
	0xac, 0x6b, 0x2c, 0x7e, 0x72, 0xd8, 0x7a, 0x70, 0x1d, 0x4f, 0xd9, 0xac, 0xb2, 0x18, 0xb8, 0x57,
	0xdf, 0xc4, 0x27, 0x49, 0xb4, 0x98, 0xe4, 0x7b, 0x1c, 0x5f, 0xb2, 0x4b, 0x6d, 0x2d, 0xf4, 0x60,
	0x9d, 0x56, 0x35, 0xbc, 0x99, 0xfa, 0x37, 0x9e, 0x24, 0xd1, 0xe2, 0x3c, 0x3f, 0xe0, 0xd3, 0x4f,
	0x76, 0xb5, 0x97, 0xe2, 0x5a, 0x34, 0x0e, 0x38, 0x67, 0xa7, 0xde, 0x94, 0x18, 0x47, 0x64, 0x23,
	0xcc, 0x33, 0x76, 0xa6, 0x76, 0x22, 0xca, 0xbc, 0x58, 0xdd, 0x88, 0x71, 0xb3, 0x30, 0x79, 0x9f,
	0x89, 0xf0, 0x4f, 0x50, 0xae, 0xb6, 0x6c, 0xfa, 0x42, 0x95, 0xf9, 0x86, 0xcd, 0x46, 0x39, 0x8e,
	0xdf, 0x8a, 0xa3, 0x7b, 0x11, 0x87, 0x95, 0xe7, 0xcb, 0xff, 0x48, 0xc3, 0xdc, 0xf7, 0xd1, 0xfa,
	0xf9, 0xe3, 0x69, 0xa3, 0xbb, 0x6f, 0xaf, 0x44, 0x81, 0x8d, 0x24, 0xe7, 0x9d, 0xc6, 0x01, 0x84,
	0x75, 0xb7, 0x4a, 0x1e, 0x3d, 0xdc, 0x63, 0xab, 0x02, 0x56, 0x53, 0x3a, 0xc1, 0xc3, 0x5f, 0x00,
	0x00, 0x00, 0xff, 0xff, 0xca, 0x3e, 0x0e, 0x01, 0xe1, 0x01, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// FunnelClient is the client API for Funnel service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type FunnelClient interface {
	StreamBlocks(ctx context.Context, in *StreamBlockRequest, opts ...grpc.CallOption) (Funnel_StreamBlocksClient, error)
}

type funnelClient struct {
	cc grpc.ClientConnInterface
}

func NewFunnelClient(cc grpc.ClientConnInterface) FunnelClient {
	return &funnelClient{cc}
}

func (c *funnelClient) StreamBlocks(ctx context.Context, in *StreamBlockRequest, opts ...grpc.CallOption) (Funnel_StreamBlocksClient, error) {
	stream, err := c.cc.NewStream(ctx, &_Funnel_serviceDesc.Streams[0], "/dfuse.eosio.funnel.v1.Funnel/StreamBlocks", opts...)
	if err != nil {
		return nil, err
	}
	x := &funnelStreamBlocksClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Funnel_StreamBlocksClient interface {
	Recv() (*StreamBlockResponse, error)
	grpc.ClientStream
}

type funnelStreamBlocksClient struct {
	grpc.ClientStream
}

func (x *funnelStreamBlocksClient) Recv() (*StreamBlockResponse, error) {
	m := new(StreamBlockResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// FunnelServer is the server API for Funnel service.
type FunnelServer interface {
	StreamBlocks(*StreamBlockRequest, Funnel_StreamBlocksServer) error
}

// UnimplementedFunnelServer can be embedded to have forward compatible implementations.
type UnimplementedFunnelServer struct {
}

func (*UnimplementedFunnelServer) StreamBlocks(req *StreamBlockRequest, srv Funnel_StreamBlocksServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamBlocks not implemented")
}

func RegisterFunnelServer(s *grpc.Server, srv FunnelServer) {
	s.RegisterService(&_Funnel_serviceDesc, srv)
}

func _Funnel_StreamBlocks_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(StreamBlockRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(FunnelServer).StreamBlocks(m, &funnelStreamBlocksServer{stream})
}

type Funnel_StreamBlocksServer interface {
	Send(*StreamBlockResponse) error
	grpc.ServerStream
}

type funnelStreamBlocksServer struct {
	grpc.ServerStream
}

func (x *funnelStreamBlocksServer) Send(m *StreamBlockResponse) error {
	return x.ServerStream.SendMsg(m)
}

var _Funnel_serviceDesc = grpc.ServiceDesc{
	ServiceName: "dfuse.eosio.funnel.v1.Funnel",
	HandlerType: (*FunnelServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StreamBlocks",
			Handler:       _Funnel_StreamBlocks_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "dfuse/eosio/funnel/v1/funnel.proto",
}
