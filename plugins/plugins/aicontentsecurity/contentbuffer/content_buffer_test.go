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

package contentbuffer

import (
	"fmt"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

type resultAccumulator struct {
	t             *testing.T
	totalEvents   int
	allChunks     []string
	initialBuffer *ContentBuffer
}

func newResultAccumulator(t *testing.T, buffer *ContentBuffer) *resultAccumulator {
	return &resultAccumulator{
		t:             t,
		initialBuffer: buffer,
	}
}

func (acc *resultAccumulator) accumulate() {
	result := acc.initialBuffer.GetCompletedResult()
	acc.totalEvents += result.CompletedEvents
	acc.allChunks = append(acc.allChunks, result.Chunks...)
}

func (acc *resultAccumulator) flushAndAccumulate() {
	acc.initialBuffer.Flush()
	acc.accumulate()
}

func (acc *resultAccumulator) check(expectedTotalEvents int, expectedChunks []string) {
	assert.Equal(acc.t, expectedTotalEvents, acc.totalEvents, "Total event count does not match")
	assert.Equal(acc.t, expectedChunks, acc.allChunks, "Chunk content does not match")
}

func TestContentBuffer(t *testing.T) {
	// Basic usage without overlap
	t.Run("Basic/NoOverlap", func(t *testing.T) {
		buffer := NewContentBuffer(WithMaxChars(10), WithOverlapCharNum(0))
		acc := newResultAccumulator(t, buffer)

		buffer.Write([]byte("事件A1"))
		buffer.Write([]byte("事件B2"))
		buffer.Write([]byte("事件C3"))
		acc.flushAndAccumulate()

		expectedChunks := []string{"事件A1事件B2事件", "C3"}
		acc.check(3, expectedChunks)
	})

	// Basic usage with overlap
	t.Run("Basic/WithOverlap", func(t *testing.T) {
		buffer := NewContentBuffer(WithMaxChars(10), WithOverlapCharNum(3))
		acc := newResultAccumulator(t, buffer)

		buffer.Write([]byte("0123456789"))
		buffer.Write([]byte("ABCDEFGHIJ"))
		acc.flushAndAccumulate()

		expectedChunks := []string{"0123456789", "789ABCDEFG", "EFGHIJ"}
		acc.check(2, expectedChunks)
	})

	// A single long event spans multiple chunks with overlap
	t.Run("Complex/SingleLongEventWithOverlap", func(t *testing.T) {
		longEvent := "这是一个非常非常长的事件，它毫无疑问会跨越我们设定的分片边界。" // 31
		secondEvent := "第二个事件。"                        // 6
		buffer := NewContentBuffer(WithMaxChars(15), WithOverlapCharNum(5))
		acc := newResultAccumulator(t, buffer)

		buffer.Write([]byte(longEvent))
		buffer.Write([]byte(secondEvent))
		acc.flushAndAccumulate()

		expectedChunks := []string{
			"这是一个非常非常长的事件，它毫",
			"事件，它毫无疑问会跨越我们设定",
			"越我们设定的分片边界。第二个事",
			"。第二个事件。",
		}
		acc.check(2, expectedChunks)
	})

	// A single write creates multiple chunks
	t.Run("Complex/SingleWriteCreatesMultipleChunks", func(t *testing.T) {
		buffer := NewContentBuffer(WithMaxChars(5), WithOverlapCharNum(2))
		acc := newResultAccumulator(t, buffer)

		buffer.Write([]byte("ABCDEFGHIJKL"))
		acc.flushAndAccumulate()

		expectedChunks := []string{"ABCDE", "DEFGH", "GHIJK", "JKL"}
		acc.check(1, expectedChunks)
	})

	// Multi-byte character boundary test
	t.Run("Complex/MultiByteCharacters", func(t *testing.T) {
		buffer := NewContentBuffer(WithMaxChars(6), WithOverlapCharNum(2))
		acc := newResultAccumulator(t, buffer)

		buffer.Write([]byte("测试中文字符"))
		buffer.Write([]byte("码"))
		buffer.Write([]byte("继续测试多字节"))
		acc.flushAndAccumulate()

		for i, chunk := range acc.allChunks {
			assert.True(t, utf8.ValidString(chunk), "Chunk %d should be valid UTF-8: %q", i, chunk)
		}

		expectedChunks := []string{"测试中文字符", "字符码继续测", "续测试多字节", "字节"}
		acc.check(3, expectedChunks)
	})

	// Write exactly to a boundary
	t.Run("EdgeCase/WriteExactlyToBoundary", func(t *testing.T) {
		buffer := NewContentBuffer(WithMaxChars(10), WithOverlapCharNum(2))
		acc := newResultAccumulator(t, buffer)

		buffer.Write([]byte("0123456789"))
		buffer.Write([]byte("A"))
		acc.flushAndAccumulate()

		expectedChunks := []string{"0123456789", "89A"}
		acc.check(2, expectedChunks)
	})

	// Successive writes and compaction
	t.Run("EdgeCase/SuccessiveWritesAndCompaction", func(t *testing.T) {
		buffer := NewContentBuffer(WithMaxChars(10), WithOverlapCharNum(3))
		acc := newResultAccumulator(t, buffer)

		buffer.Write([]byte("0123456789"))
		buffer.Write([]byte("ABCDEFG"))
		buffer.Write([]byte("HIJ"))
		acc.flushAndAccumulate()

		expectedChunks := []string{"0123456789", "789ABCDEFG", "EFGHIJ"}
		acc.check(3, expectedChunks)
	})

	// Maximal legal overlap
	t.Run("EdgeCase/MaximalLegalOverlap", func(t *testing.T) {
		buffer := NewContentBuffer(WithMaxChars(6), WithOverlapCharNum(5))
		acc := newResultAccumulator(t, buffer)

		buffer.Write([]byte("123456ABC"))
		acc.flushAndAccumulate()

		expectedChunks := []string{"123456", "23456A", "3456AB", "456ABC", "56ABC"}
		acc.check(1, expectedChunks)
	})

	// First chunk is smaller than the overlap number
	t.Run("EdgeCase/FirstChunkSmallerThanOverlap", func(t *testing.T) {
		buffer := NewContentBuffer(WithMaxChars(10), WithOverlapCharNum(5))
		acc := newResultAccumulator(t, buffer)

		buffer.Write([]byte("ABC"))
		buffer.Write([]byte("DEFGHIJKLM"))
		buffer.Write([]byte("NOPQRSTUVW"))
		acc.flushAndAccumulate()

		expectedChunks := []string{"ABCDEFGHIJ", "FGHIJKLMNO", "KLMNOPQRST", "PQRSTUVW"}
		acc.check(3, expectedChunks)
	})

	// Single-rune events with overlap
	t.Run("EdgeCase/SingleRuneEventsWithOverlap", func(t *testing.T) {
		buffer := NewContentBuffer(WithMaxChars(3), WithOverlapCharNum(1))
		acc := newResultAccumulator(t, buffer)

		buffer.Write([]byte("A"))
		buffer.Write([]byte("B"))
		buffer.Write([]byte("C"))
		buffer.Write([]byte("D"))
		buffer.Write([]byte("E"))
		buffer.Write([]byte("F"))
		buffer.Write([]byte("G"))
		acc.flushAndAccumulate()

		expectedChunks := []string{"ABC", "CDE", "EFG", "G"}
		acc.check(7, expectedChunks)
	})

	// Test event counting with empty and nil writes
	t.Run("EdgeCase/EventCountingWithEmptyAndNilWrites", func(t *testing.T) {
		t.Run("MixedWithData", func(t *testing.T) {
			buffer := NewContentBuffer(WithMaxChars(10), WithOverlapCharNum(2))
			acc := newResultAccumulator(t, buffer)

			buffer.Write([]byte("AB"))
			buffer.Write([]byte(""))
			buffer.Write(nil)
			buffer.Write([]byte("CD"))
			acc.flushAndAccumulate()

			expectedChunks := []string{"ABCD"}
			acc.check(4, expectedChunks)
		})

		t.Run("OnlyEmptyAndNil", func(t *testing.T) {
			buffer := NewContentBuffer(WithMaxChars(10), WithOverlapCharNum(2))
			acc := newResultAccumulator(t, buffer)

			buffer.Write([]byte(""))
			buffer.Write(nil)
			acc.flushAndAccumulate()

			acc.check(2, nil)
		})

		t.Run("MixedWithChunking", func(t *testing.T) {
			buffer := NewContentBuffer(WithMaxChars(4), WithOverlapCharNum(2))
			acc := newResultAccumulator(t, buffer)

			buffer.Write([]byte("AB")) // Event 1
			buffer.Write([]byte(""))   // Event 2
			buffer.Write([]byte("CD")) // Event 3,"ABCD"
			buffer.Write([]byte("EF")) // Event 4,"CDEF"
			buffer.Write(nil)          // Event 5
			buffer.Write([]byte("GH")) // Event 6,"EFGH"
			acc.flushAndAccumulate()

			expectedChunks := []string{"ABCD", "CDEF", "EFGH", "GH"}
			acc.check(6, expectedChunks)
		})
	})

	// Test event counting logic with different overlap strategies
	t.Run("Feature/EventCountingLogic", func(t *testing.T) {
		// Scenario 1: Default behavior, event count in overlap region is delayed
		t.Run("DelayedCountingInOverlap", func(t *testing.T) {
			buffer := NewContentBuffer(WithMaxChars(10), WithOverlapCharNum(3))
			buffer.overlapCountDelayed = true // Explicitly set to default
			acc := newResultAccumulator(t, buffer)

			buffer.Write([]byte("0123456")) // Event 1 - ends before overlap zone
			buffer.Write([]byte("78"))      // Event 2 - starts in overlap zone
			buffer.Write([]byte("9A"))      // Event 3 - starts in overlap zone, triggers chunk
			acc.accumulate()                // Get the first chunk

			assert.Equal(t, 1, acc.totalEvents, "First chunk should only count events fully outside the overlap zone")
			assert.Equal(t, []string{"0123456789"}, acc.allChunks)

			acc.flushAndAccumulate() // Flushes the rest ("789A")
			assert.Equal(t, 3, acc.totalEvents, "After flush, all events should be counted")
			assert.Equal(t, []string{"0123456789", "789A"}, acc.allChunks, "Second chunk content is incorrect")
		})

		// Scenario 2: Delayed counting is off, events in overlap are counted immediately
		t.Run("ImmediateCountingInOverlap", func(t *testing.T) {
			buffer := NewContentBuffer(WithMaxChars(10), WithOverlapCharNum(3))
			buffer.overlapCountDelayed = false // Disable delay
			acc := newResultAccumulator(t, buffer)

			buffer.Write([]byte("0123456")) // Event 1
			buffer.Write([]byte("78"))      // Event 2 (in overlap, should be counted immediately)
			buffer.Write([]byte("9A"))      // Event 3 (triggers chunk)
			acc.accumulate()                // Get the first chunk

			assert.Equal(t, 2, acc.totalEvents, "First two events should be counted immediately in the first chunk")
			assert.Equal(t, []string{"0123456789"}, acc.allChunks)

			acc.flushAndAccumulate()
			assert.Equal(t, 3, acc.totalEvents, "Total events after flush should be 3")
			assert.Equal(t, []string{"0123456789", "789A"}, acc.allChunks, "Second chunk content is incorrect")
		})

		// Scenario 3: With no overlap, the overlapCountDelayed flag has no effect
		t.Run("NoOverlapIgnoresFlag", func(t *testing.T) {
			for _, delayed := range []bool{true, false} {
				t.Run(fmt.Sprintf("delayed=%v", delayed), func(t *testing.T) {
					buffer := NewContentBuffer(WithMaxChars(10), WithOverlapCharNum(0))
					buffer.overlapCountDelayed = delayed
					acc := newResultAccumulator(t, buffer)

					buffer.Write([]byte("0123456789")) // Event 1, creates chunk
					buffer.Write([]byte("ABC"))        // Event 2
					acc.flushAndAccumulate()

					expectedChunks := []string{"0123456789", "ABC"}
					acc.check(2, expectedChunks)
				})
			}
		})

		// Write fills a chunk exactly
		t.Run("WriteFillsChunkExactly", func(t *testing.T) {
			t.Run("Delayed", func(t *testing.T) {
				buffer := NewContentBuffer(WithMaxChars(5), WithOverlapCharNum(2))
				buffer.overlapCountDelayed = true
				acc := newResultAccumulator(t, buffer)

				buffer.Write([]byte("ABC")) // Event 1 - ends before overlap
				buffer.Write([]byte("DE"))  // Event 2 - is the overlap, count delayed
				acc.accumulate()

				assert.Equal(t, 1, acc.totalEvents, "Event count for the event filling the chunk (Event 2) should be delayed")
				assert.Equal(t, []string{"ABCDE"}, acc.allChunks)

				buffer.Write([]byte("F"))
				acc.flushAndAccumulate()
				acc.check(3, []string{"ABCDE", "DEF"})
			})

			t.Run("Immediate", func(t *testing.T) {
				buffer := NewContentBuffer(WithMaxChars(5), WithOverlapCharNum(2))
				buffer.overlapCountDelayed = false
				acc := newResultAccumulator(t, buffer)

				buffer.Write([]byte("ABC"))
				buffer.Write([]byte("DE"))
				acc.accumulate()

				assert.Equal(t, 1, acc.totalEvents)
				assert.Equal(t, []string{"ABCDE"}, acc.allChunks)

				buffer.Write([]byte("F"))
				acc.flushAndAccumulate()
				acc.check(3, []string{"ABCDE", "DEF"})
			})
		})

		// Flush with a partial overlap
		t.Run("FlushWithPartialOverlap", func(t *testing.T) {
			t.Run("Delayed", func(t *testing.T) {
				buffer := NewContentBuffer(WithMaxChars(10), WithOverlapCharNum(4))
				buffer.overlapCountDelayed = true
				acc := newResultAccumulator(t, buffer)

				buffer.Write([]byte("0123456789"))
				acc.accumulate()
				assert.Equal(t, 0, acc.totalEvents, "First chunk count should be 0 as the event crosses into overlap")
				assert.Equal(t, []string{"0123456789"}, acc.allChunks)

				buffer.Write([]byte("AB"))
				acc.flushAndAccumulate()
				acc.check(2, []string{"0123456789", "6789AB"})
			})

			t.Run("Immediate", func(t *testing.T) {
				buffer := NewContentBuffer(WithMaxChars(10), WithOverlapCharNum(4))
				buffer.overlapCountDelayed = false
				acc := newResultAccumulator(t, buffer)

				buffer.Write([]byte("0123456789"))
				acc.accumulate()
				assert.Equal(t, 0, acc.totalEvents)

				buffer.Write([]byte("AB"))
				acc.flushAndAccumulate()
				acc.check(2, []string{"0123456789", "6789AB"})
			})
		})

		// Empty events in the overlap zone
		t.Run("EmptyEventsInOverlap", func(t *testing.T) {
			t.Run("Delayed", func(t *testing.T) {
				buffer := NewContentBuffer(WithMaxChars(5), WithOverlapCharNum(2))
				buffer.overlapCountDelayed = true
				acc := newResultAccumulator(t, buffer)

				buffer.Write([]byte("ABCDE"))
				acc.accumulate()
				assert.Equal(t, 0, acc.totalEvents, "First chunk count should be 0 as the event crosses into overlap")
				assert.Equal(t, []string{"ABCDE"}, acc.allChunks)

				buffer.Write([]byte(""))
				buffer.Write(nil)
				buffer.Write([]byte("F"))
				acc.flushAndAccumulate()

				acc.check(4, []string{"ABCDE", "DEF"})
			})

			t.Run("Immediate", func(t *testing.T) {
				buffer := NewContentBuffer(WithMaxChars(5), WithOverlapCharNum(2))
				buffer.overlapCountDelayed = false
				acc := newResultAccumulator(t, buffer)

				buffer.Write([]byte("ABCDE"))
				acc.accumulate()
				assert.Equal(t, 0, acc.totalEvents)
				assert.Equal(t, []string{"ABCDE"}, acc.allChunks)

				buffer.Write([]byte(""))
				buffer.Write(nil)
				buffer.Write([]byte("F"))
				acc.flushAndAccumulate()

				acc.check(4, []string{"ABCDE", "DEF"})
			})
		})
	})

	// New: State machine interaction tests
	t.Run("Boundary/StateInteractions", func(t *testing.T) {
		t.Run("RepeatedFlushIsIdempotent", func(t *testing.T) {
			buffer := NewContentBuffer(WithMaxChars(10), WithOverlapCharNum(2))
			acc := newResultAccumulator(t, buffer)

			buffer.Write([]byte("ABC"))
			buffer.Flush()
			buffer.Flush()

			acc.accumulate()
			assert.Equal(t, 1, acc.totalEvents)
			assert.Equal(t, []string{"ABC"}, acc.allChunks)

			buffer.Flush()
			result := buffer.GetCompletedResult()
			assert.Equal(t, 0, result.CompletedEvents)
			assert.Empty(t, result.Chunks)
		})

		t.Run("GetResultBetweenEveryWrite", func(t *testing.T) {
			buffer := NewContentBuffer(WithMaxChars(5), WithOverlapCharNum(2))
			acc := newResultAccumulator(t, buffer)

			assert.Equal(t, 0, acc.totalEvents)
			assert.Empty(t, acc.allChunks)
			buffer.Write([]byte("ABC")) // ABC
			acc.accumulate()            // empty
			assert.Equal(t, 0, acc.totalEvents)

			buffer.Write([]byte("DEF")) // ABCDE DEF
			acc.accumulate()            // "ABCDE"
			assert.Equal(t, []string{"ABCDE"}, acc.allChunks)
			assert.Equal(t, 1, acc.totalEvents)

			buffer.Write([]byte("GHI"))              // ABCDE DEFGH GHI
			acc.accumulate()                         // "DEFGH"
			acc.check(2, []string{"ABCDE", "DEFGH"}) // Events 2 and 3 are counted now

			buffer.Flush()
			acc.accumulate()
			acc.check(3, []string{"ABCDE", "DEFGH", "GHI"})
		})
	})

	// New: Destructive test for UTF-8
	t.Run("Boundary/InvalidUTF8Sequence", func(t *testing.T) {
		buffer := NewContentBuffer(WithMaxChars(10), WithOverlapCharNum(0))
		acc := newResultAccumulator(t, buffer)

		buffer.Write([]byte("你好\xff世界")) // Event 1
		acc.flushAndAccumulate()

		expectedChunks := []string{"你好世界"}
		acc.check(1, expectedChunks)
		assert.True(t, utf8.ValidString(acc.allChunks[0]), "Chunk content must be valid UTF-8")
	})

	// New: Blackbox test with a complex sequence of calls
	t.Run("Blackbox/ComplexInteractionSequence", func(t *testing.T) {
		buffer := NewContentBuffer(WithMaxChars(8), WithOverlapCharNum(3))
		acc := newResultAccumulator(t, buffer)

		buffer.Write([]byte("事件一"))
		acc.accumulate()
		acc.check(0, nil)

		buffer.Write([]byte("然后是事件二"))
		acc.accumulate()
		acc.check(1, []string{"事件一然后是事件"})

		buffer.Write([]byte("三"))
		buffer.Flush()
		acc.accumulate()
		acc.check(3, []string{"事件一然后是事件", "是事件二三"})
	})
}
