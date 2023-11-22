// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v4.24.4
// source: plugins/casbin/config.proto

package casbin

import (
	reflect "reflect"
	sync "sync"

	_ "github.com/envoyproxy/protoc-gen-validate/validate"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Config_Source int32

const (
	Config_header Config_Source = 0
)

// Enum value maps for Config_Source.
var (
	Config_Source_name = map[int32]string{
		0: "header",
	}
	Config_Source_value = map[string]int32{
		"header": 0,
	}
)

func (x Config_Source) Enum() *Config_Source {
	p := new(Config_Source)
	*p = x
	return p
}

func (x Config_Source) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Config_Source) Descriptor() protoreflect.EnumDescriptor {
	return file_plugins_casbin_config_proto_enumTypes[0].Descriptor()
}

func (Config_Source) Type() protoreflect.EnumType {
	return &file_plugins_casbin_config_proto_enumTypes[0]
}

func (x Config_Source) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Config_Source.Descriptor instead.
func (Config_Source) EnumDescriptor() ([]byte, []int) {
	return file_plugins_casbin_config_proto_rawDescGZIP(), []int{0, 0}
}

type Config struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Rule  *Config_Rule  `protobuf:"bytes,1,opt,name=rule,proto3" json:"rule,omitempty"`
	Token *Config_Token `protobuf:"bytes,2,opt,name=token,proto3" json:"token,omitempty"`
}

func (x *Config) Reset() {
	*x = Config{}
	if protoimpl.UnsafeEnabled {
		mi := &file_plugins_casbin_config_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Config) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Config) ProtoMessage() {}

func (x *Config) ProtoReflect() protoreflect.Message {
	mi := &file_plugins_casbin_config_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Config.ProtoReflect.Descriptor instead.
func (*Config) Descriptor() ([]byte, []int) {
	return file_plugins_casbin_config_proto_rawDescGZIP(), []int{0}
}

func (x *Config) GetRule() *Config_Rule {
	if x != nil {
		return x.Rule
	}
	return nil
}

func (x *Config) GetToken() *Config_Token {
	if x != nil {
		return x.Token
	}
	return nil
}

type Config_Rule struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Model  string `protobuf:"bytes,1,opt,name=model,proto3" json:"model,omitempty"`
	Policy string `protobuf:"bytes,2,opt,name=policy,proto3" json:"policy,omitempty"`
}

func (x *Config_Rule) Reset() {
	*x = Config_Rule{}
	if protoimpl.UnsafeEnabled {
		mi := &file_plugins_casbin_config_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Config_Rule) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Config_Rule) ProtoMessage() {}

func (x *Config_Rule) ProtoReflect() protoreflect.Message {
	mi := &file_plugins_casbin_config_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Config_Rule.ProtoReflect.Descriptor instead.
func (*Config_Rule) Descriptor() ([]byte, []int) {
	return file_plugins_casbin_config_proto_rawDescGZIP(), []int{0, 0}
}

func (x *Config_Rule) GetModel() string {
	if x != nil {
		return x.Model
	}
	return ""
}

func (x *Config_Rule) GetPolicy() string {
	if x != nil {
		return x.Policy
	}
	return ""
}

type Config_Token struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Source Config_Source `protobuf:"varint,1,opt,name=source,proto3,enum=plugins.casbin.Config_Source" json:"source,omitempty"`
	Name   string        `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
}

func (x *Config_Token) Reset() {
	*x = Config_Token{}
	if protoimpl.UnsafeEnabled {
		mi := &file_plugins_casbin_config_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Config_Token) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Config_Token) ProtoMessage() {}

func (x *Config_Token) ProtoReflect() protoreflect.Message {
	mi := &file_plugins_casbin_config_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Config_Token.ProtoReflect.Descriptor instead.
func (*Config_Token) Descriptor() ([]byte, []int) {
	return file_plugins_casbin_config_proto_rawDescGZIP(), []int{0, 1}
}

func (x *Config_Token) GetSource() Config_Source {
	if x != nil {
		return x.Source
	}
	return Config_header
}

func (x *Config_Token) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

var File_plugins_casbin_config_proto protoreflect.FileDescriptor

var file_plugins_casbin_config_proto_rawDesc = []byte{
	0x0a, 0x1b, 0x70, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x73, 0x2f, 0x63, 0x61, 0x73, 0x62, 0x69, 0x6e,
	0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0e, 0x70,
	0x6c, 0x75, 0x67, 0x69, 0x6e, 0x73, 0x2e, 0x63, 0x61, 0x73, 0x62, 0x69, 0x6e, 0x1a, 0x17, 0x76,
	0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xc5, 0x02, 0x0a, 0x06, 0x43, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x12, 0x39, 0x0a, 0x04, 0x72, 0x75, 0x6c, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x1b, 0x2e, 0x70, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x73, 0x2e, 0x63, 0x61, 0x73, 0x62, 0x69, 0x6e,
	0x2e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x52, 0x75, 0x6c, 0x65, 0x42, 0x08, 0xfa, 0x42,
	0x05, 0x8a, 0x01, 0x02, 0x10, 0x01, 0x52, 0x04, 0x72, 0x75, 0x6c, 0x65, 0x12, 0x3c, 0x0a, 0x05,
	0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x70, 0x6c,
	0x75, 0x67, 0x69, 0x6e, 0x73, 0x2e, 0x63, 0x61, 0x73, 0x62, 0x69, 0x6e, 0x2e, 0x43, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x2e, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x42, 0x08, 0xfa, 0x42, 0x05, 0x8a, 0x01,
	0x02, 0x10, 0x01, 0x52, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x1a, 0x4c, 0x0a, 0x04, 0x52, 0x75,
	0x6c, 0x65, 0x12, 0x20, 0x0a, 0x05, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x42, 0x0a, 0xfa, 0x42, 0x07, 0x72, 0x05, 0x10, 0x01, 0xd0, 0x01, 0x00, 0x52, 0x05, 0x6d,
	0x6f, 0x64, 0x65, 0x6c, 0x12, 0x22, 0x0a, 0x06, 0x70, 0x6f, 0x6c, 0x69, 0x63, 0x79, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x42, 0x0a, 0xfa, 0x42, 0x07, 0x72, 0x05, 0x10, 0x01, 0xd0, 0x01, 0x00,
	0x52, 0x06, 0x70, 0x6f, 0x6c, 0x69, 0x63, 0x79, 0x1a, 0x5e, 0x0a, 0x05, 0x54, 0x6f, 0x6b, 0x65,
	0x6e, 0x12, 0x35, 0x0a, 0x06, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0e, 0x32, 0x1d, 0x2e, 0x70, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x73, 0x2e, 0x63, 0x61, 0x73, 0x62,
	0x69, 0x6e, 0x2e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65,
	0x52, 0x06, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x12, 0x1e, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x42, 0x0a, 0xfa, 0x42, 0x07, 0x72, 0x05, 0x10, 0x01, 0xd0,
	0x01, 0x00, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x22, 0x14, 0x0a, 0x06, 0x53, 0x6f, 0x75, 0x72,
	0x63, 0x65, 0x12, 0x0a, 0x0a, 0x06, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x10, 0x00, 0x42, 0x1c,
	0x5a, 0x1a, 0x6d, 0x6f, 0x73, 0x6e, 0x2e, 0x69, 0x6f, 0x2f, 0x6d, 0x6f, 0x65, 0x2f, 0x70, 0x6c,
	0x75, 0x67, 0x69, 0x6e, 0x73, 0x2f, 0x63, 0x61, 0x73, 0x62, 0x69, 0x6e, 0x62, 0x06, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_plugins_casbin_config_proto_rawDescOnce sync.Once
	file_plugins_casbin_config_proto_rawDescData = file_plugins_casbin_config_proto_rawDesc
)

func file_plugins_casbin_config_proto_rawDescGZIP() []byte {
	file_plugins_casbin_config_proto_rawDescOnce.Do(func() {
		file_plugins_casbin_config_proto_rawDescData = protoimpl.X.CompressGZIP(file_plugins_casbin_config_proto_rawDescData)
	})
	return file_plugins_casbin_config_proto_rawDescData
}

var file_plugins_casbin_config_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_plugins_casbin_config_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_plugins_casbin_config_proto_goTypes = []interface{}{
	(Config_Source)(0),   // 0: plugins.casbin.Config.Source
	(*Config)(nil),       // 1: plugins.casbin.Config
	(*Config_Rule)(nil),  // 2: plugins.casbin.Config.Rule
	(*Config_Token)(nil), // 3: plugins.casbin.Config.Token
}
var file_plugins_casbin_config_proto_depIdxs = []int32{
	2, // 0: plugins.casbin.Config.rule:type_name -> plugins.casbin.Config.Rule
	3, // 1: plugins.casbin.Config.token:type_name -> plugins.casbin.Config.Token
	0, // 2: plugins.casbin.Config.Token.source:type_name -> plugins.casbin.Config.Source
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_plugins_casbin_config_proto_init() }
func file_plugins_casbin_config_proto_init() {
	if File_plugins_casbin_config_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_plugins_casbin_config_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Config); i {
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
		file_plugins_casbin_config_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Config_Rule); i {
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
		file_plugins_casbin_config_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Config_Token); i {
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
			RawDescriptor: file_plugins_casbin_config_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_plugins_casbin_config_proto_goTypes,
		DependencyIndexes: file_plugins_casbin_config_proto_depIdxs,
		EnumInfos:         file_plugins_casbin_config_proto_enumTypes,
		MessageInfos:      file_plugins_casbin_config_proto_msgTypes,
	}.Build()
	File_plugins_casbin_config_proto = out.File
	file_plugins_casbin_config_proto_rawDesc = nil
	file_plugins_casbin_config_proto_goTypes = nil
	file_plugins_casbin_config_proto_depIdxs = nil
}
