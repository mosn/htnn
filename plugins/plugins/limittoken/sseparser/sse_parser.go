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
	"bytes"
	"strings"
	"unsafe"
)

// ParsedEvent structured data from a single SSE event.
type ParsedEvent struct {
	ID    string
	Event string
	Data  string
	Retry string
}

// findEventEnd looks for an SSE event terminator: two consecutive line breaks.
// this function returns (-1, 0) to indicate "need more data".
func findEventEnd(data []byte) (int, int) {
	n := len(data)
	if n == 0 {
		return -1, 0
	}

	isCRLFAt := func(i int) bool {
		return i+1 < n && data[i] == '\r' && data[i+1] == '\n'
	}

	// Scan from left to right: find first line break (LF or CRLF),
	// then check whether a second line break immediately follows.
	for i := 0; i < n; i++ {
		var lb1Len int

		// detect first line break: CRLF or LF
		if isCRLFAt(i) {
			lb1Len = 2
		} else if data[i] == '\n' {
			lb1Len = 1
		} else if data[i] == '\r' {
			// single '\r' not followed by '\n' -> ambiguous (could become CRLF once next chunk arrives)
			// Conservative choice: if '\r' is the last byte, wait for more data.
			if i+1 >= n {
				return -1, 0
			}
			// If next byte exists but is not '\n', then '\r' is not a CRLF and not an LF -> treat as normal byte.
			continue
		} else {
			continue
		}

		j := i + lb1Len // start index for potential second line break

		// If there is no room for a second line break, we cannot conclude yet.
		if j >= n {
			return -1, 0
		}

		// detect second line break: CRLF or LF
		if isCRLFAt(j) {
			return i, lb1Len + 2
		}
		if data[j] == '\n' {
			return i, lb1Len + 1
		}
		// If second char is '\r' and it's the last byte, wait for more data (could be CRLF)
		if data[j] == '\r' {
			if j+1 >= n {
				return -1, 0
			}
			// if j+1 exists and isn't '\n', then it's not a line break -> continue scanning
			if j+1 < n && data[j+1] != '\n' {
				continue
			}
		}
		// otherwise second char is not a line break -> continue scanning
	}

	return -1, 0
}

// StreamEventParser is a parser for handling Server-Sent Events (SSE) streams.
type StreamEventParser struct {
	buf             []byte
	eventBoundaries []int

	parsedOffset    int
	lastParseEnd    int
	initialCapacity int

	// Reusable builder to reduce allocations for multi-line data fields.
	dataBuilder strings.Builder

	// Reusable slice for collecting data lines to reduce allocations.
	dataLines [][]byte
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
	}
	for _, opt := range opts {
		opt(m)
	}
	m.buf = make([]byte, 0, m.initialCapacity)
	m.eventBoundaries = make([]int, 0, 16)
	m.dataLines = make([][]byte, 0, 8)
	m.dataBuilder.Grow(1024)
	return m
}

func (p *StreamEventParser) Append(data []byte) {
	p.buf = append(p.buf, data...)
}

func (p *StreamEventParser) b2s(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(&b[0], len(b))
}

func (p *StreamEventParser) parseRawEvent(data []byte) (*ParsedEvent, int) {
	end, terminatorLen := findEventEnd(data)
	if end == -1 {
		return nil, 0
	}

	eventBlock := data[:end]
	consumedBytes := end + terminatorLen
	event := &ParsedEvent{}

	p.dataLines = p.dataLines[:0]

	for len(eventBlock) > 0 {
		lineEndIndex := bytes.IndexByte(eventBlock, '\n')
		var line []byte

		if lineEndIndex != -1 {
			line = eventBlock[:lineEndIndex]
			eventBlock = eventBlock[lineEndIndex+1:]
		} else {
			// This is the last line in the block.
			line = eventBlock
			eventBlock = nil
		}

		// Handle CRLF line endings by trimming the CR.
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}

		if len(line) == 0 || line[0] == ':' {
			// Ignore empty lines or comment lines.
			continue
		}

		var fieldNameBytes, valueBytes []byte
		if colonIndex := bytes.IndexByte(line, ':'); colonIndex != -1 {
			fieldNameBytes = line[:colonIndex]
			valueBytes = line[colonIndex+1:]
			// Per SSE spec, trim a single leading space if it exists.
			if len(valueBytes) > 0 && valueBytes[0] == ' ' {
				valueBytes = valueBytes[1:]
			}
		} else {
			// No colon: field with an empty value.
			fieldNameBytes = line
			valueBytes = []byte{} // Represent empty value as an empty slice
		}

		// Use bytes.Equal to avoid allocating a string for the field name.
		if bytes.Equal(fieldNameBytes, []byte("id")) {
			event.ID = p.b2s(valueBytes)
		} else if bytes.Equal(fieldNameBytes, []byte("event")) {
			event.Event = p.b2s(valueBytes)
		} else if bytes.Equal(fieldNameBytes, []byte("retry")) {
			event.Retry = p.b2s(valueBytes)
		} else if bytes.Equal(fieldNameBytes, []byte("data")) {
			p.dataLines = append(p.dataLines, valueBytes)
		}
		// Ignore unknown fields.
	}

	// Process collected data lines using the reusable slice from the struct.
	if len(p.dataLines) == 1 {
		event.Data = p.b2s(p.dataLines[0])
	} else if len(p.dataLines) > 1 {
		// For multi-line data, the builder is necessary for concatenation.
		p.dataBuilder.Reset()
		for i, line := range p.dataLines {
			if i > 0 {
				p.dataBuilder.WriteByte('\n')
			}
			// Write the byte slice directly to avoid an intermediate string conversion.
			p.dataBuilder.Write(line)
		}
		event.Data = p.dataBuilder.String()
	}
	// If len(p.dataLines) == 0, event.Data remains its zero value (""), which is correct.

	return event, consumedBytes
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
	parsedEvent, consumedBytes := p.parseRawEvent(unparsedData)

	if parsedEvent == nil {
		// need more data
		return nil, nil
	}

	p.lastParseEnd += consumedBytes
	p.eventBoundaries = append(p.eventBoundaries, p.lastParseEnd)

	return parsedEvent, nil
}

// Parse attempts to parse and consume all the event in the buffer.
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

	newParsedOffset := p.eventBoundaries[n-1]
	consumedBytes := newParsedOffset - p.parsedOffset
	p.parsedOffset = newParsedOffset
	p.eventBoundaries = p.eventBoundaries[n:]

	return consumedBytes
}

// PruneParsedData removes all consumed data from the buffer.
func (p *StreamEventParser) PruneParsedData() {
	if p.parsedOffset == 0 {
		return
	}

	for i := range p.eventBoundaries {
		p.eventBoundaries[i] -= p.parsedOffset
	}

	unparsedLen := len(p.buf) - p.parsedOffset
	copy(p.buf, p.buf[p.parsedOffset:])
	p.buf = p.buf[:unparsedLen]

	p.lastParseEnd -= p.parsedOffset
	p.parsedOffset = 0
}

func (p *StreamEventParser) Cap() int {
	return cap(p.buf)
}

func (p *StreamEventParser) Len() int {
	return len(p.buf)
}

func (p *StreamEventParser) AllBytes() []byte {
	return p.buf
}

func (p *StreamEventParser) ParsedBytes() []byte {
	if p.parsedOffset <= 0 {
		return nil
	}
	return p.buf[:p.parsedOffset]
}

func (p *StreamEventParser) UnparsedBytes() []byte {
	if p.lastParseEnd >= len(p.buf) {
		return nil
	}
	return p.buf[p.lastParseEnd:]
}
