// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        v3.12.4
// source: rpc.proto

package rrpc

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

type GetCommitIndexArgs struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *GetCommitIndexArgs) Reset() {
	*x = GetCommitIndexArgs{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpc_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetCommitIndexArgs) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetCommitIndexArgs) ProtoMessage() {}

func (x *GetCommitIndexArgs) ProtoReflect() protoreflect.Message {
	mi := &file_rpc_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetCommitIndexArgs.ProtoReflect.Descriptor instead.
func (*GetCommitIndexArgs) Descriptor() ([]byte, []int) {
	return file_rpc_proto_rawDescGZIP(), []int{0}
}

type GetCommitIndexReply struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Term        int64 `protobuf:"varint,1,opt,name=term,proto3" json:"term,omitempty"`
	CommitIndex int64 `protobuf:"varint,2,opt,name=commitIndex,proto3" json:"commitIndex,omitempty"`
}

func (x *GetCommitIndexReply) Reset() {
	*x = GetCommitIndexReply{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpc_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetCommitIndexReply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetCommitIndexReply) ProtoMessage() {}

func (x *GetCommitIndexReply) ProtoReflect() protoreflect.Message {
	mi := &file_rpc_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetCommitIndexReply.ProtoReflect.Descriptor instead.
func (*GetCommitIndexReply) Descriptor() ([]byte, []int) {
	return file_rpc_proto_rawDescGZIP(), []int{1}
}

func (x *GetCommitIndexReply) GetTerm() int64 {
	if x != nil {
		return x.Term
	}
	return 0
}

func (x *GetCommitIndexReply) GetCommitIndex() int64 {
	if x != nil {
		return x.CommitIndex
	}
	return 0
}

type RequestVoteArgs struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Term         int64 `protobuf:"varint,1,opt,name=term,proto3" json:"term,omitempty"`
	CandidateId  int32 `protobuf:"varint,2,opt,name=candidateId,proto3" json:"candidateId,omitempty"`
	LastLogIndex int64 `protobuf:"varint,3,opt,name=lastLogIndex,proto3" json:"lastLogIndex,omitempty"`
	LastLogTerm  int64 `protobuf:"varint,4,opt,name=lastLogTerm,proto3" json:"lastLogTerm,omitempty"`
}

func (x *RequestVoteArgs) Reset() {
	*x = RequestVoteArgs{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpc_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RequestVoteArgs) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RequestVoteArgs) ProtoMessage() {}

func (x *RequestVoteArgs) ProtoReflect() protoreflect.Message {
	mi := &file_rpc_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RequestVoteArgs.ProtoReflect.Descriptor instead.
func (*RequestVoteArgs) Descriptor() ([]byte, []int) {
	return file_rpc_proto_rawDescGZIP(), []int{2}
}

func (x *RequestVoteArgs) GetTerm() int64 {
	if x != nil {
		return x.Term
	}
	return 0
}

func (x *RequestVoteArgs) GetCandidateId() int32 {
	if x != nil {
		return x.CandidateId
	}
	return 0
}

func (x *RequestVoteArgs) GetLastLogIndex() int64 {
	if x != nil {
		return x.LastLogIndex
	}
	return 0
}

func (x *RequestVoteArgs) GetLastLogTerm() int64 {
	if x != nil {
		return x.LastLogTerm
	}
	return 0
}

type RequestVoteReply struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Term        int64 `protobuf:"varint,1,opt,name=term,proto3" json:"term,omitempty"`
	VoteGranted bool  `protobuf:"varint,2,opt,name=voteGranted,proto3" json:"voteGranted,omitempty"`
}

func (x *RequestVoteReply) Reset() {
	*x = RequestVoteReply{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpc_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RequestVoteReply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RequestVoteReply) ProtoMessage() {}

func (x *RequestVoteReply) ProtoReflect() protoreflect.Message {
	mi := &file_rpc_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RequestVoteReply.ProtoReflect.Descriptor instead.
func (*RequestVoteReply) Descriptor() ([]byte, []int) {
	return file_rpc_proto_rawDescGZIP(), []int{3}
}

func (x *RequestVoteReply) GetTerm() int64 {
	if x != nil {
		return x.Term
	}
	return 0
}

func (x *RequestVoteReply) GetVoteGranted() bool {
	if x != nil {
		return x.VoteGranted
	}
	return false
}

type Op struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Optype  uint32 `protobuf:"varint,1,opt,name=optype,proto3" json:"optype,omitempty"`
	IncrId  uint64 `protobuf:"varint,2,opt,name=incrId,proto3" json:"incrId,omitempty"`
	ClerkId int32  `protobuf:"varint,3,opt,name=clerkId,proto3" json:"clerkId,omitempty"`
	Key     string `protobuf:"bytes,4,opt,name=key,proto3" json:"key,omitempty"`
	Value   string `protobuf:"bytes,5,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *Op) Reset() {
	*x = Op{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpc_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Op) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Op) ProtoMessage() {}

func (x *Op) ProtoReflect() protoreflect.Message {
	mi := &file_rpc_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Op.ProtoReflect.Descriptor instead.
func (*Op) Descriptor() ([]byte, []int) {
	return file_rpc_proto_rawDescGZIP(), []int{4}
}

func (x *Op) GetOptype() uint32 {
	if x != nil {
		return x.Optype
	}
	return 0
}

func (x *Op) GetIncrId() uint64 {
	if x != nil {
		return x.IncrId
	}
	return 0
}

func (x *Op) GetClerkId() int32 {
	if x != nil {
		return x.ClerkId
	}
	return 0
}

func (x *Op) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

func (x *Op) GetValue() string {
	if x != nil {
		return x.Value
	}
	return ""
}

type Entry struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Term    int64 `protobuf:"varint,1,opt,name=term,proto3" json:"term,omitempty"`
	Command *Op   `protobuf:"bytes,2,opt,name=command,proto3" json:"command,omitempty"`
}

func (x *Entry) Reset() {
	*x = Entry{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpc_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Entry) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Entry) ProtoMessage() {}

func (x *Entry) ProtoReflect() protoreflect.Message {
	mi := &file_rpc_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Entry.ProtoReflect.Descriptor instead.
func (*Entry) Descriptor() ([]byte, []int) {
	return file_rpc_proto_rawDescGZIP(), []int{5}
}

func (x *Entry) GetTerm() int64 {
	if x != nil {
		return x.Term
	}
	return 0
}

func (x *Entry) GetCommand() *Op {
	if x != nil {
		return x.Command
	}
	return nil
}

type AppendEntriesArgs struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Term         int64    `protobuf:"varint,1,opt,name=term,proto3" json:"term,omitempty"`
	LeaderId     int32    `protobuf:"varint,2,opt,name=leaderId,proto3" json:"leaderId,omitempty"`
	PrevLogIndex int64    `protobuf:"varint,3,opt,name=prevLogIndex,proto3" json:"prevLogIndex,omitempty"`
	PrevLogTerm  int64    `protobuf:"varint,4,opt,name=prevLogTerm,proto3" json:"prevLogTerm,omitempty"`
	LeaderCommit int64    `protobuf:"varint,5,opt,name=leaderCommit,proto3" json:"leaderCommit,omitempty"`
	Entries      []*Entry `protobuf:"bytes,6,rep,name=entries,proto3" json:"entries,omitempty"`
}

func (x *AppendEntriesArgs) Reset() {
	*x = AppendEntriesArgs{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpc_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AppendEntriesArgs) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AppendEntriesArgs) ProtoMessage() {}

func (x *AppendEntriesArgs) ProtoReflect() protoreflect.Message {
	mi := &file_rpc_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AppendEntriesArgs.ProtoReflect.Descriptor instead.
func (*AppendEntriesArgs) Descriptor() ([]byte, []int) {
	return file_rpc_proto_rawDescGZIP(), []int{6}
}

func (x *AppendEntriesArgs) GetTerm() int64 {
	if x != nil {
		return x.Term
	}
	return 0
}

func (x *AppendEntriesArgs) GetLeaderId() int32 {
	if x != nil {
		return x.LeaderId
	}
	return 0
}

func (x *AppendEntriesArgs) GetPrevLogIndex() int64 {
	if x != nil {
		return x.PrevLogIndex
	}
	return 0
}

func (x *AppendEntriesArgs) GetPrevLogTerm() int64 {
	if x != nil {
		return x.PrevLogTerm
	}
	return 0
}

func (x *AppendEntriesArgs) GetLeaderCommit() int64 {
	if x != nil {
		return x.LeaderCommit
	}
	return 0
}

func (x *AppendEntriesArgs) GetEntries() []*Entry {
	if x != nil {
		return x.Entries
	}
	return nil
}

type AppendEntriesReply struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Term    int64 `protobuf:"varint,1,opt,name=term,proto3" json:"term,omitempty"`
	Success bool  `protobuf:"varint,2,opt,name=success,proto3" json:"success,omitempty"`
	XTerm   int64 `protobuf:"varint,3,opt,name=xTerm,proto3" json:"xTerm,omitempty"`
	XIndex  int64 `protobuf:"varint,4,opt,name=xIndex,proto3" json:"xIndex,omitempty"`
	XLen    int64 `protobuf:"varint,5,opt,name=xLen,proto3" json:"xLen,omitempty"`
}

func (x *AppendEntriesReply) Reset() {
	*x = AppendEntriesReply{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpc_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AppendEntriesReply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AppendEntriesReply) ProtoMessage() {}

func (x *AppendEntriesReply) ProtoReflect() protoreflect.Message {
	mi := &file_rpc_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AppendEntriesReply.ProtoReflect.Descriptor instead.
func (*AppendEntriesReply) Descriptor() ([]byte, []int) {
	return file_rpc_proto_rawDescGZIP(), []int{7}
}

func (x *AppendEntriesReply) GetTerm() int64 {
	if x != nil {
		return x.Term
	}
	return 0
}

func (x *AppendEntriesReply) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *AppendEntriesReply) GetXTerm() int64 {
	if x != nil {
		return x.XTerm
	}
	return 0
}

func (x *AppendEntriesReply) GetXIndex() int64 {
	if x != nil {
		return x.XIndex
	}
	return 0
}

func (x *AppendEntriesReply) GetXLen() int64 {
	if x != nil {
		return x.XLen
	}
	return 0
}

type InstallSnapshotArgs struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Term                int64  `protobuf:"varint,1,opt,name=term,proto3" json:"term,omitempty"`
	LastIncludedIndex   int64  `protobuf:"varint,2,opt,name=lastIncludedIndex,proto3" json:"lastIncludedIndex,omitempty"`
	LastIncludedTerm    int64  `protobuf:"varint,3,opt,name=lastIncludedTerm,proto3" json:"lastIncludedTerm,omitempty"`
	LeaderId            int32  `protobuf:"varint,4,opt,name=leaderId,proto3" json:"leaderId,omitempty"`
	Done                bool   `protobuf:"varint,5,opt,name=done,proto3" json:"done,omitempty"`
	LastIncludedCommand *Op    `protobuf:"bytes,6,opt,name=lastIncludedCommand,proto3" json:"lastIncludedCommand,omitempty"`
	Data                []byte `protobuf:"bytes,7,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *InstallSnapshotArgs) Reset() {
	*x = InstallSnapshotArgs{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpc_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *InstallSnapshotArgs) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InstallSnapshotArgs) ProtoMessage() {}

func (x *InstallSnapshotArgs) ProtoReflect() protoreflect.Message {
	mi := &file_rpc_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use InstallSnapshotArgs.ProtoReflect.Descriptor instead.
func (*InstallSnapshotArgs) Descriptor() ([]byte, []int) {
	return file_rpc_proto_rawDescGZIP(), []int{8}
}

func (x *InstallSnapshotArgs) GetTerm() int64 {
	if x != nil {
		return x.Term
	}
	return 0
}

func (x *InstallSnapshotArgs) GetLastIncludedIndex() int64 {
	if x != nil {
		return x.LastIncludedIndex
	}
	return 0
}

func (x *InstallSnapshotArgs) GetLastIncludedTerm() int64 {
	if x != nil {
		return x.LastIncludedTerm
	}
	return 0
}

func (x *InstallSnapshotArgs) GetLeaderId() int32 {
	if x != nil {
		return x.LeaderId
	}
	return 0
}

func (x *InstallSnapshotArgs) GetDone() bool {
	if x != nil {
		return x.Done
	}
	return false
}

func (x *InstallSnapshotArgs) GetLastIncludedCommand() *Op {
	if x != nil {
		return x.LastIncludedCommand
	}
	return nil
}

func (x *InstallSnapshotArgs) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

type InstallSnapshotReply struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Term int64 `protobuf:"varint,1,opt,name=term,proto3" json:"term,omitempty"`
}

func (x *InstallSnapshotReply) Reset() {
	*x = InstallSnapshotReply{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpc_proto_msgTypes[9]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *InstallSnapshotReply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InstallSnapshotReply) ProtoMessage() {}

func (x *InstallSnapshotReply) ProtoReflect() protoreflect.Message {
	mi := &file_rpc_proto_msgTypes[9]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use InstallSnapshotReply.ProtoReflect.Descriptor instead.
func (*InstallSnapshotReply) Descriptor() ([]byte, []int) {
	return file_rpc_proto_rawDescGZIP(), []int{9}
}

func (x *InstallSnapshotReply) GetTerm() int64 {
	if x != nil {
		return x.Term
	}
	return 0
}

var File_rpc_proto protoreflect.FileDescriptor

var file_rpc_proto_rawDesc = []byte{
	0x0a, 0x09, 0x72, 0x70, 0x63, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x14, 0x0a, 0x12, 0x47,
	0x65, 0x74, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x49, 0x6e, 0x64, 0x65, 0x78, 0x41, 0x72, 0x67,
	0x73, 0x22, 0x4b, 0x0a, 0x13, 0x47, 0x65, 0x74, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x49, 0x6e,
	0x64, 0x65, 0x78, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x65, 0x72, 0x6d,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x04, 0x74, 0x65, 0x72, 0x6d, 0x12, 0x20, 0x0a, 0x0b,
	0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x49, 0x6e, 0x64, 0x65, 0x78, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x03, 0x52, 0x0b, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x49, 0x6e, 0x64, 0x65, 0x78, 0x22, 0x8d,
	0x01, 0x0a, 0x0f, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x56, 0x6f, 0x74, 0x65, 0x41, 0x72,
	0x67, 0x73, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x65, 0x72, 0x6d, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03,
	0x52, 0x04, 0x74, 0x65, 0x72, 0x6d, 0x12, 0x20, 0x0a, 0x0b, 0x63, 0x61, 0x6e, 0x64, 0x69, 0x64,
	0x61, 0x74, 0x65, 0x49, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0b, 0x63, 0x61, 0x6e,
	0x64, 0x69, 0x64, 0x61, 0x74, 0x65, 0x49, 0x64, 0x12, 0x22, 0x0a, 0x0c, 0x6c, 0x61, 0x73, 0x74,
	0x4c, 0x6f, 0x67, 0x49, 0x6e, 0x64, 0x65, 0x78, 0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x52, 0x0c,
	0x6c, 0x61, 0x73, 0x74, 0x4c, 0x6f, 0x67, 0x49, 0x6e, 0x64, 0x65, 0x78, 0x12, 0x20, 0x0a, 0x0b,
	0x6c, 0x61, 0x73, 0x74, 0x4c, 0x6f, 0x67, 0x54, 0x65, 0x72, 0x6d, 0x18, 0x04, 0x20, 0x01, 0x28,
	0x03, 0x52, 0x0b, 0x6c, 0x61, 0x73, 0x74, 0x4c, 0x6f, 0x67, 0x54, 0x65, 0x72, 0x6d, 0x22, 0x48,
	0x0a, 0x10, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x56, 0x6f, 0x74, 0x65, 0x52, 0x65, 0x70,
	0x6c, 0x79, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x65, 0x72, 0x6d, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03,
	0x52, 0x04, 0x74, 0x65, 0x72, 0x6d, 0x12, 0x20, 0x0a, 0x0b, 0x76, 0x6f, 0x74, 0x65, 0x47, 0x72,
	0x61, 0x6e, 0x74, 0x65, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0b, 0x76, 0x6f, 0x74,
	0x65, 0x47, 0x72, 0x61, 0x6e, 0x74, 0x65, 0x64, 0x22, 0x76, 0x0a, 0x02, 0x4f, 0x70, 0x12, 0x16,
	0x0a, 0x06, 0x6f, 0x70, 0x74, 0x79, 0x70, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x06,
	0x6f, 0x70, 0x74, 0x79, 0x70, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x69, 0x6e, 0x63, 0x72, 0x49, 0x64,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x06, 0x69, 0x6e, 0x63, 0x72, 0x49, 0x64, 0x12, 0x18,
	0x0a, 0x07, 0x63, 0x6c, 0x65, 0x72, 0x6b, 0x49, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x52,
	0x07, 0x63, 0x6c, 0x65, 0x72, 0x6b, 0x49, 0x64, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18,
	0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65,
	0x22, 0x3a, 0x0a, 0x05, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x65, 0x72,
	0x6d, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x04, 0x74, 0x65, 0x72, 0x6d, 0x12, 0x1d, 0x0a,
	0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x03,
	0x2e, 0x4f, 0x70, 0x52, 0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x22, 0xcf, 0x01, 0x0a,
	0x11, 0x41, 0x70, 0x70, 0x65, 0x6e, 0x64, 0x45, 0x6e, 0x74, 0x72, 0x69, 0x65, 0x73, 0x41, 0x72,
	0x67, 0x73, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x65, 0x72, 0x6d, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03,
	0x52, 0x04, 0x74, 0x65, 0x72, 0x6d, 0x12, 0x1a, 0x0a, 0x08, 0x6c, 0x65, 0x61, 0x64, 0x65, 0x72,
	0x49, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x08, 0x6c, 0x65, 0x61, 0x64, 0x65, 0x72,
	0x49, 0x64, 0x12, 0x22, 0x0a, 0x0c, 0x70, 0x72, 0x65, 0x76, 0x4c, 0x6f, 0x67, 0x49, 0x6e, 0x64,
	0x65, 0x78, 0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x52, 0x0c, 0x70, 0x72, 0x65, 0x76, 0x4c, 0x6f,
	0x67, 0x49, 0x6e, 0x64, 0x65, 0x78, 0x12, 0x20, 0x0a, 0x0b, 0x70, 0x72, 0x65, 0x76, 0x4c, 0x6f,
	0x67, 0x54, 0x65, 0x72, 0x6d, 0x18, 0x04, 0x20, 0x01, 0x28, 0x03, 0x52, 0x0b, 0x70, 0x72, 0x65,
	0x76, 0x4c, 0x6f, 0x67, 0x54, 0x65, 0x72, 0x6d, 0x12, 0x22, 0x0a, 0x0c, 0x6c, 0x65, 0x61, 0x64,
	0x65, 0x72, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x18, 0x05, 0x20, 0x01, 0x28, 0x03, 0x52, 0x0c,
	0x6c, 0x65, 0x61, 0x64, 0x65, 0x72, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x12, 0x20, 0x0a, 0x07,
	0x65, 0x6e, 0x74, 0x72, 0x69, 0x65, 0x73, 0x18, 0x06, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x06, 0x2e,
	0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x07, 0x65, 0x6e, 0x74, 0x72, 0x69, 0x65, 0x73, 0x22, 0x84,
	0x01, 0x0a, 0x12, 0x41, 0x70, 0x70, 0x65, 0x6e, 0x64, 0x45, 0x6e, 0x74, 0x72, 0x69, 0x65, 0x73,
	0x52, 0x65, 0x70, 0x6c, 0x79, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x65, 0x72, 0x6d, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x03, 0x52, 0x04, 0x74, 0x65, 0x72, 0x6d, 0x12, 0x18, 0x0a, 0x07, 0x73, 0x75, 0x63,
	0x63, 0x65, 0x73, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07, 0x73, 0x75, 0x63, 0x63,
	0x65, 0x73, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x78, 0x54, 0x65, 0x72, 0x6d, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x03, 0x52, 0x05, 0x78, 0x54, 0x65, 0x72, 0x6d, 0x12, 0x16, 0x0a, 0x06, 0x78, 0x49, 0x6e,
	0x64, 0x65, 0x78, 0x18, 0x04, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06, 0x78, 0x49, 0x6e, 0x64, 0x65,
	0x78, 0x12, 0x12, 0x0a, 0x04, 0x78, 0x4c, 0x65, 0x6e, 0x18, 0x05, 0x20, 0x01, 0x28, 0x03, 0x52,
	0x04, 0x78, 0x4c, 0x65, 0x6e, 0x22, 0xfe, 0x01, 0x0a, 0x13, 0x49, 0x6e, 0x73, 0x74, 0x61, 0x6c,
	0x6c, 0x53, 0x6e, 0x61, 0x70, 0x73, 0x68, 0x6f, 0x74, 0x41, 0x72, 0x67, 0x73, 0x12, 0x12, 0x0a,
	0x04, 0x74, 0x65, 0x72, 0x6d, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x04, 0x74, 0x65, 0x72,
	0x6d, 0x12, 0x2c, 0x0a, 0x11, 0x6c, 0x61, 0x73, 0x74, 0x49, 0x6e, 0x63, 0x6c, 0x75, 0x64, 0x65,
	0x64, 0x49, 0x6e, 0x64, 0x65, 0x78, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x11, 0x6c, 0x61,
	0x73, 0x74, 0x49, 0x6e, 0x63, 0x6c, 0x75, 0x64, 0x65, 0x64, 0x49, 0x6e, 0x64, 0x65, 0x78, 0x12,
	0x2a, 0x0a, 0x10, 0x6c, 0x61, 0x73, 0x74, 0x49, 0x6e, 0x63, 0x6c, 0x75, 0x64, 0x65, 0x64, 0x54,
	0x65, 0x72, 0x6d, 0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x52, 0x10, 0x6c, 0x61, 0x73, 0x74, 0x49,
	0x6e, 0x63, 0x6c, 0x75, 0x64, 0x65, 0x64, 0x54, 0x65, 0x72, 0x6d, 0x12, 0x1a, 0x0a, 0x08, 0x6c,
	0x65, 0x61, 0x64, 0x65, 0x72, 0x49, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x05, 0x52, 0x08, 0x6c,
	0x65, 0x61, 0x64, 0x65, 0x72, 0x49, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x6f, 0x6e, 0x65, 0x18,
	0x05, 0x20, 0x01, 0x28, 0x08, 0x52, 0x04, 0x64, 0x6f, 0x6e, 0x65, 0x12, 0x35, 0x0a, 0x13, 0x6c,
	0x61, 0x73, 0x74, 0x49, 0x6e, 0x63, 0x6c, 0x75, 0x64, 0x65, 0x64, 0x43, 0x6f, 0x6d, 0x6d, 0x61,
	0x6e, 0x64, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x03, 0x2e, 0x4f, 0x70, 0x52, 0x13, 0x6c,
	0x61, 0x73, 0x74, 0x49, 0x6e, 0x63, 0x6c, 0x75, 0x64, 0x65, 0x64, 0x43, 0x6f, 0x6d, 0x6d, 0x61,
	0x6e, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0c,
	0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x22, 0x2a, 0x0a, 0x14, 0x49, 0x6e, 0x73, 0x74, 0x61, 0x6c,
	0x6c, 0x53, 0x6e, 0x61, 0x70, 0x73, 0x68, 0x6f, 0x74, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x12, 0x12,
	0x0a, 0x04, 0x74, 0x65, 0x72, 0x6d, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x04, 0x74, 0x65,
	0x72, 0x6d, 0x32, 0xf9, 0x01, 0x0a, 0x04, 0x52, 0x61, 0x66, 0x74, 0x12, 0x34, 0x0a, 0x0b, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x56, 0x6f, 0x74, 0x65, 0x12, 0x10, 0x2e, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x56, 0x6f, 0x74, 0x65, 0x41, 0x72, 0x67, 0x73, 0x1a, 0x11, 0x2e, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x56, 0x6f, 0x74, 0x65, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x22,
	0x00, 0x12, 0x3a, 0x0a, 0x0d, 0x41, 0x70, 0x70, 0x65, 0x6e, 0x64, 0x45, 0x6e, 0x74, 0x72, 0x69,
	0x65, 0x73, 0x12, 0x12, 0x2e, 0x41, 0x70, 0x70, 0x65, 0x6e, 0x64, 0x45, 0x6e, 0x74, 0x72, 0x69,
	0x65, 0x73, 0x41, 0x72, 0x67, 0x73, 0x1a, 0x13, 0x2e, 0x41, 0x70, 0x70, 0x65, 0x6e, 0x64, 0x45,
	0x6e, 0x74, 0x72, 0x69, 0x65, 0x73, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x22, 0x00, 0x12, 0x40, 0x0a,
	0x0f, 0x49, 0x6e, 0x73, 0x74, 0x61, 0x6c, 0x6c, 0x53, 0x6e, 0x61, 0x70, 0x73, 0x68, 0x6f, 0x74,
	0x12, 0x14, 0x2e, 0x49, 0x6e, 0x73, 0x74, 0x61, 0x6c, 0x6c, 0x53, 0x6e, 0x61, 0x70, 0x73, 0x68,
	0x6f, 0x74, 0x41, 0x72, 0x67, 0x73, 0x1a, 0x15, 0x2e, 0x49, 0x6e, 0x73, 0x74, 0x61, 0x6c, 0x6c,
	0x53, 0x6e, 0x61, 0x70, 0x73, 0x68, 0x6f, 0x74, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x22, 0x00, 0x12,
	0x3d, 0x0a, 0x0e, 0x47, 0x65, 0x74, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x49, 0x6e, 0x64, 0x65,
	0x78, 0x12, 0x13, 0x2e, 0x47, 0x65, 0x74, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x49, 0x6e, 0x64,
	0x65, 0x78, 0x41, 0x72, 0x67, 0x73, 0x1a, 0x14, 0x2e, 0x47, 0x65, 0x74, 0x43, 0x6f, 0x6d, 0x6d,
	0x69, 0x74, 0x49, 0x6e, 0x64, 0x65, 0x78, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x22, 0x00, 0x42, 0x08,
	0x5a, 0x06, 0x2e, 0x3b, 0x72, 0x72, 0x70, 0x63, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_rpc_proto_rawDescOnce sync.Once
	file_rpc_proto_rawDescData = file_rpc_proto_rawDesc
)

func file_rpc_proto_rawDescGZIP() []byte {
	file_rpc_proto_rawDescOnce.Do(func() {
		file_rpc_proto_rawDescData = protoimpl.X.CompressGZIP(file_rpc_proto_rawDescData)
	})
	return file_rpc_proto_rawDescData
}

var file_rpc_proto_msgTypes = make([]protoimpl.MessageInfo, 10)
var file_rpc_proto_goTypes = []any{
	(*GetCommitIndexArgs)(nil),   // 0: GetCommitIndexArgs
	(*GetCommitIndexReply)(nil),  // 1: GetCommitIndexReply
	(*RequestVoteArgs)(nil),      // 2: RequestVoteArgs
	(*RequestVoteReply)(nil),     // 3: RequestVoteReply
	(*Op)(nil),                   // 4: Op
	(*Entry)(nil),                // 5: Entry
	(*AppendEntriesArgs)(nil),    // 6: AppendEntriesArgs
	(*AppendEntriesReply)(nil),   // 7: AppendEntriesReply
	(*InstallSnapshotArgs)(nil),  // 8: InstallSnapshotArgs
	(*InstallSnapshotReply)(nil), // 9: InstallSnapshotReply
}
var file_rpc_proto_depIdxs = []int32{
	4, // 0: Entry.command:type_name -> Op
	5, // 1: AppendEntriesArgs.entries:type_name -> Entry
	4, // 2: InstallSnapshotArgs.lastIncludedCommand:type_name -> Op
	2, // 3: Raft.RequestVote:input_type -> RequestVoteArgs
	6, // 4: Raft.AppendEntries:input_type -> AppendEntriesArgs
	8, // 5: Raft.InstallSnapshot:input_type -> InstallSnapshotArgs
	0, // 6: Raft.GetCommitIndex:input_type -> GetCommitIndexArgs
	3, // 7: Raft.RequestVote:output_type -> RequestVoteReply
	7, // 8: Raft.AppendEntries:output_type -> AppendEntriesReply
	9, // 9: Raft.InstallSnapshot:output_type -> InstallSnapshotReply
	1, // 10: Raft.GetCommitIndex:output_type -> GetCommitIndexReply
	7, // [7:11] is the sub-list for method output_type
	3, // [3:7] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_rpc_proto_init() }
func file_rpc_proto_init() {
	if File_rpc_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_rpc_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*GetCommitIndexArgs); i {
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
		file_rpc_proto_msgTypes[1].Exporter = func(v any, i int) any {
			switch v := v.(*GetCommitIndexReply); i {
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
		file_rpc_proto_msgTypes[2].Exporter = func(v any, i int) any {
			switch v := v.(*RequestVoteArgs); i {
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
		file_rpc_proto_msgTypes[3].Exporter = func(v any, i int) any {
			switch v := v.(*RequestVoteReply); i {
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
		file_rpc_proto_msgTypes[4].Exporter = func(v any, i int) any {
			switch v := v.(*Op); i {
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
		file_rpc_proto_msgTypes[5].Exporter = func(v any, i int) any {
			switch v := v.(*Entry); i {
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
		file_rpc_proto_msgTypes[6].Exporter = func(v any, i int) any {
			switch v := v.(*AppendEntriesArgs); i {
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
		file_rpc_proto_msgTypes[7].Exporter = func(v any, i int) any {
			switch v := v.(*AppendEntriesReply); i {
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
		file_rpc_proto_msgTypes[8].Exporter = func(v any, i int) any {
			switch v := v.(*InstallSnapshotArgs); i {
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
		file_rpc_proto_msgTypes[9].Exporter = func(v any, i int) any {
			switch v := v.(*InstallSnapshotReply); i {
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
			RawDescriptor: file_rpc_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   10,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_rpc_proto_goTypes,
		DependencyIndexes: file_rpc_proto_depIdxs,
		MessageInfos:      file_rpc_proto_msgTypes,
	}.Build()
	File_rpc_proto = out.File
	file_rpc_proto_rawDesc = nil
	file_rpc_proto_goTypes = nil
	file_rpc_proto_depIdxs = nil
}
