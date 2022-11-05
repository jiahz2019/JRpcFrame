// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.20.1
// source: dynamicdiscovery.proto

package cluster

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

type DiscoveryNodeInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	NodeId            int32    `protobuf:"varint,1,opt,name=NodeId,proto3" json:"NodeId,omitempty"`
	NodeName          string   `protobuf:"bytes,2,opt,name=NodeName,proto3" json:"NodeName,omitempty"`
	ListenAddr        string   `protobuf:"bytes,3,opt,name=ListenAddr,proto3" json:"ListenAddr,omitempty"`
	MaxRpcParamLen    uint32   `protobuf:"varint,4,opt,name=MaxRpcParamLen,proto3" json:"MaxRpcParamLen,omitempty"`
	Private           bool     `protobuf:"varint,5,opt,name=Private,proto3" json:"Private,omitempty"`
	PublicServiceList []string `protobuf:"bytes,6,rep,name=PublicServiceList,proto3" json:"PublicServiceList,omitempty"`
}

func (x *DiscoveryNodeInfo) Reset() {
	*x = DiscoveryNodeInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_dynamicdiscovery_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DiscoveryNodeInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DiscoveryNodeInfo) ProtoMessage() {}

func (x *DiscoveryNodeInfo) ProtoReflect() protoreflect.Message {
	mi := &file_dynamicdiscovery_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DiscoveryNodeInfo.ProtoReflect.Descriptor instead.
func (*DiscoveryNodeInfo) Descriptor() ([]byte, []int) {
	return file_dynamicdiscovery_proto_rawDescGZIP(), []int{0}
}

func (x *DiscoveryNodeInfo) GetNodeId() int32 {
	if x != nil {
		return x.NodeId
	}
	return 0
}

func (x *DiscoveryNodeInfo) GetNodeName() string {
	if x != nil {
		return x.NodeName
	}
	return ""
}

func (x *DiscoveryNodeInfo) GetListenAddr() string {
	if x != nil {
		return x.ListenAddr
	}
	return ""
}

func (x *DiscoveryNodeInfo) GetMaxRpcParamLen() uint32 {
	if x != nil {
		return x.MaxRpcParamLen
	}
	return 0
}

func (x *DiscoveryNodeInfo) GetPrivate() bool {
	if x != nil {
		return x.Private
	}
	return false
}

func (x *DiscoveryNodeInfo) GetPublicServiceList() []string {
	if x != nil {
		return x.PublicServiceList
	}
	return nil
}

//Client->Master
type DiscoverClientReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	NodeInfo *DiscoveryNodeInfo `protobuf:"bytes,1,opt,name=nodeInfo,proto3" json:"nodeInfo,omitempty"`
}

func (x *DiscoverClientReq) Reset() {
	*x = DiscoverClientReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_dynamicdiscovery_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DiscoverClientReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DiscoverClientReq) ProtoMessage() {}

func (x *DiscoverClientReq) ProtoReflect() protoreflect.Message {
	mi := &file_dynamicdiscovery_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DiscoverClientReq.ProtoReflect.Descriptor instead.
func (*DiscoverClientReq) Descriptor() ([]byte, []int) {
	return file_dynamicdiscovery_proto_rawDescGZIP(), []int{1}
}

func (x *DiscoverClientReq) GetNodeInfo() *DiscoveryNodeInfo {
	if x != nil {
		return x.NodeInfo
	}
	return nil
}

//Master->Client
type DiscoverMasterReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	MasterNodeId int32                `protobuf:"varint,1,opt,name=MasterNodeId,proto3" json:"MasterNodeId,omitempty"`
	IsFull       bool                 `protobuf:"varint,2,opt,name=IsFull,proto3" json:"IsFull,omitempty"`
	DelNodeId    int32                `protobuf:"varint,3,opt,name=DelNodeId,proto3" json:"DelNodeId,omitempty"`
	NodeInfo     []*DiscoveryNodeInfo `protobuf:"bytes,4,rep,name=nodeInfo,proto3" json:"nodeInfo,omitempty"`
}

func (x *DiscoverMasterReq) Reset() {
	*x = DiscoverMasterReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_dynamicdiscovery_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DiscoverMasterReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DiscoverMasterReq) ProtoMessage() {}

func (x *DiscoverMasterReq) ProtoReflect() protoreflect.Message {
	mi := &file_dynamicdiscovery_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DiscoverMasterReq.ProtoReflect.Descriptor instead.
func (*DiscoverMasterReq) Descriptor() ([]byte, []int) {
	return file_dynamicdiscovery_proto_rawDescGZIP(), []int{2}
}

func (x *DiscoverMasterReq) GetMasterNodeId() int32 {
	if x != nil {
		return x.MasterNodeId
	}
	return 0
}

func (x *DiscoverMasterReq) GetIsFull() bool {
	if x != nil {
		return x.IsFull
	}
	return false
}

func (x *DiscoverMasterReq) GetDelNodeId() int32 {
	if x != nil {
		return x.DelNodeId
	}
	return 0
}

func (x *DiscoverMasterReq) GetNodeInfo() []*DiscoveryNodeInfo {
	if x != nil {
		return x.NodeInfo
	}
	return nil
}

type Empty struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *Empty) Reset() {
	*x = Empty{}
	if protoimpl.UnsafeEnabled {
		mi := &file_dynamicdiscovery_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Empty) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Empty) ProtoMessage() {}

func (x *Empty) ProtoReflect() protoreflect.Message {
	mi := &file_dynamicdiscovery_proto_msgTypes[3]
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
	return file_dynamicdiscovery_proto_rawDescGZIP(), []int{3}
}

var File_dynamicdiscovery_proto protoreflect.FileDescriptor

var file_dynamicdiscovery_proto_rawDesc = []byte{
	0x0a, 0x16, 0x64, 0x79, 0x6e, 0x61, 0x6d, 0x69, 0x63, 0x64, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x65,
	0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x07, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65,
	0x72, 0x22, 0xd7, 0x01, 0x0a, 0x11, 0x44, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x4e,
	0x6f, 0x64, 0x65, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x16, 0x0a, 0x06, 0x4e, 0x6f, 0x64, 0x65, 0x49,
	0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x06, 0x4e, 0x6f, 0x64, 0x65, 0x49, 0x64, 0x12,
	0x1a, 0x0a, 0x08, 0x4e, 0x6f, 0x64, 0x65, 0x4e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x08, 0x4e, 0x6f, 0x64, 0x65, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x1e, 0x0a, 0x0a, 0x4c,
	0x69, 0x73, 0x74, 0x65, 0x6e, 0x41, 0x64, 0x64, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x0a, 0x4c, 0x69, 0x73, 0x74, 0x65, 0x6e, 0x41, 0x64, 0x64, 0x72, 0x12, 0x26, 0x0a, 0x0e, 0x4d,
	0x61, 0x78, 0x52, 0x70, 0x63, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x4c, 0x65, 0x6e, 0x18, 0x04, 0x20,
	0x01, 0x28, 0x0d, 0x52, 0x0e, 0x4d, 0x61, 0x78, 0x52, 0x70, 0x63, 0x50, 0x61, 0x72, 0x61, 0x6d,
	0x4c, 0x65, 0x6e, 0x12, 0x18, 0x0a, 0x07, 0x50, 0x72, 0x69, 0x76, 0x61, 0x74, 0x65, 0x18, 0x05,
	0x20, 0x01, 0x28, 0x08, 0x52, 0x07, 0x50, 0x72, 0x69, 0x76, 0x61, 0x74, 0x65, 0x12, 0x2c, 0x0a,
	0x11, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x4c, 0x69,
	0x73, 0x74, 0x18, 0x06, 0x20, 0x03, 0x28, 0x09, 0x52, 0x11, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x63,
	0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x4c, 0x69, 0x73, 0x74, 0x22, 0x4b, 0x0a, 0x11, 0x44,
	0x69, 0x73, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x52, 0x65, 0x71,
	0x12, 0x36, 0x0a, 0x08, 0x6e, 0x6f, 0x64, 0x65, 0x49, 0x6e, 0x66, 0x6f, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x2e, 0x44, 0x69, 0x73,
	0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x4e, 0x6f, 0x64, 0x65, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x08,
	0x6e, 0x6f, 0x64, 0x65, 0x49, 0x6e, 0x66, 0x6f, 0x22, 0xa5, 0x01, 0x0a, 0x11, 0x44, 0x69, 0x73,
	0x63, 0x6f, 0x76, 0x65, 0x72, 0x4d, 0x61, 0x73, 0x74, 0x65, 0x72, 0x52, 0x65, 0x71, 0x12, 0x22,
	0x0a, 0x0c, 0x4d, 0x61, 0x73, 0x74, 0x65, 0x72, 0x4e, 0x6f, 0x64, 0x65, 0x49, 0x64, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x05, 0x52, 0x0c, 0x4d, 0x61, 0x73, 0x74, 0x65, 0x72, 0x4e, 0x6f, 0x64, 0x65,
	0x49, 0x64, 0x12, 0x16, 0x0a, 0x06, 0x49, 0x73, 0x46, 0x75, 0x6c, 0x6c, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x08, 0x52, 0x06, 0x49, 0x73, 0x46, 0x75, 0x6c, 0x6c, 0x12, 0x1c, 0x0a, 0x09, 0x44, 0x65,
	0x6c, 0x4e, 0x6f, 0x64, 0x65, 0x49, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x52, 0x09, 0x44,
	0x65, 0x6c, 0x4e, 0x6f, 0x64, 0x65, 0x49, 0x64, 0x12, 0x36, 0x0a, 0x08, 0x6e, 0x6f, 0x64, 0x65,
	0x49, 0x6e, 0x66, 0x6f, 0x18, 0x04, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x63, 0x6c, 0x75,
	0x73, 0x74, 0x65, 0x72, 0x2e, 0x44, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x4e, 0x6f,
	0x64, 0x65, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x08, 0x6e, 0x6f, 0x64, 0x65, 0x49, 0x6e, 0x66, 0x6f,
	0x22, 0x07, 0x0a, 0x05, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x42, 0x0b, 0x5a, 0x09, 0x2e, 0x2f, 0x63,
	0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_dynamicdiscovery_proto_rawDescOnce sync.Once
	file_dynamicdiscovery_proto_rawDescData = file_dynamicdiscovery_proto_rawDesc
)

func file_dynamicdiscovery_proto_rawDescGZIP() []byte {
	file_dynamicdiscovery_proto_rawDescOnce.Do(func() {
		file_dynamicdiscovery_proto_rawDescData = protoimpl.X.CompressGZIP(file_dynamicdiscovery_proto_rawDescData)
	})
	return file_dynamicdiscovery_proto_rawDescData
}

var file_dynamicdiscovery_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_dynamicdiscovery_proto_goTypes = []interface{}{
	(*DiscoveryNodeInfo)(nil), // 0: cluster.DiscoveryNodeInfo
	(*DiscoverClientReq)(nil), // 1: cluster.DiscoverClientReq
	(*DiscoverMasterReq)(nil), // 2: cluster.DiscoverMasterReq
	(*Empty)(nil),             // 3: cluster.Empty
}
var file_dynamicdiscovery_proto_depIdxs = []int32{
	0, // 0: cluster.DiscoverClientReq.nodeInfo:type_name -> cluster.DiscoveryNodeInfo
	0, // 1: cluster.DiscoverMasterReq.nodeInfo:type_name -> cluster.DiscoveryNodeInfo
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_dynamicdiscovery_proto_init() }
func file_dynamicdiscovery_proto_init() {
	if File_dynamicdiscovery_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_dynamicdiscovery_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DiscoveryNodeInfo); i {
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
		file_dynamicdiscovery_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DiscoverClientReq); i {
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
		file_dynamicdiscovery_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DiscoverMasterReq); i {
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
		file_dynamicdiscovery_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
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
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_dynamicdiscovery_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_dynamicdiscovery_proto_goTypes,
		DependencyIndexes: file_dynamicdiscovery_proto_depIdxs,
		MessageInfos:      file_dynamicdiscovery_proto_msgTypes,
	}.Build()
	File_dynamicdiscovery_proto = out.File
	file_dynamicdiscovery_proto_rawDesc = nil
	file_dynamicdiscovery_proto_goTypes = nil
	file_dynamicdiscovery_proto_depIdxs = nil
}