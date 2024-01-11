// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v4.22.3
// source: pb/upgradeable-consensus.proto

package pb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type PacketType int32

const (
	PacketType_P2PPACKET    PacketType = 0
	PacketType_CLIENTPACKET PacketType = 1
)

// Enum value maps for PacketType.
var (
	PacketType_name = map[int32]string{
		0: "P2PPACKET",
		1: "CLIENTPACKET",
	}
	PacketType_value = map[string]int32{
		"P2PPACKET":    0,
		"CLIENTPACKET": 1,
	}
)

func (x PacketType) Enum() *PacketType {
	p := new(PacketType)
	*p = x
	return p
}

func (x PacketType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (PacketType) Descriptor() protoreflect.EnumDescriptor {
	return file_pb_upgradeable_consensus_proto_enumTypes[0].Descriptor()
}

func (PacketType) Type() protoreflect.EnumType {
	return &file_pb_upgradeable_consensus_proto_enumTypes[0]
}

func (x PacketType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use PacketType.Descriptor instead.
func (PacketType) EnumDescriptor() ([]byte, []int) {
	return file_pb_upgradeable_consensus_proto_rawDescGZIP(), []int{0}
}

type TransactionType int32

const (
	TransactionType_NORMAL   TransactionType = 0
	TransactionType_UPGRADE  TransactionType = 1
	TransactionType_TIMEVOTE TransactionType = 2
	TransactionType_LOCK     TransactionType = 3
)

// Enum value maps for TransactionType.
var (
	TransactionType_name = map[int32]string{
		0: "NORMAL",
		1: "UPGRADE",
		2: "TIMEVOTE",
		3: "LOCK",
	}
	TransactionType_value = map[string]int32{
		"NORMAL":   0,
		"UPGRADE":  1,
		"TIMEVOTE": 2,
		"LOCK":     3,
	}
)

func (x TransactionType) Enum() *TransactionType {
	p := new(TransactionType)
	*p = x
	return p
}

func (x TransactionType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (TransactionType) Descriptor() protoreflect.EnumDescriptor {
	return file_pb_upgradeable_consensus_proto_enumTypes[1].Descriptor()
}

func (TransactionType) Type() protoreflect.EnumType {
	return &file_pb_upgradeable_consensus_proto_enumTypes[1]
}

func (x TransactionType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use TransactionType.Descriptor instead.
func (TransactionType) EnumDescriptor() ([]byte, []int) {
	return file_pb_upgradeable_consensus_proto_rawDescGZIP(), []int{1}
}

type Packet struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Msg         []byte     `protobuf:"bytes,1,opt,name=msg,proto3" json:"msg,omitempty"`
	ConsensusID int64      `protobuf:"varint,2,opt,name=consensusID,proto3" json:"consensusID,omitempty"`
	Epoch       int64      `protobuf:"varint,3,opt,name=epoch,proto3" json:"epoch,omitempty"`
	ChainID     int32      `protobuf:"varint,4,opt,name=chainID,proto3" json:"chainID,omitempty"`
	Type        PacketType `protobuf:"varint,5,opt,name=type,proto3,enum=pb.PacketType" json:"type,omitempty"`
}

func (x *Packet) Reset() {
	*x = Packet{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pb_upgradeable_consensus_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Packet) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Packet) ProtoMessage() {}

func (x *Packet) ProtoReflect() protoreflect.Message {
	mi := &file_pb_upgradeable_consensus_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Packet.ProtoReflect.Descriptor instead.
func (*Packet) Descriptor() ([]byte, []int) {
	return file_pb_upgradeable_consensus_proto_rawDescGZIP(), []int{0}
}

func (x *Packet) GetMsg() []byte {
	if x != nil {
		return x.Msg
	}
	return nil
}

func (x *Packet) GetConsensusID() int64 {
	if x != nil {
		return x.ConsensusID
	}
	return 0
}

func (x *Packet) GetEpoch() int64 {
	if x != nil {
		return x.Epoch
	}
	return 0
}

func (x *Packet) GetChainID() int32 {
	if x != nil {
		return x.ChainID
	}
	return 0
}

func (x *Packet) GetType() PacketType {
	if x != nil {
		return x.Type
	}
	return PacketType_P2PPACKET
}

type Transaction struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Type    TransactionType `protobuf:"varint,1,opt,name=type,proto3,enum=pb.TransactionType" json:"type,omitempty"`
	Payload []byte          `protobuf:"bytes,2,opt,name=payload,proto3" json:"payload,omitempty"`
	ChainID int32           `protobuf:"varint,3,opt,name=chainID,proto3" json:"chainID,omitempty"`
}

func (x *Transaction) Reset() {
	*x = Transaction{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pb_upgradeable_consensus_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Transaction) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Transaction) ProtoMessage() {}

func (x *Transaction) ProtoReflect() protoreflect.Message {
	mi := &file_pb_upgradeable_consensus_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Transaction.ProtoReflect.Descriptor instead.
func (*Transaction) Descriptor() ([]byte, []int) {
	return file_pb_upgradeable_consensus_proto_rawDescGZIP(), []int{1}
}

func (x *Transaction) GetType() TransactionType {
	if x != nil {
		return x.Type
	}
	return TransactionType_NORMAL
}

func (x *Transaction) GetPayload() []byte {
	if x != nil {
		return x.Payload
	}
	return nil
}

func (x *Transaction) GetChainID() int32 {
	if x != nil {
		return x.ChainID
	}
	return 0
}

var File_pb_upgradeable_consensus_proto protoreflect.FileDescriptor

var file_pb_upgradeable_consensus_proto_rawDesc = []byte{
	0x0a, 0x1e, 0x70, 0x62, 0x2f, 0x75, 0x70, 0x67, 0x72, 0x61, 0x64, 0x65, 0x61, 0x62, 0x6c, 0x65,
	0x2d, 0x63, 0x6f, 0x6e, 0x73, 0x65, 0x6e, 0x73, 0x75, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x02, 0x70, 0x62, 0x1a, 0x0f, 0x70, 0x62, 0x2f, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x90, 0x01, 0x0a, 0x06, 0x50, 0x61, 0x63, 0x6b, 0x65, 0x74,
	0x12, 0x10, 0x0a, 0x03, 0x6d, 0x73, 0x67, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x03, 0x6d,
	0x73, 0x67, 0x12, 0x20, 0x0a, 0x0b, 0x63, 0x6f, 0x6e, 0x73, 0x65, 0x6e, 0x73, 0x75, 0x73, 0x49,
	0x44, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x0b, 0x63, 0x6f, 0x6e, 0x73, 0x65, 0x6e, 0x73,
	0x75, 0x73, 0x49, 0x44, 0x12, 0x14, 0x0a, 0x05, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x03, 0x52, 0x05, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x68,
	0x61, 0x69, 0x6e, 0x49, 0x44, 0x18, 0x04, 0x20, 0x01, 0x28, 0x05, 0x52, 0x07, 0x63, 0x68, 0x61,
	0x69, 0x6e, 0x49, 0x44, 0x12, 0x22, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x05, 0x20, 0x01,
	0x28, 0x0e, 0x32, 0x0e, 0x2e, 0x70, 0x62, 0x2e, 0x50, 0x61, 0x63, 0x6b, 0x65, 0x74, 0x54, 0x79,
	0x70, 0x65, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x22, 0x6a, 0x0a, 0x0b, 0x54, 0x72, 0x61, 0x6e,
	0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x27, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x13, 0x2e, 0x70, 0x62, 0x2e, 0x54, 0x72, 0x61, 0x6e, 0x73,
	0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65,
	0x12, 0x18, 0x0a, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x0c, 0x52, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x68,
	0x61, 0x69, 0x6e, 0x49, 0x44, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x52, 0x07, 0x63, 0x68, 0x61,
	0x69, 0x6e, 0x49, 0x44, 0x2a, 0x2d, 0x0a, 0x0a, 0x50, 0x61, 0x63, 0x6b, 0x65, 0x74, 0x54, 0x79,
	0x70, 0x65, 0x12, 0x0d, 0x0a, 0x09, 0x50, 0x32, 0x50, 0x50, 0x41, 0x43, 0x4b, 0x45, 0x54, 0x10,
	0x00, 0x12, 0x10, 0x0a, 0x0c, 0x43, 0x4c, 0x49, 0x45, 0x4e, 0x54, 0x50, 0x41, 0x43, 0x4b, 0x45,
	0x54, 0x10, 0x01, 0x2a, 0x42, 0x0a, 0x0f, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69,
	0x6f, 0x6e, 0x54, 0x79, 0x70, 0x65, 0x12, 0x0a, 0x0a, 0x06, 0x4e, 0x4f, 0x52, 0x4d, 0x41, 0x4c,
	0x10, 0x00, 0x12, 0x0b, 0x0a, 0x07, 0x55, 0x50, 0x47, 0x52, 0x41, 0x44, 0x45, 0x10, 0x01, 0x12,
	0x0c, 0x0a, 0x08, 0x54, 0x49, 0x4d, 0x45, 0x56, 0x4f, 0x54, 0x45, 0x10, 0x02, 0x12, 0x08, 0x0a,
	0x04, 0x4c, 0x4f, 0x43, 0x4b, 0x10, 0x03, 0x32, 0x26, 0x0a, 0x03, 0x50, 0x32, 0x50, 0x12, 0x1f,
	0x0a, 0x04, 0x53, 0x65, 0x6e, 0x64, 0x12, 0x0a, 0x2e, 0x70, 0x62, 0x2e, 0x50, 0x61, 0x63, 0x6b,
	0x65, 0x74, 0x1a, 0x09, 0x2e, 0x70, 0x62, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x00, 0x42,
	0x06, 0x5a, 0x04, 0x2e, 0x2f, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_pb_upgradeable_consensus_proto_rawDescOnce sync.Once
	file_pb_upgradeable_consensus_proto_rawDescData = file_pb_upgradeable_consensus_proto_rawDesc
)

func file_pb_upgradeable_consensus_proto_rawDescGZIP() []byte {
	file_pb_upgradeable_consensus_proto_rawDescOnce.Do(func() {
		file_pb_upgradeable_consensus_proto_rawDescData = protoimpl.X.CompressGZIP(file_pb_upgradeable_consensus_proto_rawDescData)
	})
	return file_pb_upgradeable_consensus_proto_rawDescData
}

var file_pb_upgradeable_consensus_proto_enumTypes = make([]protoimpl.EnumInfo, 2)
var file_pb_upgradeable_consensus_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_pb_upgradeable_consensus_proto_goTypes = []interface{}{
	(PacketType)(0),      // 0: pb.PacketType
	(TransactionType)(0), // 1: pb.TransactionType
	(*Packet)(nil),       // 2: pb.Packet
	(*Transaction)(nil),  // 3: pb.Transaction
	(*Empty)(nil),        // 4: pb.Empty
}
var file_pb_upgradeable_consensus_proto_depIdxs = []int32{
	0, // 0: pb.Packet.type:type_name -> pb.PacketType
	1, // 1: pb.Transaction.type:type_name -> pb.TransactionType
	2, // 2: pb.P2P.Send:input_type -> pb.Packet
	4, // 3: pb.P2P.Send:output_type -> pb.Empty
	3, // [3:4] is the sub-list for method output_type
	2, // [2:3] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_pb_upgradeable_consensus_proto_init() }
func file_pb_upgradeable_consensus_proto_init() {
	if File_pb_upgradeable_consensus_proto != nil {
		return
	}
	file_pb_common_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_pb_upgradeable_consensus_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Packet); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_pb_upgradeable_consensus_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Transaction); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_pb_upgradeable_consensus_proto_rawDesc,
			NumEnums:      2,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_pb_upgradeable_consensus_proto_goTypes,
		DependencyIndexes: file_pb_upgradeable_consensus_proto_depIdxs,
		EnumInfos:         file_pb_upgradeable_consensus_proto_enumTypes,
		MessageInfos:      file_pb_upgradeable_consensus_proto_msgTypes,
	}.Build()
	File_pb_upgradeable_consensus_proto = out.File
	file_pb_upgradeable_consensus_proto_rawDesc = nil
	file_pb_upgradeable_consensus_proto_goTypes = nil
	file_pb_upgradeable_consensus_proto_depIdxs = nil
}
