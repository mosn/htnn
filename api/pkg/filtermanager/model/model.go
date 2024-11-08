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

package model

// This package puts filtermanager relative definitions that use across the internal packages.
// It's not a part of the API, so it's not recommended to use it in plugin code.

import (
	"sync"
	"time"

	"mosn.io/htnn/api/pkg/filtermanager/api"
)

type FilterConfig struct {
	Name   string      `json:"name,omitempty"`
	Config interface{} `json:"config,omitempty"`
}

type ParsedFilterConfig struct {
	Name         string
	ParsedConfig interface{}
	InitOnce     sync.Once
	InitFailure  error
	Factory      api.FilterFactory
}

type FilterWrapper struct {
	api.Filter
	Name string
}

func NewFilterWrapper(name string, f api.Filter) *FilterWrapper {
	return &FilterWrapper{
		Filter: f,
		Name:   name,
	}
}

type executionRecord struct {
	name     string
	duration time.Duration
}

type ExecutionRecords struct {
	records []*executionRecord
}

func NewExecutionRecords() *ExecutionRecords {
	return &ExecutionRecords{
		records: make([]*executionRecord, 0, 8),
	}
}

// Record & ForEach should only be called in OnLog phase

func (e *ExecutionRecords) Record(name string, duration time.Duration) {
	e.records = append(e.records, &executionRecord{
		name:     name,
		duration: duration,
	})
}

func (e *ExecutionRecords) ForEach(f func(name string, duration time.Duration)) {
	for _, record := range e.records {
		f(record.name, record.duration)
	}
}
