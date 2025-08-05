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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tznbdbb/sseparser"
)

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

func TestToParsedEvent_CorrectlyJoinsMultiLineData(t *testing.T) {
	rawEvent := sseparser.Event{
		sseparser.Field{Name: "data", Value: "line 1"},
		sseparser.Field{Name: "id", Value: "the-id"},
		sseparser.Field{Name: "data", Value: "line 2"},
		sseparser.Field{Name: "event", Value: "my-event"},
	}

	parsed := toParsedEvent(rawEvent)

	assert.Equal(t, "the-id", parsed.ID)
	assert.Equal(t, "my-event", parsed.Event)
	assert.Equal(t, "line 1\nline 2", parsed.Data)
}

// TestBufferManagement_PruneAndShrink 旨在测试缓冲区的内部管理逻辑，
func TestBufferManagement_PruneAndShrink(t *testing.T) {
	// 场景1：测试在消耗部分数据后，显式调用 PruneParsedData 可以腾出空间，避免不必要的扩容。
	t.Run("Should Prune Data to Make Space Instead of Resizing", func(t *testing.T) {
		initialCap := 110
		parser := NewStreamEventParser(WithCapacity(initialCap))

		event1 := []byte("id: 1\ndata: first event, fairly long to occupy space\n\n") // 56 bytes
		event2 := []byte("id: 2\ndata: second event, also quite long message\n\n")    // 53 bytes
		require.True(t, len(event1)+len(event2) <= initialCap, "Test data should fit in initial capacity")

		// 填充缓冲区到接近满的状态
		parser.Append(event1)
		parser.Append(event2)
		require.Equal(t, initialCap, parser.Cap())
		require.Equal(t, len(event1)+len(event2), parser.Len()) // len = 109

		// 解析并消耗第一个事件。此时 event1 的字节仍在缓冲区，只是逻辑上被标记为“已解析”
		parsedEvent, err := parser.Parse()
		require.NoError(t, err)
		require.NotNil(t, parsedEvent)
		require.Equal(t, "1", parsedEvent.ID)

		parser.PruneParsedData()
		// 修剪后，缓冲区中只剩下 event2 的数据。
		require.Equal(t, len(event2), parser.Len(), "Buffer length should be equal to the unparsed event's length after pruning")
		require.Equal(t, initialCap, parser.Cap())

		// 追加event后容量不应该改变，因为修剪已消耗的数据腾出了足够空间。
		event3 := []byte("id: 3\ndata: a third event to fit after prune\n\n") // ~47 bytes
		parser.Append(event3)
		assert.Equal(t, initialCap, parser.Cap(), "Capacity should not increase because pruning created enough space")
		assert.Equal(t, len(event2)+len(event3), parser.Len())

		// 验证剩余的事件可以被正确解析
		ev2, err2 := parser.Parse()
		require.NoError(t, err2)
		require.NotNil(t, ev2)
		assert.Equal(t, "2", ev2.ID)

		ev3, err3 := parser.Parse()
		require.NoError(t, err3)
		require.NotNil(t, ev3)
		assert.Equal(t, "3", ev3.ID)
	})

	// 场景2：测试缓冲区在扩容后，如果所有数据都被消耗并修剪，会收缩回初始容量。
	t.Run("Should Shrink Buffer After Resizing and Consumption", func(t *testing.T) {
		initialCap := 50
		parser := NewStreamEventParser(WithCapacity(initialCap))

		// 这个事件会强制缓冲区扩容
		largeEvent := []byte("id: large\ndata: this event is specifically designed to be larger than the initial capacity\n\n") // 96 bytes
		require.True(t, len(largeEvent) > initialCap, "Event must be larger than initial capacity to trigger resize")
		parser.Append(largeEvent)

		// 不假设扩容后确切的容量，因为这是 Go 运行时决定的。 只验证容量确实比初始值增大了。
		originalCap := parser.Cap()
		require.True(t, originalCap > initialCap, "Buffer should have resized to accommodate large event")

		// 解析并消耗这个大事件
		event, err := parser.Parse()
		require.NoError(t, err)
		require.NotNil(t, event)

		parser.PruneParsedData()

		// 当缓冲区为空时，收缩回 initialCapacity。
		assert.Equal(t, 0, parser.Len(), "Buffer should be empty after consuming and pruning all events")
		assert.Equal(t, initialCap, parser.Cap(), "Buffer should shrink back to initial capacity after being emptied")
	})
}

// TestBoundaryAndMalformedEvents 用于测试各种边界情况和格式不规范的 SSE 事件，
func TestBoundaryAndMalformedEvents(t *testing.T) {
	// SSE 规范要求能够处理 CRLF (`\r\n`) 作为换行符
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

	// 无效的字段（比如没有冒号）应该被直接忽略
	t.Run("Ignores Invalid Fields Without Colon", func(t *testing.T) {
		parser := NewStreamEventParser()
		// "thisisinvalid" 应该被忽略, 但 "data: valid" 应该被解析
		parser.Append([]byte("thisisinvalid\ndata: valid\n\n"))

		event, err := parser.Parse()
		require.NoError(t, err)
		require.NotNil(t, event)
		assert.Equal(t, "valid", event.Data)
	})
}

func TestInterleavedOperations(t *testing.T) {
	parser := NewStreamEventParser()

	event1Data := []byte("data: first\n\n")
	event2Data := []byte("data: second\n\n")
	event3Data := []byte("id: 3\ndata: third\n\n")

	parser.Append(event1Data)
	parser.Append(event2Data)

	// 尝试解析一个event
	ev1, err1 := parser.TryParse()
	require.NoError(t, err1)
	require.NotNil(t, ev1)
	assert.Equal(t, "first", ev1.Data)

	// 再次 TryParse，应该看到第二个事件
	ev2, err2 := parser.TryParse()
	require.NoError(t, err2)
	require.NotNil(t, ev2)
	assert.Equal(t, "second", ev2.Data)

	// 此时没有更多可查看的事件了
	evNil, errNil := parser.TryParse()
	require.NoError(t, errNil)
	assert.Nil(t, evNil)

	// 只消耗第一个被查看的事件
	consumedBytes := parser.Consume(1)
	assert.Equal(t, len(event1Data), consumedBytes, "Should consume the bytes of the first event")

	// 追加第三个事件的数据
	parser.Append(event3Data)

	// 再次 TryParse，应该看到第三个事件
	ev3, err3 := parser.TryParse()
	require.NoError(t, err3)
	require.NotNil(t, ev3)
	assert.Equal(t, "third", ev3.Data)

	// 一次性消耗剩余的两个事件 (event 2 和 event 3)
	consumedBytes = parser.Consume(2)
	assert.Equal(t, len(event2Data)+len(event3Data), consumedBytes)

	parser.PruneParsedData()

	// 最终缓冲区为空
	assert.Equal(t, 0, parser.Len(), "Buffer should be empty")
	finalEv, finalErr := parser.Parse()
	assert.NoError(t, finalErr)
	assert.Nil(t, finalEv, "No events should remain in the parser")
}
