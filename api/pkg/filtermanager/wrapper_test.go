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

package filtermanager

import (
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/filtermanager/model"
	"mosn.io/htnn/api/plugins/tests/pkg/envoy"
)

func TestDebugFilter(t *testing.T) {
	cb := envoy.NewFilterCallbackHandler()
	raw1 := &api.PassThroughFilter{}
	f1 := NewDebugFilter("one", raw1, cb)
	raw2 := &api.PassThroughFilter{}
	f2 := NewDebugFilter("two", raw2, cb)

	f2.DecodeHeaders(nil, true)
	f1.DecodeHeaders(nil, true)

	type RecordWrapper struct {
		PluginName string
		Record     time.Duration
	}

	getRecords := func(executionRecords *model.ExecutionRecords) []RecordWrapper {
		records := []RecordWrapper{}
		executionRecords.ForEach(func(name string, duration time.Duration) {
			r := RecordWrapper{
				PluginName: name,
				Record:     duration,
			}
			records = append(records, r)
		})
		return records
	}
	executionRecords := model.NewExecutionRecords()
	name, duration := f2.(*debugFilter).reportExecution()
	executionRecords.Record(name, duration)
	name, duration = f1.(*debugFilter).reportExecution()
	executionRecords.Record(name, duration)
	records := getRecords(executionRecords)
	t.Logf("get records %+v\n", records) // for debug when test failed
	assert.Equal(t, "two", records[0].PluginName)
	assert.True(t, records[0].Record > 0)
	assert.Equal(t, "one", records[1].PluginName)
	assert.True(t, records[1].Record > 0)
	decodeHeadersCost := records[1].Record

	patches := gomonkey.ApplyMethodFunc(raw1, "DecodeData", func(data api.BufferInstance, endStream bool) api.ResultAction {
		time.Sleep(100 * time.Millisecond)
		return api.Continue

	})
	patches.ApplyMethodFunc(raw1, "EncodeHeaders", func(headers api.ResponseHeaderMap, endStream bool) api.ResultAction {
		time.Sleep(50 * time.Millisecond)
		return api.Continue
	})
	patches.ApplyMethodFunc(raw1, "EncodeData", func(data api.BufferInstance, endStream bool) api.ResultAction {
		time.Sleep(20 * time.Millisecond)
		return api.Continue
	})
	defer patches.Reset()
	f1.DecodeData(nil, false)
	f1.DecodeData(nil, true)
	f1.EncodeHeaders(nil, false)
	f1.EncodeData(nil, true)

	executionRecords = model.NewExecutionRecords()
	name, duration = f1.(*debugFilter).reportExecution()
	executionRecords.Record(name, duration)
	records = getRecords(executionRecords)
	t.Logf("get records %+v\n", records) // for debug when test failed
	// Should be the sum of multiple calls
	delta := 10 * time.Millisecond
	rec := records[0].Record - decodeHeadersCost
	assert.True(t, 270*time.Millisecond-delta < rec && rec < 270*time.Millisecond+delta, rec)
}
