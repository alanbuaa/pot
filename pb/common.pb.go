// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v4.22.3
// source: pb/common.proto

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

type MsgType int32

const (
	MsgType_PREPARE        MsgType = 0
	MsgType_PREPARE_VOTE   MsgType = 1
	MsgType_PRECOMMIT      MsgType = 2
	MsgType_PRECOMMIT_VOTE MsgType = 3
	MsgType_COMMIT         MsgType = 4
	MsgType_COMMIT_VOTE    MsgType = 5
	MsgType_NEWVIEW        MsgType = 6
	MsgType_DECIDE         MsgType = 7
)

// Enum value maps for MsgType.
var (
	MsgType_name = map[int32]string{
		0: "PREPARE",
		1: "PREPARE_VOTE",
		2: "PRECOMMIT",
		3: "PRECOMMIT_VOTE",
		4: "COMMIT",
		5: "COMMIT_VOTE",
		6: "NEWVIEW",
		7: "DECIDE",
	}
	MsgType_value = map[string]int32{
		"PREPARE":        0,
		"PREPARE_VOTE":   1,
		"PRECOMMIT":      2,
		"PRECOMMIT_VOTE": 3,
		"COMMIT":         4,
		"COMMIT_VOTE":    5,
		"NEWVIEW":        6,
		"DECIDE":         7,
	}
)

func (x MsgType) Enum() *MsgType {
	p := new(MsgType)
	*p = x
	return p
}

func (x MsgType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (MsgType) Descriptor() protoreflect.EnumDescriptor {
	return file_pb_common_proto_enumTypes[0].Descriptor()
}

func (MsgType) Type() protoreflect.EnumType {
	return &file_pb_common_proto_enumTypes[0]
}

func (x MsgType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use MsgType.Descriptor instead.
func (MsgType) EnumDescriptor() ([]byte, []int) {
	return file_pb_common_proto_rawDescGZIP(), []int{0}
}

type Empty struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *Empty) Reset() {
	*x = Empty{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pb_common_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Empty) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Empty) ProtoMessage() {}

func (x *Empty) ProtoReflect() protoreflect.Message {
	mi := &file_pb_common_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Empty.ProtoReflect.Descriptor instead.
func (*Empty) Descriptor() ([]byte, []int) {
	return file_pb_common_proto_rawDescGZIP(), []int{0}
}

type WhirlyBlock struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ParentHash []byte      `protobuf:"bytes,1,opt,name=ParentHash,proto3" json:"ParentHash,omitempty"`
	Hash       []byte      `protobuf:"bytes,2,opt,name=Hash,proto3" json:"Hash,omitempty"`
	Height     uint64      `protobuf:"varint,3,opt,name=height,proto3" json:"height,omitempty"`
	Txs        [][]byte    `protobuf:"bytes,4,rep,name=txs,proto3" json:"txs,omitempty"`
	Justify    *QuorumCert `protobuf:"bytes,5,opt,name=Justify,proto3" json:"Justify,omitempty"`
	Committed  bool        `protobuf:"varint,6,opt,name=committed,proto3" json:"committed,omitempty"`
}

func (x *WhirlyBlock) Reset() {
	*x = WhirlyBlock{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pb_common_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *WhirlyBlock) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*WhirlyBlock) ProtoMessage() {}

func (x *WhirlyBlock) ProtoReflect() protoreflect.Message {
	mi := &file_pb_common_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use WhirlyBlock.ProtoReflect.Descriptor instead.
func (*WhirlyBlock) Descriptor() ([]byte, []int) {
	return file_pb_common_proto_rawDescGZIP(), []int{1}
}

func (x *WhirlyBlock) GetParentHash() []byte {
	if x != nil {
		return x.ParentHash
	}
	return nil
}

func (x *WhirlyBlock) GetHash() []byte {
	if x != nil {
		return x.Hash
	}
	return nil
}

func (x *WhirlyBlock) GetHeight() uint64 {
	if x != nil {
		return x.Height
	}
	return 0
}

func (x *WhirlyBlock) GetTxs() [][]byte {
	if x != nil {
		return x.Txs
	}
	return nil
}

func (x *WhirlyBlock) GetJustify() *QuorumCert {
	if x != nil {
		return x.Justify
	}
	return nil
}

func (x *WhirlyBlock) GetCommitted() bool {
	if x != nil {
		return x.Committed
	}
	return false
}

type QuorumCert struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	BlockHash []byte  `protobuf:"bytes,1,opt,name=BlockHash,proto3" json:"BlockHash,omitempty"`
	Type      MsgType `protobuf:"varint,2,opt,name=type,proto3,enum=pb.MsgType" json:"type,omitempty"`
	ViewNum   uint64  `protobuf:"varint,3,opt,name=viewNum,proto3" json:"viewNum,omitempty"`
	Signature []byte  `protobuf:"bytes,4,opt,name=signature,proto3" json:"signature,omitempty"`
}

func (x *QuorumCert) Reset() {
	*x = QuorumCert{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pb_common_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *QuorumCert) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QuorumCert) ProtoMessage() {}

func (x *QuorumCert) ProtoReflect() protoreflect.Message {
	mi := &file_pb_common_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QuorumCert.ProtoReflect.Descriptor instead.
func (*QuorumCert) Descriptor() ([]byte, []int) {
	return file_pb_common_proto_rawDescGZIP(), []int{2}
}

func (x *QuorumCert) GetBlockHash() []byte {
	if x != nil {
		return x.BlockHash
	}
	return nil
}

func (x *QuorumCert) GetType() MsgType {
	if x != nil {
		return x.Type
	}
	return MsgType_PREPARE
}

func (x *QuorumCert) GetViewNum() uint64 {
	if x != nil {
		return x.ViewNum
	}
	return 0
}

func (x *QuorumCert) GetSignature() []byte {
	if x != nil {
		return x.Signature
	}
	return nil
}

type Request struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Tx      []byte `protobuf:"bytes,1,opt,name=tx,proto3" json:"tx,omitempty"`
	ChainID int32  `protobuf:"varint,2,opt,name=chainID,proto3" json:"chainID,omitempty"`
}

func (x *Request) Reset() {
	*x = Request{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pb_common_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Request) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Request) ProtoMessage() {}

func (x *Request) ProtoReflect() protoreflect.Message {
	mi := &file_pb_common_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Request.ProtoReflect.Descriptor instead.
func (*Request) Descriptor() ([]byte, []int) {
	return file_pb_common_proto_rawDescGZIP(), []int{3}
}

func (x *Request) GetTx() []byte {
	if x != nil {
		return x.Tx
	}
	return nil
}

func (x *Request) GetChainID() int32 {
	if x != nil {
		return x.ChainID
	}
	return 0
}

type Reply struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Tx      []byte `protobuf:"bytes,1,opt,name=tx,proto3" json:"tx,omitempty"`
	Receipt []byte `protobuf:"bytes,2,opt,name=receipt,proto3" json:"receipt,omitempty"`
	ChainID int32  `protobuf:"varint,3,opt,name=chainID,proto3" json:"chainID,omitempty"`
}

func (x *Reply) Reset() {
	*x = Reply{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pb_common_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Reply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Reply) ProtoMessage() {}

func (x *Reply) ProtoReflect() protoreflect.Message {
	mi := &file_pb_common_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Reply.ProtoReflect.Descriptor instead.
func (*Reply) Descriptor() ([]byte, []int) {
	return file_pb_common_proto_rawDescGZIP(), []int{4}
}

func (x *Reply) GetTx() []byte {
	if x != nil {
		return x.Tx
	}
	return nil
}

func (x *Reply) GetReceipt() []byte {
	if x != nil {
		return x.Receipt
	}
	return nil
}

func (x *Reply) GetChainID() int32 {
	if x != nil {
		return x.ChainID
	}
	return 0
}

var File_pb_common_proto protoreflect.FileDescriptor

var file_pb_common_proto_rawDesc = []byte{
	0x0a, 0x0f, 0x70, 0x62, 0x2f, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x02, 0x70, 0x62, 0x22, 0x07, 0x0a, 0x05, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0xb3,
	0x01, 0x0a, 0x0b, 0x57, 0x68, 0x69, 0x72, 0x6c, 0x79, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x12, 0x1e,
	0x0a, 0x0a, 0x50, 0x61, 0x72, 0x65, 0x6e, 0x74, 0x48, 0x61, 0x73, 0x68, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0c, 0x52, 0x0a, 0x50, 0x61, 0x72, 0x65, 0x6e, 0x74, 0x48, 0x61, 0x73, 0x68, 0x12, 0x12,
	0x0a, 0x04, 0x48, 0x61, 0x73, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x48, 0x61,
	0x73, 0x68, 0x12, 0x16, 0x0a, 0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x04, 0x52, 0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x12, 0x10, 0x0a, 0x03, 0x74, 0x78,
	0x73, 0x18, 0x04, 0x20, 0x03, 0x28, 0x0c, 0x52, 0x03, 0x74, 0x78, 0x73, 0x12, 0x28, 0x0a, 0x07,
	0x4a, 0x75, 0x73, 0x74, 0x69, 0x66, 0x79, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e,
	0x70, 0x62, 0x2e, 0x51, 0x75, 0x6f, 0x72, 0x75, 0x6d, 0x43, 0x65, 0x72, 0x74, 0x52, 0x07, 0x4a,
	0x75, 0x73, 0x74, 0x69, 0x66, 0x79, 0x12, 0x1c, 0x0a, 0x09, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74,
	0x74, 0x65, 0x64, 0x18, 0x06, 0x20, 0x01, 0x28, 0x08, 0x52, 0x09, 0x63, 0x6f, 0x6d, 0x6d, 0x69,
	0x74, 0x74, 0x65, 0x64, 0x22, 0x83, 0x01, 0x0a, 0x0a, 0x51, 0x75, 0x6f, 0x72, 0x75, 0x6d, 0x43,
	0x65, 0x72, 0x74, 0x12, 0x1c, 0x0a, 0x09, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x48, 0x61, 0x73, 0x68,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x48, 0x61, 0x73,
	0x68, 0x12, 0x1f, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32,
	0x0b, 0x2e, 0x70, 0x62, 0x2e, 0x4d, 0x73, 0x67, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74, 0x79,
	0x70, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x69, 0x65, 0x77, 0x4e, 0x75, 0x6d, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x04, 0x52, 0x07, 0x76, 0x69, 0x65, 0x77, 0x4e, 0x75, 0x6d, 0x12, 0x1c, 0x0a, 0x09,
	0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0c, 0x52,
	0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x22, 0x33, 0x0a, 0x07, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x0e, 0x0a, 0x02, 0x74, 0x78, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0c, 0x52, 0x02, 0x74, 0x78, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x49, 0x44,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x07, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x49, 0x44, 0x22,
	0x4b, 0x0a, 0x05, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x12, 0x0e, 0x0a, 0x02, 0x74, 0x78, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0c, 0x52, 0x02, 0x74, 0x78, 0x12, 0x18, 0x0a, 0x07, 0x72, 0x65, 0x63, 0x65,
	0x69, 0x70, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x72, 0x65, 0x63, 0x65, 0x69,
	0x70, 0x74, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x49, 0x44, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x05, 0x52, 0x07, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x49, 0x44, 0x2a, 0x81, 0x01, 0x0a,
	0x07, 0x4d, 0x73, 0x67, 0x54, 0x79, 0x70, 0x65, 0x12, 0x0b, 0x0a, 0x07, 0x50, 0x52, 0x45, 0x50,
	0x41, 0x52, 0x45, 0x10, 0x00, 0x12, 0x10, 0x0a, 0x0c, 0x50, 0x52, 0x45, 0x50, 0x41, 0x52, 0x45,
	0x5f, 0x56, 0x4f, 0x54, 0x45, 0x10, 0x01, 0x12, 0x0d, 0x0a, 0x09, 0x50, 0x52, 0x45, 0x43, 0x4f,
	0x4d, 0x4d, 0x49, 0x54, 0x10, 0x02, 0x12, 0x12, 0x0a, 0x0e, 0x50, 0x52, 0x45, 0x43, 0x4f, 0x4d,
	0x4d, 0x49, 0x54, 0x5f, 0x56, 0x4f, 0x54, 0x45, 0x10, 0x03, 0x12, 0x0a, 0x0a, 0x06, 0x43, 0x4f,
	0x4d, 0x4d, 0x49, 0x54, 0x10, 0x04, 0x12, 0x0f, 0x0a, 0x0b, 0x43, 0x4f, 0x4d, 0x4d, 0x49, 0x54,
	0x5f, 0x56, 0x4f, 0x54, 0x45, 0x10, 0x05, 0x12, 0x0b, 0x0a, 0x07, 0x4e, 0x45, 0x57, 0x56, 0x49,
	0x45, 0x57, 0x10, 0x06, 0x12, 0x0a, 0x0a, 0x06, 0x44, 0x45, 0x43, 0x49, 0x44, 0x45, 0x10, 0x07,
	0x42, 0x06, 0x5a, 0x04, 0x2e, 0x2f, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_pb_common_proto_rawDescOnce sync.Once
	file_pb_common_proto_rawDescData = file_pb_common_proto_rawDesc
)

func file_pb_common_proto_rawDescGZIP() []byte {
	file_pb_common_proto_rawDescOnce.Do(func() {
		file_pb_common_proto_rawDescData = protoimpl.X.CompressGZIP(file_pb_common_proto_rawDescData)
	})
	return file_pb_common_proto_rawDescData
}

var file_pb_common_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_pb_common_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_pb_common_proto_goTypes = []interface{}{
	(MsgType)(0),        // 0: pb.MsgType
	(*Empty)(nil),       // 1: pb.Empty
	(*WhirlyBlock)(nil), // 2: pb.WhirlyBlock
	(*QuorumCert)(nil),  // 3: pb.QuorumCert
	(*Request)(nil),     // 4: pb.Request
	(*Reply)(nil),       // 5: pb.Reply
}
var file_pb_common_proto_depIdxs = []int32{
	3, // 0: pb.WhirlyBlock.Justify:type_name -> pb.QuorumCert
	0, // 1: pb.QuorumCert.type:type_name -> pb.MsgType
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_pb_common_proto_init() }
func file_pb_common_proto_init() {
	if File_pb_common_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_pb_common_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Empty); i {
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
		file_pb_common_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*WhirlyBlock); i {
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
		file_pb_common_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*QuorumCert); i {
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
		file_pb_common_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Request); i {
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
		file_pb_common_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Reply); i {
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
			RawDescriptor: file_pb_common_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_pb_common_proto_goTypes,
		DependencyIndexes: file_pb_common_proto_depIdxs,
		EnumInfos:         file_pb_common_proto_enumTypes,
		MessageInfos:      file_pb_common_proto_msgTypes,
	}.Build()
	File_pb_common_proto = out.File
	file_pb_common_proto_rawDesc = nil
	file_pb_common_proto_goTypes = nil
	file_pb_common_proto_depIdxs = nil
}
