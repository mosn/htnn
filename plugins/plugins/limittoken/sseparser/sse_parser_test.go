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
	"math/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindEventEnd(t *testing.T) {
	testCases := []struct {
		name        string
		input       []byte
		expectedPos int
		expectedLen int
	}{
		// Cases: No terminator found
		{"empty buffer", []byte(""), -1, 0},
		{"no terminator", []byte("data: hello"), -1, 0},

		// Cases: Ambiguous endings (incomplete terminator)
		{"ends with single CR", []byte("data: hello\r"), -1, 0},
		{"ends with single LF", []byte("data: hello\n"), -1, 0},
		{"ends with CRLF", []byte("data: hello\r\n"), -1, 0},
		{"ends with LF then CR", []byte("data: hello\n\r"), -1, 0},

		// Cases: Valid terminators
		{"simple LF LF", []byte("data: a\n\n"), 7, 2},
		{"simple CRLF CRLF", []byte("data: a\r\n\r\n"), 7, 4},
		{"mixed LF CRLF", []byte("data: a\n\r\n"), 7, 3},
		{"mixed CRLF LF", []byte("data: a\r\n\n"), 7, 3},

		// Cases: Terminators at the start of the buffer
		{"terminator at start LF LF", []byte("\n\n"), 0, 2},
		{"terminator at start CRLF CRLF", []byte("\r\n\r\n"), 0, 4},
		{"terminator at start LF CRLF", []byte("\n\r\n"), 0, 3},
		{"terminator at start CRLF LF", []byte("\r\n\n"), 0, 3},

		// Cases: Other scenarios
		{"finds first of multiple terminators", []byte("id: 1\n\ndata: 2\n\n"), 5, 2},
		{"single CR in middle is ignored", []byte("data: a\r_b\n\n"), 10, 2},
		{"LF followed by CR in middle is not a terminator", []byte("data: event\n\r-not-a-terminator-\n\n"), 31, 2},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pos, length := findEventEnd(tc.input)
			assert.Equal(t, tc.expectedPos, pos, "Position mismatch")
			assert.Equal(t, tc.expectedLen, length, "Length mismatch")
		})
	}
}

func TestParserOptions(t *testing.T) {
	t.Run("WithCapacity", func(t *testing.T) {
		// Test with a specific capacity
		parser := NewStreamEventParser(WithCapacity(1024))
		assert.GreaterOrEqual(t, parser.Cap(), 1024)

		// Test with zero or negative capacity (should use default)
		parser = NewStreamEventParser(WithCapacity(0))
		assert.Greater(t, parser.Cap(), 0, "Capacity should be the default, not zero")

		parser = NewStreamEventParser(WithCapacity(-100))
		assert.Greater(t, parser.Cap(), 0, "Negative capacity should be ignored")
	})

}

func TestTryParse_EmptyBuffer(t *testing.T) {
	parser := NewStreamEventParser()
	event, err := parser.TryParse()
	assert.NoError(t, err, "Error should be nil for an empty buffer")
	assert.Nil(t, event, "Event should be nil for an empty buffer")
}

func TestTryParse_SingleCompleteEvent(t *testing.T) {
	parser := NewStreamEventParser()
	sseData := []byte("id: 1\ndata: test message\n\n")
	parser.Append(sseData)

	event, err := parser.TryParse()
	require.NoError(t, err)
	require.NotNil(t, event)
	assert.Equal(t, "1", event.ID)
	assert.Equal(t, "test message", event.Data)

	event2, err2 := parser.TryParse()
	require.NoError(t, err2)
	require.Nil(t, event2, "Second TryParse should return nil")

	consumed := parser.Consume(1)
	assert.Equal(t, len(sseData), consumed)

	event3, err3 := parser.TryParse()
	assert.NoError(t, err3)
	assert.Nil(t, event3, "There should be no more events after consuming")
}

func TestTryParse_MultipleEvents_Sequential(t *testing.T) {
	parser := NewStreamEventParser()
	event1Data := []byte("data: first\n\n")
	event2Data := []byte("data: second\n\n")
	parser.Append(event1Data)
	parser.Append(event2Data)

	event1, err1 := parser.TryParse()
	require.NoError(t, err1)
	require.NotNil(t, event1)
	assert.Equal(t, "first", event1.Data)

	event2, err2 := parser.TryParse()
	require.NoError(t, err2)
	require.NotNil(t, event2)
	assert.Equal(t, "second", event2.Data)

	event3, err3 := parser.TryParse()
	require.NoError(t, err3)
	assert.Nil(t, event3)
}

func TestParse_DirectConsumption(t *testing.T) {
	parser := NewStreamEventParser()
	sseData := []byte("id: 1\ndata: test message\n\n")
	parser.Append(sseData)

	event, err := parser.Parse()
	require.NoError(t, err)
	require.NotNil(t, event)
	assert.Equal(t, "1", event.ID)
	assert.Equal(t, "test message", event.Data)

	event2, err2 := parser.Parse()
	assert.NoError(t, err2)
	assert.Nil(t, event2, "No more events should be available after Parse")
}

func TestTryParse_HandlesIncompleteDataGracefully(t *testing.T) {
	parser := NewStreamEventParser()
	parser.Append([]byte("data: incomplete message"))

	event1, err1 := parser.TryParse()
	assert.NoError(t, err1)
	assert.Nil(t, event1, "Event should be nil for incomplete data")

	parser.Append([]byte("\n\n"))
	event2, err2 := parser.TryParse()
	require.NoError(t, err2)
	require.NotNil(t, event2, "Should parse the event after data is complete")
	assert.Equal(t, "incomplete message", event2.Data)
}

func TestFragmentedNetworkAndMultiLineData(t *testing.T) {
	parser := NewStreamEventParser()

	fragments := [][]byte{
		[]byte("id: 123\n"),
		[]byte("data: chunk 1\n"),
		[]byte("data: chunk 2\n"),
		[]byte(": this is a comment\n"),
		[]byte("\n"),
		[]byte("event: custom\n"),
		[]byte("data: final message\n\n"),
	}

	var parsedEvents []*ParsedEvent
	for _, frag := range fragments {
		parser.Append(frag)
		for {
			event, err := parser.Parse()
			require.NoError(t, err)
			if event == nil {
				break
			}
			parsedEvents = append(parsedEvents, event)
		}
	}

	require.Len(t, parsedEvents, 2, "Should have parsed exactly two events")

	event1 := parsedEvents[0]
	assert.Equal(t, "123", event1.ID)
	assert.Equal(t, "chunk 1\nchunk 2", event1.Data)

	event2 := parsedEvents[1]
	assert.Equal(t, "custom", event2.Event)
	assert.Equal(t, "final message", event2.Data)
}

func TestTryParse_CommentOnlyEvent(t *testing.T) {
	parser := NewStreamEventParser()
	commentEvent := []byte(": first comment\n: second comment\n\n")
	parser.Append(commentEvent)

	event, err := parser.TryParse()
	require.NoError(t, err)
	require.NotNil(t, event, "A comment-only event is still a valid event")
	assert.Empty(t, event.ID)
	assert.Empty(t, event.Data)
	assert.Empty(t, event.Event)
}

// TestBoundaryAndMalformedEvents tests various boundary cases and malformed SSE events.
func TestBoundaryAndMalformedEvents(t *testing.T) {
	// The SSE specification requires handling CRLF (`\r\n`) as line endings.
	t.Run("Handles CRLF Line Endings", func(t *testing.T) {
		parser := NewStreamEventParser()
		crlfData := []byte("data: Test with CRLF\r\n\r\n")
		parser.Append(crlfData)

		event, err := parser.Parse()
		require.NoError(t, err)
		require.NotNil(t, event)
		assert.Equal(t, "Test with CRLF", event.Data)
	})

	t.Run("Handles Empty Data Field", func(t *testing.T) {
		testCases := map[string][]byte{
			"with colon":      []byte("data:\n\n"),
			"without colon":   []byte("data\n\n"),
			"with space":      []byte("data: \n\n"),
			"with id":         []byte("id: 1\ndata:\n\n"),
			"multiline empty": []byte("data: line1\ndata:\ndata: line3\n\n"),
		}

		for name, data := range testCases {
			t.Run(name, func(t *testing.T) {
				parser := NewStreamEventParser()
				parser.Append(data)
				event, err := parser.Parse()
				require.NoError(t, err)
				require.NotNil(t, event)

				if name == "multiline empty" {
					assert.Equal(t, "line1\n\nline3", event.Data)
				} else if name == "with id" {
					assert.Equal(t, "1", event.ID)
					assert.Equal(t, "", event.Data)
				} else {
					assert.Equal(t, "", event.Data)
				}
			})
		}
	})

	t.Run("Ignores Invalid Fields Without Colon", func(t *testing.T) {
		parser := NewStreamEventParser()
		parser.Append([]byte("thisisnotafield\ndata: valid\n\n"))

		event, err := parser.Parse()
		require.NoError(t, err)
		require.NotNil(t, event)
		assert.Equal(t, "valid", event.Data, "Should parse the valid field and ignore the invalid one")
	})

	t.Run("Ignores Unknown Fields With Colon", func(t *testing.T) {
		parser := NewStreamEventParser()
		parser.Append([]byte("data: a\nunknown: field\nid: 1\n\n"))
		event, err := parser.Parse()
		require.NoError(t, err)
		require.NotNil(t, event)
		assert.Equal(t, "a", event.Data)
		assert.Equal(t, "1", event.ID)
		assert.Empty(t, event.Event, "Unknown field 'unknown' should not populate any known field")
		assert.Empty(t, event.Retry)
	})
}

func TestInterleavedOperations(t *testing.T) {
	parser := NewStreamEventParser()

	event1Data := []byte("data: first\n\n")
	event2Data := []byte("data: second\n\n")
	event3Data := []byte("id: 3\ndata: third\n\n")

	parser.Append(event1Data)
	parser.Append(event2Data)

	// Try to parse one event.
	ev1, err1 := parser.TryParse()
	require.NoError(t, err1)
	require.NotNil(t, ev1)
	assert.Equal(t, "first", ev1.Data)

	// TryParse again, should see the second event.
	ev2, err2 := parser.TryParse()
	require.NoError(t, err2)
	require.NotNil(t, ev2)
	assert.Equal(t, "second", ev2.Data)

	// At this point, there are no more events to peek at.
	evNil, errNil := parser.TryParse()
	require.NoError(t, errNil)
	assert.Nil(t, evNil)

	// Consume only the first peeked event.
	consumedBytes := parser.Consume(1)
	assert.Equal(t, len(event1Data), consumedBytes, "Should consume the bytes of the first event")

	// Append the data for the third event.
	parser.Append(event3Data)

	// TryParse again, should see the third event.
	ev3, err3 := parser.TryParse()
	require.NoError(t, err3)
	require.NotNil(t, ev3)
	assert.Equal(t, "third", ev3.Data)

	// Consume the remaining two events (event 2 and event 3) at once.
	consumedBytes = parser.Consume(2)
	assert.Equal(t, len(event2Data)+len(event3Data), consumedBytes)

	parser.PruneParsedData()

	// The buffer should be empty in the end.
	assert.Equal(t, 0, parser.Len(), "Buffer should be empty")
	finalEv, finalErr := parser.Parse()
	assert.NoError(t, finalErr)
	assert.Nil(t, finalEv, "No events should remain in the parser")
}

func TestConsume_EdgeCases(t *testing.T) {
	parser := NewStreamEventParser()
	event1 := []byte("data: a\n\n")
	event2 := []byte("data: b\n\n")
	parser.Append(event1)
	parser.Append(event2)

	// TryParse both events to populate eventBoundaries
	_, _ = parser.TryParse()
	_, _ = parser.TryParse()
	require.Len(t, parser.eventBoundaries, 2)
	initialParsedOffset := parser.parsedOffset

	// Consume 0
	consumed := parser.Consume(0)
	assert.Equal(t, 0, consumed)
	assert.Len(t, parser.eventBoundaries, 2)
	assert.Equal(t, initialParsedOffset, parser.parsedOffset)

	// Consume negative
	consumed = parser.Consume(-5)
	assert.Equal(t, 0, consumed)
	assert.Len(t, parser.eventBoundaries, 2)

	// Consume more than available
	consumed = parser.Consume(5) // Only 2 are available
	assert.Equal(t, len(event1)+len(event2), consumed)
	assert.Len(t, parser.eventBoundaries, 0)
}

func TestPrune_EdgeCases(t *testing.T) {
	parser := NewStreamEventParser()

	// 1. Prune on an empty/fresh parser
	parser.PruneParsedData()
	assert.Equal(t, 0, parser.Len())
	assert.Equal(t, 0, parser.parsedOffset)
	assert.Equal(t, 0, parser.lastParseEnd)

	// 2. Append data but don't parse, then prune
	data := []byte("data: test\n\n")
	parser.Append(data)
	parser.PruneParsedData() // Should do nothing as parsedOffset is 0
	assert.Equal(t, len(data), parser.Len())

	// 3. Parse everything, then prune
	_, err := parser.Parse() // This consumes and updates parsedOffset
	require.NoError(t, err)
	parser.PruneParsedData()
	assert.Equal(t, 0, parser.Len())
	assert.Equal(t, 0, parser.parsedOffset)
	assert.Equal(t, 0, parser.lastParseEnd)
}

func TestStateAndHelpers(t *testing.T) {
	parser := NewStreamEventParser(WithCapacity(100))

	// Initial state
	assert.Equal(t, 0, parser.Len())
	assert.GreaterOrEqual(t, parser.Cap(), 100)
	assert.Empty(t, parser.AllBytes())
	assert.Nil(t, parser.ParsedBytes())
	assert.Nil(t, parser.UnparsedBytes())

	event1 := []byte("data: first\n\n")
	event2 := []byte("data: second\n\n")
	allData := bytes.Join([][]byte{event1, event2}, nil)
	parser.Append(allData)

	// State after appending
	assert.Equal(t, len(allData), parser.Len())
	assert.Equal(t, allData, parser.AllBytes())
	assert.Nil(t, parser.ParsedBytes())
	assert.Equal(t, allData, parser.UnparsedBytes())
	assert.Equal(t, 0, parser.lastParseEnd)
	assert.Equal(t, 0, parser.parsedOffset)

	// State after TryParse (but not Consume)
	ev, err := parser.TryParse()
	require.NoError(t, err)
	require.NotNil(t, ev)
	assert.Equal(t, len(event1), parser.lastParseEnd)
	assert.Equal(t, 0, parser.parsedOffset) // Not consumed yet
	assert.Nil(t, parser.ParsedBytes())
	assert.Equal(t, event2, parser.UnparsedBytes())

	// State after Consume
	parser.Consume(1)
	assert.Equal(t, len(event1), parser.parsedOffset)
	assert.Equal(t, event1, parser.ParsedBytes())
	assert.Equal(t, event2, parser.UnparsedBytes())

	// State after Prune
	parser.PruneParsedData()
	assert.Equal(t, len(event2), parser.Len())
	assert.Equal(t, event2, parser.AllBytes())
	assert.Equal(t, 0, parser.parsedOffset) // Reset after prune
	assert.Equal(t, 0, parser.lastParseEnd) // Adjusted after prune
	assert.Nil(t, parser.ParsedBytes())
	assert.Equal(t, event2, parser.UnparsedBytes())

	// Parse the rest
	_, _ = parser.Parse()
	assert.Equal(t, len(event2), parser.parsedOffset)
	assert.Equal(t, len(event2), parser.lastParseEnd)
	assert.Equal(t, event2, parser.ParsedBytes())
	assert.Nil(t, parser.UnparsedBytes())
}

func TestStreamingParser_UnsafeLifecycle(t *testing.T) {
	parser := NewStreamEventParser()

	parser.Append([]byte("id: 1\ndata: first\n\n"))

	event, err := parser.Parse()
	require.NoError(t, err)
	require.NotNil(t, event)

	assert.Equal(t, "1", event.ID)
	assert.Equal(t, "first", event.Data)

	idBeforePrune := event.ID

	parser.PruneParsedData()
	parser.Append([]byte("id: 2\ndata: second\n\n"))

	// This assertion is not guaranteed to fail, as memory might not be overwritten
	// in the exact same way every time. The principle is what's being tested.
	// The goal is to demonstrate the danger that the underlying array has changed.
	assert.NotEqual(t, "1", idBeforePrune, "string from unsafe event should be corrupted after prune")
}

func FuzzParse(f *testing.F) {
	f.Add([]byte("data: valid event\n\n"))
	f.Add([]byte("id: 1\ndata: first\n\n:comment\ndata: second\n\n"))
	f.Add([]byte("data: incomplete event"))
	f.Add([]byte("data: line1\ndata: line2\n\n"))
	f.Add([]byte(":\n:\n\n"))
	f.Add([]byte("\n\n"))
	f.Add([]byte("data\n\n"))
	f.Add([]byte("data:\r\n\r\n"))
	f.Add([]byte("field without colon\n\n"))
	f.Add([]byte(""))
	f.Add([]byte{'d', 'a', 't', 'a', ':', ' ', 0, 1, 2, 3, '\n', '\n'})                      // Non-UTF-8 data
	f.Add([]byte("data: line1\ndata: line2\r\ndata: line3\n\n"))                             // Mixed line endings
	f.Add([]byte("data: first\n\n" + strings.Repeat(":comment\n", 50) + "data: second\n\n")) // Lots of comments
	f.Add([]byte("data: first\n\n" + strings.Repeat("\n", 50) + "data: second\n\n"))         // Lots of empty lines
	f.Add([]byte("\n\n\r\n\r\n\n\n"))                                                        // Only terminators
	f.Add([]byte("data: a\r\n\ndata: b\n\r\n"))                                              // Terminator variations
	f.Add([]byte("data: " + strings.Repeat("A", 4096) + "\n\n"))                             // Very long data line

	f.Fuzz(func(t *testing.T, data []byte) {
		groundTruthParser := NewStreamEventParser()
		groundTruthParser.Append(data)
		var groundTruthEvents []*ParsedEvent
		for {
			event, err := groundTruthParser.Parse()
			require.NoError(t, err, "Ground truth parser should never error")
			if event == nil {
				break
			}
			groundTruthEvents = append(groundTruthEvents, event)
		}

		runStreamingTest(t, data, groundTruthEvents)
	})
}

func runStreamingTest(t *testing.T, data []byte, groundTruthEvents []*ParsedEvent) {
	t.Helper()

	// Initialize the streaming parser for the test.
	streamingParser := NewStreamEventParser()
	eventIndex := 0 // Tracks the current event index for comparison.
	offset := 0

	// Simulate a data stream by appending chunks until all data is processed.
	for offset < len(data) {
		// Randomly determine the size of the next chunk to append.
		remaining := len(data) - offset
		chunkSize := 1
		if remaining > 1 {
			// Limit max chunk size to better simulate a stream.
			const maxChunkSize = 64
			size := rand.Intn(remaining) + 1
			if size > maxChunkSize {
				size = maxChunkSize
			}
			chunkSize = size
		}
		end := offset + chunkSize
		chunk := data[offset:end]
		streamingParser.Append(chunk)
		offset = end

		// After appending a new chunk, loop to parse any complete events that may have formed.
		for {
			var event *ParsedEvent
			var err error

			event, err = streamingParser.Parse()

			require.NoError(t, err, "Streaming parser should not error")

			if event == nil {
				break
			}

			require.Less(t, eventIndex, len(groundTruthEvents),
				"Produced more events than ground truth at index %d", eventIndex)
			expectedEvent := groundTruthEvents[eventIndex]

			assert.Equal(t, expectedEvent.ID, event.ID, "ID mismatch at event %d", eventIndex)
			assert.Equal(t, expectedEvent.Event, event.Event, "Event mismatch at event %d", eventIndex)
			assert.Equal(t, expectedEvent.Data, event.Data, "Data mismatch at event %d", eventIndex)
			assert.Equal(t, expectedEvent.Retry, event.Retry, "Retry mismatch at event %d", eventIndex)

			eventIndex++
		}

		// Randomly decide whether to prune already parsed data from the buffer.
		if rand.Intn(2) == 0 {
			streamingParser.PruneParsedData()
		}
	}

	assert.Equal(t, len(groundTruthEvents), eventIndex, "Final event count mismatch")

	// Clean up and check the final state.
	streamingParser.PruneParsedData()
	assert.Zero(t, len(streamingParser.ParsedBytes()), "Parsed bytes should be empty after final prune")
}
