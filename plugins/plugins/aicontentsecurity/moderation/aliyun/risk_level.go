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

package aliyun

import (
	"fmt"
)

type RiskLevel int

const (
	None RiskLevel = iota
	Low
	Medium
	High
)

var (
	riskLevelStrings = []string{"none", "low", "medium", "high"}
	riskLevelMap     = map[string]RiskLevel{
		"none":   None,
		"low":    Low,
		"medium": Medium,
		"high":   High,
	}
)

// String For fmt.Stringer
func (r RiskLevel) String() string {
	if r < None || r > High {
		return "unknown"
	}
	return riskLevelStrings[r]
}

func ParseRiskLevel(s string) (RiskLevel, error) {
	level, ok := riskLevelMap[s]
	if !ok {
		return None, fmt.Errorf("invalid risk level: %q", s)
	}
	return level, nil
}

func (r RiskLevel) MarshalJSON() ([]byte, error) {
	return []byte(`"` + r.String() + `"`), nil
}

func (r *RiskLevel) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*r = None
		return nil
	}
	str := string(data[1 : len(data)-1])
	parsedLevel, err := ParseRiskLevel(str)
	if err != nil {
		return err
	}
	*r = parsedLevel
	return nil
}
