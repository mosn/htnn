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
