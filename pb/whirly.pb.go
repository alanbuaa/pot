// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v3.21.12
// source: pb/whirly.proto

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

type WhirlyMsg struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Payload:
	//
	//	*WhirlyMsg_WhirlyProposal
	//	*WhirlyMsg_WhirlyVote
	//	*WhirlyMsg_WhirlyNewView
	//	*WhirlyMsg_Request
	//	*WhirlyMsg_Reply
	//	*WhirlyMsg_NewLeaderNotify
	//	*WhirlyMsg_NewLeaderEcho
	//	*WhirlyMsg_WhirlyPing
	Payload isWhirlyMsg_Payload `protobuf_oneof:"Payload"`
}

func (x *WhirlyMsg) Reset() {
	*x = WhirlyMsg{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pb_whirly_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *WhirlyMsg) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*WhirlyMsg) ProtoMessage() {}

func (x *WhirlyMsg) ProtoReflect() protoreflect.Message {
	mi := &file_pb_whirly_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use WhirlyMsg.ProtoReflect.Descriptor instead.
func (*WhirlyMsg) Descriptor() ([]byte, []int) {
	return file_pb_whirly_proto_rawDescGZIP(), []int{0}
}

func (m *WhirlyMsg) GetPayload() isWhirlyMsg_Payload {
	if m != nil {
		return m.Payload
	}
	return nil
}

func (x *WhirlyMsg) GetWhirlyProposal() *WhirlyProposal {
	if x, ok := x.GetPayload().(*WhirlyMsg_WhirlyProposal); ok {
		return x.WhirlyProposal
	}
	return nil
}

func (x *WhirlyMsg) GetWhirlyVote() *WhirlyVote {
	if x, ok := x.GetPayload().(*WhirlyMsg_WhirlyVote); ok {
		return x.WhirlyVote
	}
	return nil
}

func (x *WhirlyMsg) GetWhirlyNewView() *WhirlyNewView {
	if x, ok := x.GetPayload().(*WhirlyMsg_WhirlyNewView); ok {
		return x.WhirlyNewView
	}
	return nil
}

func (x *WhirlyMsg) GetRequest() *Request {
	if x, ok := x.GetPayload().(*WhirlyMsg_Request); ok {
		return x.Request
	}
	return nil
}

func (x *WhirlyMsg) GetReply() *Reply {
	if x, ok := x.GetPayload().(*WhirlyMsg_Reply); ok {
		return x.Reply
	}
	return nil
}

func (x *WhirlyMsg) GetNewLeaderNotify() *NewLeaderNotify {
	if x, ok := x.GetPayload().(*WhirlyMsg_NewLeaderNotify); ok {
		return x.NewLeaderNotify
	}
	return nil
}

func (x *WhirlyMsg) GetNewLeaderEcho() *NewLeaderEcho {
	if x, ok := x.GetPayload().(*WhirlyMsg_NewLeaderEcho); ok {
		return x.NewLeaderEcho
	}
	return nil
}

func (x *WhirlyMsg) GetWhirlyPing() *WhirlyPing {
	if x, ok := x.GetPayload().(*WhirlyMsg_WhirlyPing); ok {
		return x.WhirlyPing
	}
	return nil
}

type isWhirlyMsg_Payload interface {
	isWhirlyMsg_Payload()
}

type WhirlyMsg_WhirlyProposal struct {
	WhirlyProposal *WhirlyProposal `protobuf:"bytes,1,opt,name=whirlyProposal,proto3,oneof"`
}

type WhirlyMsg_WhirlyVote struct {
	WhirlyVote *WhirlyVote `protobuf:"bytes,2,opt,name=whirlyVote,proto3,oneof"`
}

type WhirlyMsg_WhirlyNewView struct {
	WhirlyNewView *WhirlyNewView `protobuf:"bytes,3,opt,name=whirlyNewView,proto3,oneof"`
}

type WhirlyMsg_Request struct {
	Request *Request `protobuf:"bytes,4,opt,name=request,proto3,oneof"`
}

type WhirlyMsg_Reply struct {
	Reply *Reply `protobuf:"bytes,5,opt,name=reply,proto3,oneof"`
}

type WhirlyMsg_NewLeaderNotify struct {
	NewLeaderNotify *NewLeaderNotify `protobuf:"bytes,6,opt,name=newLeaderNotify,proto3,oneof"`
}

type WhirlyMsg_NewLeaderEcho struct {
	NewLeaderEcho *NewLeaderEcho `protobuf:"bytes,7,opt,name=newLeaderEcho,proto3,oneof"`
}

type WhirlyMsg_WhirlyPing struct {
	WhirlyPing *WhirlyPing `protobuf:"bytes,8,opt,name=whirlyPing,proto3,oneof"`
}

func (*WhirlyMsg_WhirlyProposal) isWhirlyMsg_Payload() {}

func (*WhirlyMsg_WhirlyVote) isWhirlyMsg_Payload() {}

func (*WhirlyMsg_WhirlyNewView) isWhirlyMsg_Payload() {}

func (*WhirlyMsg_Request) isWhirlyMsg_Payload() {}

func (*WhirlyMsg_Reply) isWhirlyMsg_Payload() {}

func (*WhirlyMsg_NewLeaderNotify) isWhirlyMsg_Payload() {}

func (*WhirlyMsg_NewLeaderEcho) isWhirlyMsg_Payload() {}

func (*WhirlyMsg_WhirlyPing) isWhirlyMsg_Payload() {}

type SimpleWhirlyProof struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	BlockHash []byte        `protobuf:"bytes,1,opt,name=BlockHash,proto3" json:"BlockHash,omitempty"`
	ViewNum   uint64        `protobuf:"varint,2,opt,name=viewNum,proto3" json:"viewNum,omitempty"`
	Proof     []*WhirlyVote `protobuf:"bytes,3,rep,name=proof,proto3" json:"proof,omitempty"`
}

func (x *SimpleWhirlyProof) Reset() {
	*x = SimpleWhirlyProof{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pb_whirly_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SimpleWhirlyProof) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SimpleWhirlyProof) ProtoMessage() {}

func (x *SimpleWhirlyProof) ProtoReflect() protoreflect.Message {
	mi := &file_pb_whirly_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SimpleWhirlyProof.ProtoReflect.Descriptor instead.
func (*SimpleWhirlyProof) Descriptor() ([]byte, []int) {
	return file_pb_whirly_proto_rawDescGZIP(), []int{1}
}

func (x *SimpleWhirlyProof) GetBlockHash() []byte {
	if x != nil {
		return x.BlockHash
	}
	return nil
}

func (x *SimpleWhirlyProof) GetViewNum() uint64 {
	if x != nil {
		return x.ViewNum
	}
	return 0
}

func (x *SimpleWhirlyProof) GetProof() []*WhirlyVote {
	if x != nil {
		return x.Proof
	}
	return nil
}

type WhirlyProposal struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id       uint64             `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	SenderId uint64             `protobuf:"varint,2,opt,name=senderId,proto3" json:"senderId,omitempty"`
	Block    *WhirlyBlock       `protobuf:"bytes,3,opt,name=block,proto3" json:"block,omitempty"`
	HighQC   *QuorumCert        `protobuf:"bytes,4,opt,name=highQC,proto3" json:"highQC,omitempty"`
	SwProof  *SimpleWhirlyProof `protobuf:"bytes,5,opt,name=swProof,proto3" json:"swProof,omitempty"`
	Epoch    uint64             `protobuf:"varint,6,opt,name=epoch,proto3" json:"epoch,omitempty"`
	PeerId   string             `protobuf:"bytes,7,opt,name=peerId,proto3" json:"peerId,omitempty"`
}

func (x *WhirlyProposal) Reset() {
	*x = WhirlyProposal{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pb_whirly_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *WhirlyProposal) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*WhirlyProposal) ProtoMessage() {}

func (x *WhirlyProposal) ProtoReflect() protoreflect.Message {
	mi := &file_pb_whirly_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use WhirlyProposal.ProtoReflect.Descriptor instead.
func (*WhirlyProposal) Descriptor() ([]byte, []int) {
	return file_pb_whirly_proto_rawDescGZIP(), []int{2}
}

func (x *WhirlyProposal) GetId() uint64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *WhirlyProposal) GetSenderId() uint64 {
	if x != nil {
		return x.SenderId
	}
	return 0
}

func (x *WhirlyProposal) GetBlock() *WhirlyBlock {
	if x != nil {
		return x.Block
	}
	return nil
}

func (x *WhirlyProposal) GetHighQC() *QuorumCert {
	if x != nil {
		return x.HighQC
	}
	return nil
}

func (x *WhirlyProposal) GetSwProof() *SimpleWhirlyProof {
	if x != nil {
		return x.SwProof
	}
	return nil
}

func (x *WhirlyProposal) GetEpoch() uint64 {
	if x != nil {
		return x.Epoch
	}
	return 0
}

func (x *WhirlyProposal) GetPeerId() string {
	if x != nil {
		return x.PeerId
	}
	return ""
}

type WhirlyVote struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id         uint64             `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	SenderId   uint64             `protobuf:"varint,2,opt,name=senderId,proto3" json:"senderId,omitempty"`
	BlockView  uint64             `protobuf:"varint,3,opt,name=blockView,proto3" json:"blockView,omitempty"`
	BlockHash  []byte             `protobuf:"bytes,4,opt,name=BlockHash,proto3" json:"BlockHash,omitempty"`
	Flag       bool               `protobuf:"varint,5,opt,name=flag,proto3" json:"flag,omitempty"`
	Qc         *QuorumCert        `protobuf:"bytes,6,opt,name=qc,proto3" json:"qc,omitempty"`
	PartialSig []byte             `protobuf:"bytes,7,opt,name=partialSig,proto3" json:"partialSig,omitempty"`
	SwProof    *SimpleWhirlyProof `protobuf:"bytes,8,opt,name=swProof,proto3" json:"swProof,omitempty"`
	Epoch      uint64             `protobuf:"varint,9,opt,name=epoch,proto3" json:"epoch,omitempty"`
	PeerId     string             `protobuf:"bytes,10,opt,name=peerId,proto3" json:"peerId,omitempty"`
}

func (x *WhirlyVote) Reset() {
	*x = WhirlyVote{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pb_whirly_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *WhirlyVote) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*WhirlyVote) ProtoMessage() {}

func (x *WhirlyVote) ProtoReflect() protoreflect.Message {
	mi := &file_pb_whirly_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use WhirlyVote.ProtoReflect.Descriptor instead.
func (*WhirlyVote) Descriptor() ([]byte, []int) {
	return file_pb_whirly_proto_rawDescGZIP(), []int{3}
}

func (x *WhirlyVote) GetId() uint64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *WhirlyVote) GetSenderId() uint64 {
	if x != nil {
		return x.SenderId
	}
	return 0
}

func (x *WhirlyVote) GetBlockView() uint64 {
	if x != nil {
		return x.BlockView
	}
	return 0
}

func (x *WhirlyVote) GetBlockHash() []byte {
	if x != nil {
		return x.BlockHash
	}
	return nil
}

func (x *WhirlyVote) GetFlag() bool {
	if x != nil {
		return x.Flag
	}
	return false
}

func (x *WhirlyVote) GetQc() *QuorumCert {
	if x != nil {
		return x.Qc
	}
	return nil
}

func (x *WhirlyVote) GetPartialSig() []byte {
	if x != nil {
		return x.PartialSig
	}
	return nil
}

func (x *WhirlyVote) GetSwProof() *SimpleWhirlyProof {
	if x != nil {
		return x.SwProof
	}
	return nil
}

func (x *WhirlyVote) GetEpoch() uint64 {
	if x != nil {
		return x.Epoch
	}
	return 0
}

func (x *WhirlyVote) GetPeerId() string {
	if x != nil {
		return x.PeerId
	}
	return ""
}

type WhirlyNewView struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	LockQC  *QuorumCert `protobuf:"bytes,1,opt,name=lockQC,proto3" json:"lockQC,omitempty"`
	ViewNum uint64      `protobuf:"varint,2,opt,name=viewNum,proto3" json:"viewNum,omitempty"`
	Epoch   uint64      `protobuf:"varint,3,opt,name=epoch,proto3" json:"epoch,omitempty"`
	PeerId  string      `protobuf:"bytes,4,opt,name=peerId,proto3" json:"peerId,omitempty"`
}

func (x *WhirlyNewView) Reset() {
	*x = WhirlyNewView{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pb_whirly_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *WhirlyNewView) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*WhirlyNewView) ProtoMessage() {}

func (x *WhirlyNewView) ProtoReflect() protoreflect.Message {
	mi := &file_pb_whirly_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use WhirlyNewView.ProtoReflect.Descriptor instead.
func (*WhirlyNewView) Descriptor() ([]byte, []int) {
	return file_pb_whirly_proto_rawDescGZIP(), []int{4}
}

func (x *WhirlyNewView) GetLockQC() *QuorumCert {
	if x != nil {
		return x.LockQC
	}
	return nil
}

func (x *WhirlyNewView) GetViewNum() uint64 {
	if x != nil {
		return x.ViewNum
	}
	return 0
}

func (x *WhirlyNewView) GetEpoch() uint64 {
	if x != nil {
		return x.Epoch
	}
	return 0
}

func (x *WhirlyNewView) GetPeerId() string {
	if x != nil {
		return x.PeerId
	}
	return ""
}

type NewLeaderNotify struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Leader uint64 `protobuf:"varint,1,opt,name=leader,proto3" json:"leader,omitempty"`
	Epoch  uint64 `protobuf:"varint,2,opt,name=epoch,proto3" json:"epoch,omitempty"`
	Proof  []byte `protobuf:"bytes,3,opt,name=proof,proto3" json:"proof,omitempty"`
	PeerId string `protobuf:"bytes,4,opt,name=peerId,proto3" json:"peerId,omitempty"`
}

func (x *NewLeaderNotify) Reset() {
	*x = NewLeaderNotify{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pb_whirly_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NewLeaderNotify) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NewLeaderNotify) ProtoMessage() {}

func (x *NewLeaderNotify) ProtoReflect() protoreflect.Message {
	mi := &file_pb_whirly_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NewLeaderNotify.ProtoReflect.Descriptor instead.
func (*NewLeaderNotify) Descriptor() ([]byte, []int) {
	return file_pb_whirly_proto_rawDescGZIP(), []int{5}
}

func (x *NewLeaderNotify) GetLeader() uint64 {
	if x != nil {
		return x.Leader
	}
	return 0
}

func (x *NewLeaderNotify) GetEpoch() uint64 {
	if x != nil {
		return x.Epoch
	}
	return 0
}

func (x *NewLeaderNotify) GetProof() []byte {
	if x != nil {
		return x.Proof
	}
	return nil
}

func (x *NewLeaderNotify) GetPeerId() string {
	if x != nil {
		return x.PeerId
	}
	return ""
}

type NewLeaderEcho struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Leader   uint64             `protobuf:"varint,1,opt,name=leader,proto3" json:"leader,omitempty"`
	SenderId uint64             `protobuf:"varint,2,opt,name=senderId,proto3" json:"senderId,omitempty"`
	Epoch    uint64             `protobuf:"varint,3,opt,name=epoch,proto3" json:"epoch,omitempty"`
	Block    *WhirlyBlock       `protobuf:"bytes,4,opt,name=block,proto3" json:"block,omitempty"`
	SwProof  *SimpleWhirlyProof `protobuf:"bytes,5,opt,name=swProof,proto3" json:"swProof,omitempty"`
	VHeight  uint64             `protobuf:"varint,6,opt,name=vHeight,proto3" json:"vHeight,omitempty"`
	PeerId   string             `protobuf:"bytes,7,opt,name=peerId,proto3" json:"peerId,omitempty"`
}

func (x *NewLeaderEcho) Reset() {
	*x = NewLeaderEcho{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pb_whirly_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NewLeaderEcho) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NewLeaderEcho) ProtoMessage() {}

func (x *NewLeaderEcho) ProtoReflect() protoreflect.Message {
	mi := &file_pb_whirly_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NewLeaderEcho.ProtoReflect.Descriptor instead.
func (*NewLeaderEcho) Descriptor() ([]byte, []int) {
	return file_pb_whirly_proto_rawDescGZIP(), []int{6}
}

func (x *NewLeaderEcho) GetLeader() uint64 {
	if x != nil {
		return x.Leader
	}
	return 0
}

func (x *NewLeaderEcho) GetSenderId() uint64 {
	if x != nil {
		return x.SenderId
	}
	return 0
}

func (x *NewLeaderEcho) GetEpoch() uint64 {
	if x != nil {
		return x.Epoch
	}
	return 0
}

func (x *NewLeaderEcho) GetBlock() *WhirlyBlock {
	if x != nil {
		return x.Block
	}
	return nil
}

func (x *NewLeaderEcho) GetSwProof() *SimpleWhirlyProof {
	if x != nil {
		return x.SwProof
	}
	return nil
}

func (x *NewLeaderEcho) GetVHeight() uint64 {
	if x != nil {
		return x.VHeight
	}
	return 0
}

func (x *NewLeaderEcho) GetPeerId() string {
	if x != nil {
		return x.PeerId
	}
	return ""
}

type WhirlyPing struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id     uint64 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	PeerId string `protobuf:"bytes,2,opt,name=peerId,proto3" json:"peerId,omitempty"`
}

func (x *WhirlyPing) Reset() {
	*x = WhirlyPing{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pb_whirly_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *WhirlyPing) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*WhirlyPing) ProtoMessage() {}

func (x *WhirlyPing) ProtoReflect() protoreflect.Message {
	mi := &file_pb_whirly_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use WhirlyPing.ProtoReflect.Descriptor instead.
func (*WhirlyPing) Descriptor() ([]byte, []int) {
	return file_pb_whirly_proto_rawDescGZIP(), []int{7}
}

func (x *WhirlyPing) GetId() uint64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *WhirlyPing) GetPeerId() string {
	if x != nil {
		return x.PeerId
	}
	return ""
}

var File_pb_whirly_proto protoreflect.FileDescriptor

var file_pb_whirly_proto_rawDesc = []byte{
	0x0a, 0x0f, 0x70, 0x62, 0x2f, 0x77, 0x68, 0x69, 0x72, 0x6c, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x02, 0x70, 0x62, 0x1a, 0x0f, 0x70, 0x62, 0x2f, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xbb, 0x03, 0x0a, 0x09, 0x57, 0x68, 0x69, 0x72, 0x6c,
	0x79, 0x4d, 0x73, 0x67, 0x12, 0x3c, 0x0a, 0x0e, 0x77, 0x68, 0x69, 0x72, 0x6c, 0x79, 0x50, 0x72,
	0x6f, 0x70, 0x6f, 0x73, 0x61, 0x6c, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x70,
	0x62, 0x2e, 0x57, 0x68, 0x69, 0x72, 0x6c, 0x79, 0x50, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x61, 0x6c,
	0x48, 0x00, 0x52, 0x0e, 0x77, 0x68, 0x69, 0x72, 0x6c, 0x79, 0x50, 0x72, 0x6f, 0x70, 0x6f, 0x73,
	0x61, 0x6c, 0x12, 0x30, 0x0a, 0x0a, 0x77, 0x68, 0x69, 0x72, 0x6c, 0x79, 0x56, 0x6f, 0x74, 0x65,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x70, 0x62, 0x2e, 0x57, 0x68, 0x69, 0x72,
	0x6c, 0x79, 0x56, 0x6f, 0x74, 0x65, 0x48, 0x00, 0x52, 0x0a, 0x77, 0x68, 0x69, 0x72, 0x6c, 0x79,
	0x56, 0x6f, 0x74, 0x65, 0x12, 0x39, 0x0a, 0x0d, 0x77, 0x68, 0x69, 0x72, 0x6c, 0x79, 0x4e, 0x65,
	0x77, 0x56, 0x69, 0x65, 0x77, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x11, 0x2e, 0x70, 0x62,
	0x2e, 0x57, 0x68, 0x69, 0x72, 0x6c, 0x79, 0x4e, 0x65, 0x77, 0x56, 0x69, 0x65, 0x77, 0x48, 0x00,
	0x52, 0x0d, 0x77, 0x68, 0x69, 0x72, 0x6c, 0x79, 0x4e, 0x65, 0x77, 0x56, 0x69, 0x65, 0x77, 0x12,
	0x27, 0x0a, 0x07, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x0b, 0x2e, 0x70, 0x62, 0x2e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x48, 0x00, 0x52,
	0x07, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x21, 0x0a, 0x05, 0x72, 0x65, 0x70, 0x6c,
	0x79, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x09, 0x2e, 0x70, 0x62, 0x2e, 0x52, 0x65, 0x70,
	0x6c, 0x79, 0x48, 0x00, 0x52, 0x05, 0x72, 0x65, 0x70, 0x6c, 0x79, 0x12, 0x3f, 0x0a, 0x0f, 0x6e,
	0x65, 0x77, 0x4c, 0x65, 0x61, 0x64, 0x65, 0x72, 0x4e, 0x6f, 0x74, 0x69, 0x66, 0x79, 0x18, 0x06,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x70, 0x62, 0x2e, 0x4e, 0x65, 0x77, 0x4c, 0x65, 0x61,
	0x64, 0x65, 0x72, 0x4e, 0x6f, 0x74, 0x69, 0x66, 0x79, 0x48, 0x00, 0x52, 0x0f, 0x6e, 0x65, 0x77,
	0x4c, 0x65, 0x61, 0x64, 0x65, 0x72, 0x4e, 0x6f, 0x74, 0x69, 0x66, 0x79, 0x12, 0x39, 0x0a, 0x0d,
	0x6e, 0x65, 0x77, 0x4c, 0x65, 0x61, 0x64, 0x65, 0x72, 0x45, 0x63, 0x68, 0x6f, 0x18, 0x07, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x11, 0x2e, 0x70, 0x62, 0x2e, 0x4e, 0x65, 0x77, 0x4c, 0x65, 0x61, 0x64,
	0x65, 0x72, 0x45, 0x63, 0x68, 0x6f, 0x48, 0x00, 0x52, 0x0d, 0x6e, 0x65, 0x77, 0x4c, 0x65, 0x61,
	0x64, 0x65, 0x72, 0x45, 0x63, 0x68, 0x6f, 0x12, 0x30, 0x0a, 0x0a, 0x77, 0x68, 0x69, 0x72, 0x6c,
	0x79, 0x50, 0x69, 0x6e, 0x67, 0x18, 0x08, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x70, 0x62,
	0x2e, 0x57, 0x68, 0x69, 0x72, 0x6c, 0x79, 0x50, 0x69, 0x6e, 0x67, 0x48, 0x00, 0x52, 0x0a, 0x77,
	0x68, 0x69, 0x72, 0x6c, 0x79, 0x50, 0x69, 0x6e, 0x67, 0x42, 0x09, 0x0a, 0x07, 0x50, 0x61, 0x79,
	0x6c, 0x6f, 0x61, 0x64, 0x22, 0x71, 0x0a, 0x11, 0x53, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x57, 0x68,
	0x69, 0x72, 0x6c, 0x79, 0x50, 0x72, 0x6f, 0x6f, 0x66, 0x12, 0x1c, 0x0a, 0x09, 0x42, 0x6c, 0x6f,
	0x63, 0x6b, 0x48, 0x61, 0x73, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x42, 0x6c,
	0x6f, 0x63, 0x6b, 0x48, 0x61, 0x73, 0x68, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x69, 0x65, 0x77, 0x4e,
	0x75, 0x6d, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x07, 0x76, 0x69, 0x65, 0x77, 0x4e, 0x75,
	0x6d, 0x12, 0x24, 0x0a, 0x05, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x0e, 0x2e, 0x70, 0x62, 0x2e, 0x57, 0x68, 0x69, 0x72, 0x6c, 0x79, 0x56, 0x6f, 0x74, 0x65,
	0x52, 0x05, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x22, 0xea, 0x01, 0x0a, 0x0e, 0x57, 0x68, 0x69, 0x72,
	0x6c, 0x79, 0x50, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x61, 0x6c, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x02, 0x69, 0x64, 0x12, 0x1a, 0x0a, 0x08, 0x73, 0x65,
	0x6e, 0x64, 0x65, 0x72, 0x49, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x08, 0x73, 0x65,
	0x6e, 0x64, 0x65, 0x72, 0x49, 0x64, 0x12, 0x25, 0x0a, 0x05, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0f, 0x2e, 0x70, 0x62, 0x2e, 0x57, 0x68, 0x69, 0x72, 0x6c,
	0x79, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x52, 0x05, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x12, 0x26, 0x0a,
	0x06, 0x68, 0x69, 0x67, 0x68, 0x51, 0x43, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e,
	0x70, 0x62, 0x2e, 0x51, 0x75, 0x6f, 0x72, 0x75, 0x6d, 0x43, 0x65, 0x72, 0x74, 0x52, 0x06, 0x68,
	0x69, 0x67, 0x68, 0x51, 0x43, 0x12, 0x2f, 0x0a, 0x07, 0x73, 0x77, 0x50, 0x72, 0x6f, 0x6f, 0x66,
	0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x15, 0x2e, 0x70, 0x62, 0x2e, 0x53, 0x69, 0x6d, 0x70,
	0x6c, 0x65, 0x57, 0x68, 0x69, 0x72, 0x6c, 0x79, 0x50, 0x72, 0x6f, 0x6f, 0x66, 0x52, 0x07, 0x73,
	0x77, 0x50, 0x72, 0x6f, 0x6f, 0x66, 0x12, 0x14, 0x0a, 0x05, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x18,
	0x06, 0x20, 0x01, 0x28, 0x04, 0x52, 0x05, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x12, 0x16, 0x0a, 0x06,
	0x70, 0x65, 0x65, 0x72, 0x49, 0x64, 0x18, 0x07, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x70, 0x65,
	0x65, 0x72, 0x49, 0x64, 0x22, 0xa7, 0x02, 0x0a, 0x0a, 0x57, 0x68, 0x69, 0x72, 0x6c, 0x79, 0x56,
	0x6f, 0x74, 0x65, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52,
	0x02, 0x69, 0x64, 0x12, 0x1a, 0x0a, 0x08, 0x73, 0x65, 0x6e, 0x64, 0x65, 0x72, 0x49, 0x64, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x08, 0x73, 0x65, 0x6e, 0x64, 0x65, 0x72, 0x49, 0x64, 0x12,
	0x1c, 0x0a, 0x09, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x56, 0x69, 0x65, 0x77, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x04, 0x52, 0x09, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x56, 0x69, 0x65, 0x77, 0x12, 0x1c, 0x0a,
	0x09, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x48, 0x61, 0x73, 0x68, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0c,
	0x52, 0x09, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x48, 0x61, 0x73, 0x68, 0x12, 0x12, 0x0a, 0x04, 0x66,
	0x6c, 0x61, 0x67, 0x18, 0x05, 0x20, 0x01, 0x28, 0x08, 0x52, 0x04, 0x66, 0x6c, 0x61, 0x67, 0x12,
	0x1e, 0x0a, 0x02, 0x71, 0x63, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x70, 0x62,
	0x2e, 0x51, 0x75, 0x6f, 0x72, 0x75, 0x6d, 0x43, 0x65, 0x72, 0x74, 0x52, 0x02, 0x71, 0x63, 0x12,
	0x1e, 0x0a, 0x0a, 0x70, 0x61, 0x72, 0x74, 0x69, 0x61, 0x6c, 0x53, 0x69, 0x67, 0x18, 0x07, 0x20,
	0x01, 0x28, 0x0c, 0x52, 0x0a, 0x70, 0x61, 0x72, 0x74, 0x69, 0x61, 0x6c, 0x53, 0x69, 0x67, 0x12,
	0x2f, 0x0a, 0x07, 0x73, 0x77, 0x50, 0x72, 0x6f, 0x6f, 0x66, 0x18, 0x08, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x15, 0x2e, 0x70, 0x62, 0x2e, 0x53, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x57, 0x68, 0x69, 0x72,
	0x6c, 0x79, 0x50, 0x72, 0x6f, 0x6f, 0x66, 0x52, 0x07, 0x73, 0x77, 0x50, 0x72, 0x6f, 0x6f, 0x66,
	0x12, 0x14, 0x0a, 0x05, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x18, 0x09, 0x20, 0x01, 0x28, 0x04, 0x52,
	0x05, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x12, 0x16, 0x0a, 0x06, 0x70, 0x65, 0x65, 0x72, 0x49, 0x64,
	0x18, 0x0a, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x70, 0x65, 0x65, 0x72, 0x49, 0x64, 0x22, 0x7f,
	0x0a, 0x0d, 0x57, 0x68, 0x69, 0x72, 0x6c, 0x79, 0x4e, 0x65, 0x77, 0x56, 0x69, 0x65, 0x77, 0x12,
	0x26, 0x0a, 0x06, 0x6c, 0x6f, 0x63, 0x6b, 0x51, 0x43, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x0e, 0x2e, 0x70, 0x62, 0x2e, 0x51, 0x75, 0x6f, 0x72, 0x75, 0x6d, 0x43, 0x65, 0x72, 0x74, 0x52,
	0x06, 0x6c, 0x6f, 0x63, 0x6b, 0x51, 0x43, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x69, 0x65, 0x77, 0x4e,
	0x75, 0x6d, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x07, 0x76, 0x69, 0x65, 0x77, 0x4e, 0x75,
	0x6d, 0x12, 0x14, 0x0a, 0x05, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x18, 0x03, 0x20, 0x01, 0x28, 0x04,
	0x52, 0x05, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x12, 0x16, 0x0a, 0x06, 0x70, 0x65, 0x65, 0x72, 0x49,
	0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x70, 0x65, 0x65, 0x72, 0x49, 0x64, 0x22,
	0x6d, 0x0a, 0x0f, 0x4e, 0x65, 0x77, 0x4c, 0x65, 0x61, 0x64, 0x65, 0x72, 0x4e, 0x6f, 0x74, 0x69,
	0x66, 0x79, 0x12, 0x16, 0x0a, 0x06, 0x6c, 0x65, 0x61, 0x64, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x04, 0x52, 0x06, 0x6c, 0x65, 0x61, 0x64, 0x65, 0x72, 0x12, 0x14, 0x0a, 0x05, 0x65, 0x70,
	0x6f, 0x63, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x05, 0x65, 0x70, 0x6f, 0x63, 0x68,
	0x12, 0x14, 0x0a, 0x05, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0c, 0x52,
	0x05, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x12, 0x16, 0x0a, 0x06, 0x70, 0x65, 0x65, 0x72, 0x49, 0x64,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x70, 0x65, 0x65, 0x72, 0x49, 0x64, 0x22, 0xe3,
	0x01, 0x0a, 0x0d, 0x4e, 0x65, 0x77, 0x4c, 0x65, 0x61, 0x64, 0x65, 0x72, 0x45, 0x63, 0x68, 0x6f,
	0x12, 0x16, 0x0a, 0x06, 0x6c, 0x65, 0x61, 0x64, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04,
	0x52, 0x06, 0x6c, 0x65, 0x61, 0x64, 0x65, 0x72, 0x12, 0x1a, 0x0a, 0x08, 0x73, 0x65, 0x6e, 0x64,
	0x65, 0x72, 0x49, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x08, 0x73, 0x65, 0x6e, 0x64,
	0x65, 0x72, 0x49, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x04, 0x52, 0x05, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x12, 0x25, 0x0a, 0x05, 0x62, 0x6c,
	0x6f, 0x63, 0x6b, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0f, 0x2e, 0x70, 0x62, 0x2e, 0x57,
	0x68, 0x69, 0x72, 0x6c, 0x79, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x52, 0x05, 0x62, 0x6c, 0x6f, 0x63,
	0x6b, 0x12, 0x2f, 0x0a, 0x07, 0x73, 0x77, 0x50, 0x72, 0x6f, 0x6f, 0x66, 0x18, 0x05, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x15, 0x2e, 0x70, 0x62, 0x2e, 0x53, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x57, 0x68,
	0x69, 0x72, 0x6c, 0x79, 0x50, 0x72, 0x6f, 0x6f, 0x66, 0x52, 0x07, 0x73, 0x77, 0x50, 0x72, 0x6f,
	0x6f, 0x66, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x48, 0x65, 0x69, 0x67, 0x68, 0x74, 0x18, 0x06, 0x20,
	0x01, 0x28, 0x04, 0x52, 0x07, 0x76, 0x48, 0x65, 0x69, 0x67, 0x68, 0x74, 0x12, 0x16, 0x0a, 0x06,
	0x70, 0x65, 0x65, 0x72, 0x49, 0x64, 0x18, 0x07, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x70, 0x65,
	0x65, 0x72, 0x49, 0x64, 0x22, 0x34, 0x0a, 0x0a, 0x57, 0x68, 0x69, 0x72, 0x6c, 0x79, 0x50, 0x69,
	0x6e, 0x67, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x02,
	0x69, 0x64, 0x12, 0x16, 0x0a, 0x06, 0x70, 0x65, 0x65, 0x72, 0x49, 0x64, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x06, 0x70, 0x65, 0x65, 0x72, 0x49, 0x64, 0x42, 0x06, 0x5a, 0x04, 0x2e, 0x2f,
	0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_pb_whirly_proto_rawDescOnce sync.Once
	file_pb_whirly_proto_rawDescData = file_pb_whirly_proto_rawDesc
)

func file_pb_whirly_proto_rawDescGZIP() []byte {
	file_pb_whirly_proto_rawDescOnce.Do(func() {
		file_pb_whirly_proto_rawDescData = protoimpl.X.CompressGZIP(file_pb_whirly_proto_rawDescData)
	})
	return file_pb_whirly_proto_rawDescData
}

var file_pb_whirly_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_pb_whirly_proto_goTypes = []interface{}{
	(*WhirlyMsg)(nil),         // 0: pb.WhirlyMsg
	(*SimpleWhirlyProof)(nil), // 1: pb.SimpleWhirlyProof
	(*WhirlyProposal)(nil),    // 2: pb.WhirlyProposal
	(*WhirlyVote)(nil),        // 3: pb.WhirlyVote
	(*WhirlyNewView)(nil),     // 4: pb.WhirlyNewView
	(*NewLeaderNotify)(nil),   // 5: pb.NewLeaderNotify
	(*NewLeaderEcho)(nil),     // 6: pb.NewLeaderEcho
	(*WhirlyPing)(nil),        // 7: pb.WhirlyPing
	(*Request)(nil),           // 8: pb.Request
	(*Reply)(nil),             // 9: pb.Reply
	(*WhirlyBlock)(nil),       // 10: pb.WhirlyBlock
	(*QuorumCert)(nil),        // 11: pb.QuorumCert
}
var file_pb_whirly_proto_depIdxs = []int32{
	2,  // 0: pb.WhirlyMsg.whirlyProposal:type_name -> pb.WhirlyProposal
	3,  // 1: pb.WhirlyMsg.whirlyVote:type_name -> pb.WhirlyVote
	4,  // 2: pb.WhirlyMsg.whirlyNewView:type_name -> pb.WhirlyNewView
	8,  // 3: pb.WhirlyMsg.request:type_name -> pb.Request
	9,  // 4: pb.WhirlyMsg.reply:type_name -> pb.Reply
	5,  // 5: pb.WhirlyMsg.newLeaderNotify:type_name -> pb.NewLeaderNotify
	6,  // 6: pb.WhirlyMsg.newLeaderEcho:type_name -> pb.NewLeaderEcho
	7,  // 7: pb.WhirlyMsg.whirlyPing:type_name -> pb.WhirlyPing
	3,  // 8: pb.SimpleWhirlyProof.proof:type_name -> pb.WhirlyVote
	10, // 9: pb.WhirlyProposal.block:type_name -> pb.WhirlyBlock
	11, // 10: pb.WhirlyProposal.highQC:type_name -> pb.QuorumCert
	1,  // 11: pb.WhirlyProposal.swProof:type_name -> pb.SimpleWhirlyProof
	11, // 12: pb.WhirlyVote.qc:type_name -> pb.QuorumCert
	1,  // 13: pb.WhirlyVote.swProof:type_name -> pb.SimpleWhirlyProof
	11, // 14: pb.WhirlyNewView.lockQC:type_name -> pb.QuorumCert
	10, // 15: pb.NewLeaderEcho.block:type_name -> pb.WhirlyBlock
	1,  // 16: pb.NewLeaderEcho.swProof:type_name -> pb.SimpleWhirlyProof
	17, // [17:17] is the sub-list for method output_type
	17, // [17:17] is the sub-list for method input_type
	17, // [17:17] is the sub-list for extension type_name
	17, // [17:17] is the sub-list for extension extendee
	0,  // [0:17] is the sub-list for field type_name
}

func init() { file_pb_whirly_proto_init() }
func file_pb_whirly_proto_init() {
	if File_pb_whirly_proto != nil {
		return
	}
	file_pb_common_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_pb_whirly_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*WhirlyMsg); i {
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
		file_pb_whirly_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SimpleWhirlyProof); i {
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
		file_pb_whirly_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*WhirlyProposal); i {
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
		file_pb_whirly_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*WhirlyVote); i {
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
		file_pb_whirly_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*WhirlyNewView); i {
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
		file_pb_whirly_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*NewLeaderNotify); i {
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
		file_pb_whirly_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*NewLeaderEcho); i {
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
		file_pb_whirly_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*WhirlyPing); i {
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
	file_pb_whirly_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*WhirlyMsg_WhirlyProposal)(nil),
		(*WhirlyMsg_WhirlyVote)(nil),
		(*WhirlyMsg_WhirlyNewView)(nil),
		(*WhirlyMsg_Request)(nil),
		(*WhirlyMsg_Reply)(nil),
		(*WhirlyMsg_NewLeaderNotify)(nil),
		(*WhirlyMsg_NewLeaderEcho)(nil),
		(*WhirlyMsg_WhirlyPing)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_pb_whirly_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_pb_whirly_proto_goTypes,
		DependencyIndexes: file_pb_whirly_proto_depIdxs,
		MessageInfos:      file_pb_whirly_proto_msgTypes,
	}.Build()
	File_pb_whirly_proto = out.File
	file_pb_whirly_proto_rawDesc = nil
	file_pb_whirly_proto_goTypes = nil
	file_pb_whirly_proto_depIdxs = nil
}
