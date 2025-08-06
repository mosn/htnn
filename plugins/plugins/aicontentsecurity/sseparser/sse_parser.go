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

package sseparser

import (
	"fmt"
	"strings"

	"github.com/tznbdbb/sseparser"
)

// ParsedEvent holds the structured data from a single SSE event.
type ParsedEvent struct {
	ID    string
	Event string
	Data  string
	Retry string
}

// toParsedEvent converts a raw sseparser.Event into our custom ParsedEvent struct.
func toParsedEvent(rawEvent sseparser.Event) *ParsedEvent {
	event := &ParsedEvent{}
	var dataBuilder strings.Builder
	var firstDataLine = true

	for _, field := range rawEvent.Fields() {
		switch field.Name {
		case "id":
			event.ID = field.Value
		case "event":
			event.Event = field.Value
		case "retry":
			event.Retry = field.Value
		case "data":
			if !firstDataLine {
				dataBuilder.WriteByte('\n')
			}
			dataBuilder.WriteString(field.Value)
			firstDataLine = false
		}
	}
	event.Data = dataBuilder.String()
	return event
}

// StreamEventParser is a parser for handling Server-Sent Events (SSE) streams.
type StreamEventParser struct {
	buf             []byte
	eventBoundaries []int

	parsedOffset int
	lastParseEnd int

	initialCapacity int

	shrinkFactor float64
	resizeFactor float64
}

type Option func(*StreamEventParser)

func WithCapacity(capacity int) Option {
	return func(m *StreamEventParser) {
		if capacity > 0 {
			m.initialCapacity = capacity
		}
	}
}

func NewStreamEventParser(opts ...Option) *StreamEventParser {
	m := &StreamEventParser{
		initialCapacity: 4096,
		shrinkFactor:    2,
		resizeFactor:    1.3,
	}
	for _, opt := range opts {
		opt(m)
	}
	m.buf = make([]byte, 0, m.initialCapacity)
	m.eventBoundaries = make([]int, 0, 16)
	return m
}

// Append appends data to the internal buffer.
func (p *StreamEventParser) Append(data []byte) {
	p.buf = append(p.buf, data...)
}

// TryParse attempts to parse the next event in the buffer without consuming it.
// It starts parsing from the end of the last parsed event and adds the boundary of the new event to the pending queue.
//
// Returns:
//   - (*ParsedEvent, nil): A new event was successfully parsed.
//   - (nil, nil): The data in the buffer is insufficient to form a complete event.
//   - (nil, error): A data format error was encountered.
func (p *StreamEventParser) TryParse() (*ParsedEvent, error) {
	if p.lastParseEnd >= len(p.buf) {
		return nil, nil
	}

	unparsedData := p.buf[p.lastParseEnd:]
	rawEvent, consumedBytes, err := sseparser.ParseRawEvent(unparsedData)
	if err != nil {
		return nil, fmt.Errorf("sse data format error: %w", err)
	}
	if rawEvent == nil {
		return nil, nil
	}

	p.lastParseEnd += consumedBytes
	p.eventBoundaries = append(p.eventBoundaries, p.lastParseEnd)

	return toParsedEvent(rawEvent), nil
}

// Parse attempts to parse and consume the next event in the buffer.
func (p *StreamEventParser) Parse() (*ParsedEvent, error) {
	parsedEvent, err := p.TryParse()
	if err != nil {
		return nil, err
	}
	if parsedEvent != nil {
		p.Consume(len(p.eventBoundaries))

	}
	return parsedEvent, nil
}

// Consume marks n previously pre-read events from TryParse() as "consumed".
func (p *StreamEventParser) Consume(n int) int {
	if n <= 0 {
		return 0
	}

	if n > len(p.eventBoundaries) {
		n = len(p.eventBoundaries)
	}

	if n == 0 {
		return 0
	}

	newParsedOffset := p.eventBoundaries[n-1]
	consumedBytes := newParsedOffset - p.parsedOffset
	p.parsedOffset = newParsedOffset

	// Remove the consumed event boundaries from the queue.
	p.eventBoundaries = p.eventBoundaries[n:]

	return consumedBytes
}

// PruneParsedData removes all consumed data from the buffer to free up memory.
func (p *StreamEventParser) PruneParsedData() {
	if p.parsedOffset == 0 {
		return
	}

	// Update the offsets of the remaining event boundaries.
	for i := range p.eventBoundaries {
		p.eventBoundaries[i] -= p.parsedOffset
	}

	unparsedLen := len(p.buf) - p.parsedOffset
	copy(p.buf, p.buf[p.parsedOffset:])
	p.buf = p.buf[:unparsedLen]

	p.lastParseEnd -= p.parsedOffset
	p.parsedOffset = 0

	p.shrinkIfNeeded()
}

// shrinkIfNeeded checks if the buffer's capacity needs to be reduced and performs the shrink if necessary.
func (p *StreamEventParser) shrinkIfNeeded() {
	currentCap := cap(p.buf)
	currentLen := len(p.buf)

	// If the buffer is empty and its capacity is greater than the initial capacity, shrink it back to the initial capacity.
	if currentLen == 0 && currentCap > p.initialCapacity {
		p.buf = make([]byte, 0, p.initialCapacity)
		return
	}

	// Consider shrinking only when the capacity is greater than the initial capacity.
	if currentCap > p.initialCapacity {
		// Calculate the target capacity to trigger a shrink.
		targetShrinkCapacity := int(float64(currentLen) * p.shrinkFactor)
		if targetShrinkCapacity < p.initialCapacity {
			targetShrinkCapacity = p.initialCapacity
		}

		// If the current capacity is much larger than the actual data requires, perform a shrink.
		if currentCap > targetShrinkCapacity {
			newBuf := make([]byte, currentLen, int(float64(currentLen)*p.resizeFactor))
			copy(newBuf, p.buf)
			p.buf = newBuf
		}
	}
}

// Cap returns the current capacity of the internal buffer.
func (p *StreamEventParser) Cap() int {
	return cap(p.buf)
}

// Len returns the current length of the data in the internal buffer.
func (p *StreamEventParser) Len() int {
	return len(p.buf)
}

// AllBytes returns a byte slice of the entire internal buffer.
func (p *StreamEventParser) AllBytes() []byte {
	return p.buf
}

// ParsedBytes returns the consumed data portion.
func (p *StreamEventParser) ParsedBytes() []byte {
	return p.buf[:p.parsedOffset]
}

// UnparsedBytes returns the portion of the data that has not yet been pre-read by TryParse.
func (p *StreamEventParser) UnparsedBytes() []byte {
	if p.lastParseEnd >= len(p.buf) {
		return nil
	}
	return p.buf[p.lastParseEnd:]
}
