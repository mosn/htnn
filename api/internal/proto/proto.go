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

package proto

import (
	"errors"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"mosn.io/htnn/api/pkg/log"
)

var (
	logger = log.DefaultLogger.WithName("proto")
)

// MessageToAnyXX below are copied from istio, under Apache License

func MessageToAnyWithError(msg proto.Message) (*anypb.Any, error) {
	if msg == nil {
		return nil, errors.New("nil message")
	}
	b, err := proto.MarshalOptions{Deterministic: true}.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return &anypb.Any{
		TypeUrl: "type.googleapis.com/" + string(msg.ProtoReflect().Descriptor().FullName()),
		Value:   b,
	}, nil
}

// MessageToAny converts from proto message to proto Any
func MessageToAny(msg proto.Message) *anypb.Any {
	out, err := MessageToAnyWithError(msg)
	if err != nil {
		logger.Error(err, fmt.Sprintf("error marshaling Any %s", prototext.Format(msg)))
		return nil
	}
	return out
}

// UnmarshalJSON should be used instead of protojson.Unmarshal. The first one contains custom options.
func UnmarshalJSON(data []byte, conf proto.Message) error {
	// Don't throw an error when there is unknown field. Therefore, we can rollback to previous
	// version quickly under the same configurations, and make A/B test easier.
	return protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(data, conf)
}
