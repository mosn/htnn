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

package api

import (
	"fmt"
	"strings"
)

type Phase int

const (
	PhaseDecodeHeaders  Phase = 0x01
	PhaseDecodeData     Phase = 0x02
	PhaseDecodeTrailers Phase = 0x04
	PhaseDecodeRequest  Phase = 0x08
	PhaseEncodeHeaders  Phase = 0x10
	PhaseEncodeData     Phase = 0x20
	PhaseEncodeTrailers Phase = 0x40
	PhaseEncodeResponse Phase = 0x80
	PhaseOnLog          Phase = 0x100
)

var (
	AllPhases = PhaseDecodeHeaders | PhaseDecodeData | PhaseDecodeTrailers | PhaseDecodeRequest |
		PhaseEncodeHeaders | PhaseEncodeData | PhaseEncodeTrailers | PhaseEncodeResponse | PhaseOnLog
)

func (p Phase) Contains(phases Phase) bool {
	return p&phases == phases
}

func (p Phase) String() string {
	var names []string

	if p&PhaseDecodeHeaders != 0 {
		names = append(names, "PhaseDecodeHeaders")
	}
	if p&PhaseDecodeData != 0 {
		names = append(names, "PhaseDecodeData")
	}
	if p&PhaseDecodeTrailers != 0 {
		names = append(names, "PhaseDecodeTrailers")
	}
	if p&PhaseDecodeRequest != 0 {
		names = append(names, "PhaseDecodeRequest")
	}
	if p&PhaseEncodeHeaders != 0 {
		names = append(names, "PhaseEncodeHeaders")
	}
	if p&PhaseEncodeData != 0 {
		names = append(names, "PhaseEncodeData")
	}
	if p&PhaseEncodeTrailers != 0 {
		names = append(names, "PhaseEncodeTrailers")
	}
	if p&PhaseEncodeResponse != 0 {
		names = append(names, "PhaseEncodeResponse")
	}
	if p&PhaseOnLog != 0 {
		names = append(names, "PhaseOnLog")
	}

	if len(names) == 0 {
		return fmt.Sprintf("Phase(%d)", p)
	}
	return strings.Join(names, " | ")
}

func MethodToPhase(meth string) Phase {
	switch meth {
	case "DecodeHeaders":
		return PhaseDecodeHeaders
	case "DecodeData":
		return PhaseDecodeData
	case "DecodeTrailers":
		return PhaseDecodeTrailers
	case "DecodeRequest":
		return PhaseDecodeRequest
	case "EncodeHeaders":
		return PhaseEncodeHeaders
	case "EncodeData":
		return PhaseEncodeData
	case "EncodeTrailers":
		return PhaseEncodeTrailers
	case "EncodeResponse":
		return PhaseEncodeResponse
	case "OnLog":
		return PhaseOnLog
	default:
		return 0
	}
}

func NewAllMethodsMap() map[string]bool {
	return map[string]bool{
		"DecodeHeaders":  true,
		"DecodeData":     true,
		"DecodeRequest":  true,
		"DecodeTrailers": true,
		"EncodeHeaders":  true,
		"EncodeData":     true,
		"EncodeResponse": true,
		"EncodeTrailers": true,
		"OnLog":          true,
	}
}
