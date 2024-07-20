// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        v3.12.4
// source: snapshot.proto

package persister

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

// Snapshot 结构定义
type Snapshot struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	RowData   []byte `protobuf:"bytes,1,opt,name=rowData,proto3" json:"rowData,omitempty"`
	LastIndex int64  `protobuf:"varint,2,opt,name=lastIndex,proto3" json:"lastIndex,omitempty"`
	LastTerm  int64  `protobuf:"varint,3,opt,name=lastTerm,proto3" json:"lastTerm,omitempty"`
}

func (x *Snapshot) Reset() {
	*x = Snapshot{}
	if protoimpl.UnsafeEnabled {
		mi := &file_snapshot_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Snapshot) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Snapshot) ProtoMessage() {}

func (x *Snapshot) ProtoReflect() protoreflect.Message {
	mi := &file_snapshot_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Snapshot.ProtoReflect.Descriptor instead.
func (*Snapshot) Descriptor() ([]byte, []int) {
	return file_snapshot_proto_rawDescGZIP(), []int{0}
}

func (x *Snapshot) GetRowData() []byte {
	if x != nil {
		return x.RowData
	}
	return nil
}

func (x *Snapshot) GetLastIndex() int64 {
	if x != nil {
		return x.LastIndex
	}
	return 0
}

func (x *Snapshot) GetLastTerm() int64 {
	if x != nil {
		return x.LastTerm
	}
	return 0
}

type State struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	VotedFor    int32 `protobuf:"varint,1,opt,name=votedFor,proto3" json:"votedFor,omitempty"`
	CurrentTerm int64 `protobuf:"varint,2,opt,name=currentTerm,proto3" json:"currentTerm,omitempty"`
}

func (x *State) Reset() {
	*x = State{}
	if protoimpl.UnsafeEnabled {
		mi := &file_snapshot_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *State) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*State) ProtoMessage() {}

func (x *State) ProtoReflect() protoreflect.Message {
	mi := &file_snapshot_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use State.ProtoReflect.Descriptor instead.
func (*State) Descriptor() ([]byte, []int) {
	return file_snapshot_proto_rawDescGZIP(), []int{1}
}

func (x *State) GetVotedFor() int32 {
	if x != nil {
		return x.VotedFor
	}
	return 0
}

func (x *State) GetCurrentTerm() int64 {
	if x != nil {
		return x.CurrentTerm
	}
	return 0
}

var File_snapshot_proto protoreflect.FileDescriptor

var file_snapshot_proto_rawDesc = []byte{
	0x0a, 0x0e, 0x73, 0x6e, 0x61, 0x70, 0x73, 0x68, 0x6f, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x22, 0x5e, 0x0a, 0x08, 0x53, 0x6e, 0x61, 0x70, 0x73, 0x68, 0x6f, 0x74, 0x12, 0x18, 0x0a, 0x07,
	0x72, 0x6f, 0x77, 0x44, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x72,
	0x6f, 0x77, 0x44, 0x61, 0x74, 0x61, 0x12, 0x1c, 0x0a, 0x09, 0x6c, 0x61, 0x73, 0x74, 0x49, 0x6e,
	0x64, 0x65, 0x78, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x09, 0x6c, 0x61, 0x73, 0x74, 0x49,
	0x6e, 0x64, 0x65, 0x78, 0x12, 0x1a, 0x0a, 0x08, 0x6c, 0x61, 0x73, 0x74, 0x54, 0x65, 0x72, 0x6d,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x52, 0x08, 0x6c, 0x61, 0x73, 0x74, 0x54, 0x65, 0x72, 0x6d,
	0x22, 0x45, 0x0a, 0x05, 0x53, 0x74, 0x61, 0x74, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x76, 0x6f, 0x74,
	0x65, 0x64, 0x46, 0x6f, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x08, 0x76, 0x6f, 0x74,
	0x65, 0x64, 0x46, 0x6f, 0x72, 0x12, 0x20, 0x0a, 0x0b, 0x63, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x74,
	0x54, 0x65, 0x72, 0x6d, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x0b, 0x63, 0x75, 0x72, 0x72,
	0x65, 0x6e, 0x74, 0x54, 0x65, 0x72, 0x6d, 0x42, 0x0e, 0x5a, 0x0c, 0x2e, 0x2f, 0x3b, 0x70, 0x65,
	0x72, 0x73, 0x69, 0x73, 0x74, 0x65, 0x72, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_snapshot_proto_rawDescOnce sync.Once
	file_snapshot_proto_rawDescData = file_snapshot_proto_rawDesc
)

func file_snapshot_proto_rawDescGZIP() []byte {
	file_snapshot_proto_rawDescOnce.Do(func() {
		file_snapshot_proto_rawDescData = protoimpl.X.CompressGZIP(file_snapshot_proto_rawDescData)
	})
	return file_snapshot_proto_rawDescData
}

var file_snapshot_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_snapshot_proto_goTypes = []any{
	(*Snapshot)(nil), // 0: Snapshot
	(*State)(nil),    // 1: State
}
var file_snapshot_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_snapshot_proto_init() }
func file_snapshot_proto_init() {
	if File_snapshot_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_snapshot_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*Snapshot); i {
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
		file_snapshot_proto_msgTypes[1].Exporter = func(v any, i int) any {
			switch v := v.(*State); i {
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
			RawDescriptor: file_snapshot_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_snapshot_proto_goTypes,
		DependencyIndexes: file_snapshot_proto_depIdxs,
		MessageInfos:      file_snapshot_proto_msgTypes,
	}.Build()
	File_snapshot_proto = out.File
	file_snapshot_proto_rawDesc = nil
	file_snapshot_proto_goTypes = nil
	file_snapshot_proto_depIdxs = nil
}
