package proto

import (
	"errors"
	"fmt"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"mosn.io/moe/pkg/log"
)

var (
	logger = log.DefaultLogger.WithName("proto")
)

// Functions below are copied from istio, under Apache License

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
