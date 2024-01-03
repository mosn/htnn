// Copyright The HTNN Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v4.24.4
// source: pkg/plugins/config.proto

package plugins

import (
	reflect "reflect"
	sync "sync"

	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Config struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Pet string `protobuf:"bytes,1,opt,name=pet,proto3" json:"pet,omitempty"`
}

func (x *Config) Reset() {
	*x = Config{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_plugins_config_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Config) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Config) ProtoMessage() {}

func (x *Config) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_plugins_config_proto_msgTypes[0]
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
	return file_pkg_plugins_config_proto_rawDescGZIP(), []int{0}
}

func (x *Config) GetPet() string {
	if x != nil {
		return x.Pet
	}
	return ""
}

var File_pkg_plugins_config_proto protoreflect.FileDescriptor

var file_pkg_plugins_config_proto_rawDesc = []byte{
	0x0a, 0x18, 0x70, 0x6b, 0x67, 0x2f, 0x70, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x73, 0x2f, 0x63, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0b, 0x70, 0x6b, 0x67, 0x2e,
	0x70, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x73, 0x22, 0x1a, 0x0a, 0x06, 0x43, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x12, 0x10, 0x0a, 0x03, 0x70, 0x65, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03,
	0x70, 0x65, 0x74, 0x42, 0x1a, 0x5a, 0x18, 0x6d, 0x6f, 0x73, 0x6e, 0x2e, 0x69, 0x6f, 0x2f, 0x68,
	0x74, 0x6e, 0x6e, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x70, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x73, 0x62,
	0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_pkg_plugins_config_proto_rawDescOnce sync.Once
	file_pkg_plugins_config_proto_rawDescData = file_pkg_plugins_config_proto_rawDesc
)

func file_pkg_plugins_config_proto_rawDescGZIP() []byte {
	file_pkg_plugins_config_proto_rawDescOnce.Do(func() {
		file_pkg_plugins_config_proto_rawDescData = protoimpl.X.CompressGZIP(file_pkg_plugins_config_proto_rawDescData)
	})
	return file_pkg_plugins_config_proto_rawDescData
}

var file_pkg_plugins_config_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_pkg_plugins_config_proto_goTypes = []interface{}{
	(*Config)(nil), // 0: pkg.plugins.Config
}
var file_pkg_plugins_config_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_pkg_plugins_config_proto_init() }
func file_pkg_plugins_config_proto_init() {
	if File_pkg_plugins_config_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_pkg_plugins_config_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
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
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_pkg_plugins_config_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_pkg_plugins_config_proto_goTypes,
		DependencyIndexes: file_pkg_plugins_config_proto_depIdxs,
		MessageInfos:      file_pkg_plugins_config_proto_msgTypes,
	}.Build()
	File_pkg_plugins_config_proto = out.File
	file_pkg_plugins_config_proto_rawDesc = nil
	file_pkg_plugins_config_proto_goTypes = nil
	file_pkg_plugins_config_proto_depIdxs = nil
}
