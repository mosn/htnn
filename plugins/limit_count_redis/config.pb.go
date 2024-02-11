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
// source: plugins/limit_count_redis/config.proto

package limit_count_redis

import (
	reflect "reflect"
	sync "sync"

	_ "github.com/envoyproxy/protoc-gen-validate/validate"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	durationpb "google.golang.org/protobuf/types/known/durationpb"

	v1 "mosn.io/htnn/api/v1"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Rule struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	TimeWindow *durationpb.Duration `protobuf:"bytes,1,opt,name=time_window,json=timeWindow,proto3" json:"time_window,omitempty"`
	Count      uint32               `protobuf:"varint,2,opt,name=count,proto3" json:"count,omitempty"`
	Key        string               `protobuf:"bytes,3,opt,name=key,proto3" json:"key,omitempty"`
}

func (x *Rule) Reset() {
	*x = Rule{}
	if protoimpl.UnsafeEnabled {
		mi := &file_plugins_limit_count_redis_config_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Rule) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Rule) ProtoMessage() {}

func (x *Rule) ProtoReflect() protoreflect.Message {
	mi := &file_plugins_limit_count_redis_config_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Rule.ProtoReflect.Descriptor instead.
func (*Rule) Descriptor() ([]byte, []int) {
	return file_plugins_limit_count_redis_config_proto_rawDescGZIP(), []int{0}
}

func (x *Rule) GetTimeWindow() *durationpb.Duration {
	if x != nil {
		return x.TimeWindow
	}
	return nil
}

func (x *Rule) GetCount() uint32 {
	if x != nil {
		return x.Count
	}
	return 0
}

func (x *Rule) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

type Config struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Source:
	//
	//	*Config_Address
	Source isConfig_Source `protobuf_oneof:"source"`
	// put a max limit as the rules are sent as one lua script
	Rules                   []*Rule       `protobuf:"bytes,2,rep,name=rules,proto3" json:"rules,omitempty"`
	FailureModeDeny         bool          `protobuf:"varint,3,opt,name=failure_mode_deny,json=failureModeDeny,proto3" json:"failure_mode_deny,omitempty"`
	EnableLimitQuotaHeaders bool          `protobuf:"varint,4,opt,name=enable_limit_quota_headers,json=enableLimitQuotaHeaders,proto3" json:"enable_limit_quota_headers,omitempty"`
	Username                string        `protobuf:"bytes,5,opt,name=username,proto3" json:"username,omitempty"`
	Password                string        `protobuf:"bytes,6,opt,name=password,proto3" json:"password,omitempty"`
	Tls                     bool          `protobuf:"varint,7,opt,name=tls,proto3" json:"tls,omitempty"`
	TlsSkipVerify           bool          `protobuf:"varint,8,opt,name=tls_skip_verify,json=tlsSkipVerify,proto3" json:"tls_skip_verify,omitempty"`
	StatusOnError           v1.StatusCode `protobuf:"varint,9,opt,name=status_on_error,json=statusOnError,proto3,enum=api.v1.StatusCode" json:"status_on_error,omitempty"`
	RateLimitedStatus       v1.StatusCode `protobuf:"varint,10,opt,name=rate_limited_status,json=rateLimitedStatus,proto3,enum=api.v1.StatusCode" json:"rate_limited_status,omitempty"`
}

func (x *Config) Reset() {
	*x = Config{}
	if protoimpl.UnsafeEnabled {
		mi := &file_plugins_limit_count_redis_config_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Config) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Config) ProtoMessage() {}

func (x *Config) ProtoReflect() protoreflect.Message {
	mi := &file_plugins_limit_count_redis_config_proto_msgTypes[1]
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
	return file_plugins_limit_count_redis_config_proto_rawDescGZIP(), []int{1}
}

func (m *Config) GetSource() isConfig_Source {
	if m != nil {
		return m.Source
	}
	return nil
}

func (x *Config) GetAddress() string {
	if x, ok := x.GetSource().(*Config_Address); ok {
		return x.Address
	}
	return ""
}

func (x *Config) GetRules() []*Rule {
	if x != nil {
		return x.Rules
	}
	return nil
}

func (x *Config) GetFailureModeDeny() bool {
	if x != nil {
		return x.FailureModeDeny
	}
	return false
}

func (x *Config) GetEnableLimitQuotaHeaders() bool {
	if x != nil {
		return x.EnableLimitQuotaHeaders
	}
	return false
}

func (x *Config) GetUsername() string {
	if x != nil {
		return x.Username
	}
	return ""
}

func (x *Config) GetPassword() string {
	if x != nil {
		return x.Password
	}
	return ""
}

func (x *Config) GetTls() bool {
	if x != nil {
		return x.Tls
	}
	return false
}

func (x *Config) GetTlsSkipVerify() bool {
	if x != nil {
		return x.TlsSkipVerify
	}
	return false
}

func (x *Config) GetStatusOnError() v1.StatusCode {
	if x != nil {
		return x.StatusOnError
	}
	return v1.StatusCode(0)
}

func (x *Config) GetRateLimitedStatus() v1.StatusCode {
	if x != nil {
		return x.RateLimitedStatus
	}
	return v1.StatusCode(0)
}

type isConfig_Source interface {
	isConfig_Source()
}

type Config_Address struct {
	Address string `protobuf:"bytes,1,opt,name=address,proto3,oneof"` // TODO: support cluster
}

func (*Config_Address) isConfig_Source() {}

var File_plugins_limit_count_redis_config_proto protoreflect.FileDescriptor

var file_plugins_limit_count_redis_config_proto_rawDesc = []byte{
	0x0a, 0x26, 0x70, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x73, 0x2f, 0x6c, 0x69, 0x6d, 0x69, 0x74, 0x5f,
	0x63, 0x6f, 0x75, 0x6e, 0x74, 0x5f, 0x72, 0x65, 0x64, 0x69, 0x73, 0x2f, 0x63, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x19, 0x70, 0x6c, 0x75, 0x67, 0x69, 0x6e,
	0x73, 0x2e, 0x6c, 0x69, 0x6d, 0x69, 0x74, 0x5f, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x5f, 0x72, 0x65,
	0x64, 0x69, 0x73, 0x1a, 0x18, 0x61, 0x70, 0x69, 0x2f, 0x76, 0x31, 0x2f, 0x68, 0x74, 0x74, 0x70,
	0x5f, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1e, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x64,
	0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x17, 0x76,
	0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x81, 0x01, 0x0a, 0x04, 0x52, 0x75, 0x6c, 0x65, 0x12,
	0x48, 0x0a, 0x0b, 0x74, 0x69, 0x6d, 0x65, 0x5f, 0x77, 0x69, 0x6e, 0x64, 0x6f, 0x77, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x42,
	0x0c, 0xfa, 0x42, 0x09, 0xaa, 0x01, 0x06, 0x08, 0x01, 0x32, 0x02, 0x08, 0x01, 0x52, 0x0a, 0x74,
	0x69, 0x6d, 0x65, 0x57, 0x69, 0x6e, 0x64, 0x6f, 0x77, 0x12, 0x1d, 0x0a, 0x05, 0x63, 0x6f, 0x75,
	0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d, 0x42, 0x07, 0xfa, 0x42, 0x04, 0x2a, 0x02, 0x28,
	0x01, 0x52, 0x05, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x22, 0xd1, 0x03, 0x0a, 0x06, 0x43,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x1a, 0x0a, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73,
	0x73, 0x12, 0x41, 0x0a, 0x05, 0x72, 0x75, 0x6c, 0x65, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x1f, 0x2e, 0x70, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x73, 0x2e, 0x6c, 0x69, 0x6d, 0x69, 0x74,
	0x5f, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x5f, 0x72, 0x65, 0x64, 0x69, 0x73, 0x2e, 0x52, 0x75, 0x6c,
	0x65, 0x42, 0x0a, 0xfa, 0x42, 0x07, 0x92, 0x01, 0x04, 0x08, 0x01, 0x10, 0x08, 0x52, 0x05, 0x72,
	0x75, 0x6c, 0x65, 0x73, 0x12, 0x2a, 0x0a, 0x11, 0x66, 0x61, 0x69, 0x6c, 0x75, 0x72, 0x65, 0x5f,
	0x6d, 0x6f, 0x64, 0x65, 0x5f, 0x64, 0x65, 0x6e, 0x79, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52,
	0x0f, 0x66, 0x61, 0x69, 0x6c, 0x75, 0x72, 0x65, 0x4d, 0x6f, 0x64, 0x65, 0x44, 0x65, 0x6e, 0x79,
	0x12, 0x3b, 0x0a, 0x1a, 0x65, 0x6e, 0x61, 0x62, 0x6c, 0x65, 0x5f, 0x6c, 0x69, 0x6d, 0x69, 0x74,
	0x5f, 0x71, 0x75, 0x6f, 0x74, 0x61, 0x5f, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x08, 0x52, 0x17, 0x65, 0x6e, 0x61, 0x62, 0x6c, 0x65, 0x4c, 0x69, 0x6d, 0x69,
	0x74, 0x51, 0x75, 0x6f, 0x74, 0x61, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x12, 0x1a, 0x0a,
	0x08, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x08, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x70, 0x61, 0x73,
	0x73, 0x77, 0x6f, 0x72, 0x64, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x70, 0x61, 0x73,
	0x73, 0x77, 0x6f, 0x72, 0x64, 0x12, 0x10, 0x0a, 0x03, 0x74, 0x6c, 0x73, 0x18, 0x07, 0x20, 0x01,
	0x28, 0x08, 0x52, 0x03, 0x74, 0x6c, 0x73, 0x12, 0x26, 0x0a, 0x0f, 0x74, 0x6c, 0x73, 0x5f, 0x73,
	0x6b, 0x69, 0x70, 0x5f, 0x76, 0x65, 0x72, 0x69, 0x66, 0x79, 0x18, 0x08, 0x20, 0x01, 0x28, 0x08,
	0x52, 0x0d, 0x74, 0x6c, 0x73, 0x53, 0x6b, 0x69, 0x70, 0x56, 0x65, 0x72, 0x69, 0x66, 0x79, 0x12,
	0x3a, 0x0a, 0x0f, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x5f, 0x6f, 0x6e, 0x5f, 0x65, 0x72, 0x72,
	0x6f, 0x72, 0x18, 0x09, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x12, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76,
	0x31, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x43, 0x6f, 0x64, 0x65, 0x52, 0x0d, 0x73, 0x74,
	0x61, 0x74, 0x75, 0x73, 0x4f, 0x6e, 0x45, 0x72, 0x72, 0x6f, 0x72, 0x12, 0x42, 0x0a, 0x13, 0x72,
	0x61, 0x74, 0x65, 0x5f, 0x6c, 0x69, 0x6d, 0x69, 0x74, 0x65, 0x64, 0x5f, 0x73, 0x74, 0x61, 0x74,
	0x75, 0x73, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x12, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76,
	0x31, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x43, 0x6f, 0x64, 0x65, 0x52, 0x11, 0x72, 0x61,
	0x74, 0x65, 0x4c, 0x69, 0x6d, 0x69, 0x74, 0x65, 0x64, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x42,
	0x0d, 0x0a, 0x06, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x12, 0x03, 0xf8, 0x42, 0x01, 0x42, 0x28,
	0x5a, 0x26, 0x6d, 0x6f, 0x73, 0x6e, 0x2e, 0x69, 0x6f, 0x2f, 0x68, 0x74, 0x6e, 0x6e, 0x2f, 0x70,
	0x6c, 0x75, 0x67, 0x69, 0x6e, 0x73, 0x2f, 0x6c, 0x69, 0x6d, 0x69, 0x74, 0x5f, 0x63, 0x6f, 0x75,
	0x6e, 0x74, 0x5f, 0x72, 0x65, 0x64, 0x69, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_plugins_limit_count_redis_config_proto_rawDescOnce sync.Once
	file_plugins_limit_count_redis_config_proto_rawDescData = file_plugins_limit_count_redis_config_proto_rawDesc
)

func file_plugins_limit_count_redis_config_proto_rawDescGZIP() []byte {
	file_plugins_limit_count_redis_config_proto_rawDescOnce.Do(func() {
		file_plugins_limit_count_redis_config_proto_rawDescData = protoimpl.X.CompressGZIP(file_plugins_limit_count_redis_config_proto_rawDescData)
	})
	return file_plugins_limit_count_redis_config_proto_rawDescData
}

var file_plugins_limit_count_redis_config_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_plugins_limit_count_redis_config_proto_goTypes = []interface{}{
	(*Rule)(nil),                // 0: plugins.limit_count_redis.Rule
	(*Config)(nil),              // 1: plugins.limit_count_redis.Config
	(*durationpb.Duration)(nil), // 2: google.protobuf.Duration
	(v1.StatusCode)(0),          // 3: api.v1.StatusCode
}
var file_plugins_limit_count_redis_config_proto_depIdxs = []int32{
	2, // 0: plugins.limit_count_redis.Rule.time_window:type_name -> google.protobuf.Duration
	0, // 1: plugins.limit_count_redis.Config.rules:type_name -> plugins.limit_count_redis.Rule
	3, // 2: plugins.limit_count_redis.Config.status_on_error:type_name -> api.v1.StatusCode
	3, // 3: plugins.limit_count_redis.Config.rate_limited_status:type_name -> api.v1.StatusCode
	4, // [4:4] is the sub-list for method output_type
	4, // [4:4] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_plugins_limit_count_redis_config_proto_init() }
func file_plugins_limit_count_redis_config_proto_init() {
	if File_plugins_limit_count_redis_config_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_plugins_limit_count_redis_config_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Rule); i {
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
		file_plugins_limit_count_redis_config_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
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
	file_plugins_limit_count_redis_config_proto_msgTypes[1].OneofWrappers = []interface{}{
		(*Config_Address)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_plugins_limit_count_redis_config_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_plugins_limit_count_redis_config_proto_goTypes,
		DependencyIndexes: file_plugins_limit_count_redis_config_proto_depIdxs,
		MessageInfos:      file_plugins_limit_count_redis_config_proto_msgTypes,
	}.Build()
	File_plugins_limit_count_redis_config_proto = out.File
	file_plugins_limit_count_redis_config_proto_rawDesc = nil
	file_plugins_limit_count_redis_config_proto_goTypes = nil
	file_plugins_limit_count_redis_config_proto_depIdxs = nil
}
