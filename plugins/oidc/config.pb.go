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
// source: plugins/oidc/config.proto

package oidc

import (
	reflect "reflect"
	sync "sync"

	_ "github.com/envoyproxy/protoc-gen-validate/validate"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	durationpb "google.golang.org/protobuf/types/known/durationpb"
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

	ClientId     string `protobuf:"bytes,1,opt,name=client_id,json=clientId,proto3" json:"client_id,omitempty"`
	ClientSecret string `protobuf:"bytes,2,opt,name=client_secret,json=clientSecret,proto3" json:"client_secret,omitempty"`
	// The issuer is the URL identifier for the service. For example: "https://accounts.google.com"
	// or "https://login.salesforce.com".
	Issuer string `protobuf:"bytes,3,opt,name=issuer,proto3" json:"issuer,omitempty"`
	// The configured URL MUST exactly match one of the Redirection URI values
	// for the Client pre-registered at the OpenID Provider
	RedirectUrl string   `protobuf:"bytes,4,opt,name=redirect_url,json=redirectUrl,proto3" json:"redirect_url,omitempty"`
	Scopes      []string `protobuf:"bytes,5,rep,name=scopes,proto3" json:"scopes,omitempty"`
	// This option is provided to skip the nonce verification. It is designed for local development.
	SkipNonceVerify bool `protobuf:"varint,6,opt,name=skip_nonce_verify,json=skipNonceVerify,proto3" json:"skip_nonce_verify,omitempty"`
	// Default to "x-id-token"
	IdTokenHeader string `protobuf:"bytes,7,opt,name=id_token_header,json=idTokenHeader,proto3" json:"id_token_header,omitempty"`
	// The timeout to wait for the OIDC provider to respond. Default to 3s.
	Timeout *durationpb.Duration `protobuf:"bytes,8,opt,name=timeout,proto3" json:"timeout,omitempty"`
}

func (x *Config) Reset() {
	*x = Config{}
	if protoimpl.UnsafeEnabled {
		mi := &file_plugins_oidc_config_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Config) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Config) ProtoMessage() {}

func (x *Config) ProtoReflect() protoreflect.Message {
	mi := &file_plugins_oidc_config_proto_msgTypes[0]
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
	return file_plugins_oidc_config_proto_rawDescGZIP(), []int{0}
}

func (x *Config) GetClientId() string {
	if x != nil {
		return x.ClientId
	}
	return ""
}

func (x *Config) GetClientSecret() string {
	if x != nil {
		return x.ClientSecret
	}
	return ""
}

func (x *Config) GetIssuer() string {
	if x != nil {
		return x.Issuer
	}
	return ""
}

func (x *Config) GetRedirectUrl() string {
	if x != nil {
		return x.RedirectUrl
	}
	return ""
}

func (x *Config) GetScopes() []string {
	if x != nil {
		return x.Scopes
	}
	return nil
}

func (x *Config) GetSkipNonceVerify() bool {
	if x != nil {
		return x.SkipNonceVerify
	}
	return false
}

func (x *Config) GetIdTokenHeader() string {
	if x != nil {
		return x.IdTokenHeader
	}
	return ""
}

func (x *Config) GetTimeout() *durationpb.Duration {
	if x != nil {
		return x.Timeout
	}
	return nil
}

var File_plugins_oidc_config_proto protoreflect.FileDescriptor

var file_plugins_oidc_config_proto_rawDesc = []byte{
	0x0a, 0x19, 0x70, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x73, 0x2f, 0x6f, 0x69, 0x64, 0x63, 0x2f, 0x63,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0c, 0x70, 0x6c, 0x75,
	0x67, 0x69, 0x6e, 0x73, 0x2e, 0x6f, 0x69, 0x64, 0x63, 0x1a, 0x1e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x64, 0x75, 0x72, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x17, 0x76, 0x61, 0x6c, 0x69, 0x64,
	0x61, 0x74, 0x65, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x22, 0xd6, 0x02, 0x0a, 0x06, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x24, 0x0a,
	0x09, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x42, 0x07, 0xfa, 0x42, 0x04, 0x72, 0x02, 0x10, 0x01, 0x52, 0x08, 0x63, 0x6c, 0x69, 0x65, 0x6e,
	0x74, 0x49, 0x64, 0x12, 0x2c, 0x0a, 0x0d, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x5f, 0x73, 0x65,
	0x63, 0x72, 0x65, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x42, 0x07, 0xfa, 0x42, 0x04, 0x72,
	0x02, 0x10, 0x01, 0x52, 0x0c, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x53, 0x65, 0x63, 0x72, 0x65,
	0x74, 0x12, 0x20, 0x0a, 0x06, 0x69, 0x73, 0x73, 0x75, 0x65, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x09, 0x42, 0x08, 0xfa, 0x42, 0x05, 0x72, 0x03, 0x88, 0x01, 0x01, 0x52, 0x06, 0x69, 0x73, 0x73,
	0x75, 0x65, 0x72, 0x12, 0x2b, 0x0a, 0x0c, 0x72, 0x65, 0x64, 0x69, 0x72, 0x65, 0x63, 0x74, 0x5f,
	0x75, 0x72, 0x6c, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x42, 0x08, 0xfa, 0x42, 0x05, 0x72, 0x03,
	0x88, 0x01, 0x01, 0x52, 0x0b, 0x72, 0x65, 0x64, 0x69, 0x72, 0x65, 0x63, 0x74, 0x55, 0x72, 0x6c,
	0x12, 0x16, 0x0a, 0x06, 0x73, 0x63, 0x6f, 0x70, 0x65, 0x73, 0x18, 0x05, 0x20, 0x03, 0x28, 0x09,
	0x52, 0x06, 0x73, 0x63, 0x6f, 0x70, 0x65, 0x73, 0x12, 0x2a, 0x0a, 0x11, 0x73, 0x6b, 0x69, 0x70,
	0x5f, 0x6e, 0x6f, 0x6e, 0x63, 0x65, 0x5f, 0x76, 0x65, 0x72, 0x69, 0x66, 0x79, 0x18, 0x06, 0x20,
	0x01, 0x28, 0x08, 0x52, 0x0f, 0x73, 0x6b, 0x69, 0x70, 0x4e, 0x6f, 0x6e, 0x63, 0x65, 0x56, 0x65,
	0x72, 0x69, 0x66, 0x79, 0x12, 0x26, 0x0a, 0x0f, 0x69, 0x64, 0x5f, 0x74, 0x6f, 0x6b, 0x65, 0x6e,
	0x5f, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x18, 0x07, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x69,
	0x64, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x12, 0x3d, 0x0a, 0x07,
	0x74, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74, 0x18, 0x08, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x42, 0x08, 0xfa, 0x42, 0x05, 0xaa, 0x01, 0x02,
	0x2a, 0x00, 0x52, 0x07, 0x74, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74, 0x42, 0x1b, 0x5a, 0x19, 0x6d,
	0x6f, 0x73, 0x6e, 0x2e, 0x69, 0x6f, 0x2f, 0x68, 0x74, 0x6e, 0x6e, 0x2f, 0x70, 0x6c, 0x75, 0x67,
	0x69, 0x6e, 0x73, 0x2f, 0x6f, 0x69, 0x64, 0x63, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_plugins_oidc_config_proto_rawDescOnce sync.Once
	file_plugins_oidc_config_proto_rawDescData = file_plugins_oidc_config_proto_rawDesc
)

func file_plugins_oidc_config_proto_rawDescGZIP() []byte {
	file_plugins_oidc_config_proto_rawDescOnce.Do(func() {
		file_plugins_oidc_config_proto_rawDescData = protoimpl.X.CompressGZIP(file_plugins_oidc_config_proto_rawDescData)
	})
	return file_plugins_oidc_config_proto_rawDescData
}

var file_plugins_oidc_config_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_plugins_oidc_config_proto_goTypes = []interface{}{
	(*Config)(nil),              // 0: plugins.oidc.Config
	(*durationpb.Duration)(nil), // 1: google.protobuf.Duration
}
var file_plugins_oidc_config_proto_depIdxs = []int32{
	1, // 0: plugins.oidc.Config.timeout:type_name -> google.protobuf.Duration
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_plugins_oidc_config_proto_init() }
func file_plugins_oidc_config_proto_init() {
	if File_plugins_oidc_config_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_plugins_oidc_config_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
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
			RawDescriptor: file_plugins_oidc_config_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_plugins_oidc_config_proto_goTypes,
		DependencyIndexes: file_plugins_oidc_config_proto_depIdxs,
		MessageInfos:      file_plugins_oidc_config_proto_msgTypes,
	}.Build()
	File_plugins_oidc_config_proto = out.File
	file_plugins_oidc_config_proto_rawDesc = nil
	file_plugins_oidc_config_proto_goTypes = nil
	file_plugins_oidc_config_proto_depIdxs = nil
}
