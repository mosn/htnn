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
	"math/rand"
	"strings"
	"testing"
	"time"
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
	t.Run("Basic/NoOverlap", func(t *testing.T) {
		buffer := NewContentBuffer(WithMaxChars(10), WithOverlapCharNum(0))
		acc := newResultAccumulator(t, buffer)

		buffer.Write([]byte("EVA1"))
		buffer.Write([]byte("EVB2"))
		buffer.Write([]byte("EVC3"))
		acc.flushAndAccumulate()

		expectedChunks := []string{"EVA1EVB2EV", "C3"}
		acc.check(3, expectedChunks)
	})

	t.Run("Basic/WithOverlap", func(t *testing.T) {
		buffer := NewContentBuffer(WithMaxChars(10), WithOverlapCharNum(3))
		acc := newResultAccumulator(t, buffer)

		buffer.Write([]byte("0123456789"))
		buffer.Write([]byte("ABCDEFGHIJ"))
		acc.flushAndAccumulate()

		expectedChunks := []string{"0123456789", "789ABCDEFG", "EFGHIJ"}
		acc.check(2, expectedChunks)
	})

	t.Run("Complex/SingleLongEventWithOverlap", func(t *testing.T) {
		longEvent := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" // 31 runes
		secondEvent := "bbbbbb"                        // 6 runes
		buffer := NewContentBuffer(WithMaxChars(15), WithOverlapCharNum(5))
		acc := newResultAccumulator(t, buffer)

		buffer.Write([]byte(longEvent))
		buffer.Write([]byte(secondEvent))
		acc.flushAndAccumulate()

		expectedChunks := []string{
			"aaaaaaaaaaaaaaa",
			"aaaaaaaaaaaaaaa",
			"aaaaaaaaaaabbbb",
			"abbbbbb",
		}
		acc.check(2, expectedChunks)
	})

	t.Run("Complex/SingleWriteCreatesMultipleChunks", func(t *testing.T) {
		buffer := NewContentBuffer(WithMaxChars(5), WithOverlapCharNum(2))
		acc := newResultAccumulator(t, buffer)

		buffer.Write([]byte("ABCDEFGHIJKL"))
		acc.flushAndAccumulate()

		expectedChunks := []string{"ABCDE", "DEFGH", "GHIJK", "JKL"}
		acc.check(1, expectedChunks)
	})

	t.Run("Complex/MultiByteCharacters", func(t *testing.T) {
		buffer := NewContentBuffer(WithMaxChars(6), WithOverlapCharNum(2))
		acc := newResultAccumulator(t, buffer)

		buffer.Write([]byte("ABCDEF"))
		buffer.Write([]byte("G"))
		buffer.Write([]byte("HIJKLMN"))
		acc.flushAndAccumulate()

		for i, chunk := range acc.allChunks {
			assert.True(t, utf8.ValidString(chunk), "Chunk %d should be valid UTF-8: %q", i, chunk)
		}

		expectedChunks := []string{"ABCDEF", "EFGHIJ", "IJKLMN", "MN"}
		acc.check(3, expectedChunks)
	})

	t.Run("EdgeCase/WriteExactlyToBoundary", func(t *testing.T) {
		buffer := NewContentBuffer(WithMaxChars(10), WithOverlapCharNum(2))
		acc := newResultAccumulator(t, buffer)

		buffer.Write([]byte("0123456789"))
		buffer.Write([]byte("A"))
		acc.flushAndAccumulate()

		expectedChunks := []string{"0123456789", "89A"}
		acc.check(2, expectedChunks)
	})

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

	t.Run("EdgeCase/MaximalLegalOverlap", func(t *testing.T) {
		buffer := NewContentBuffer(WithMaxChars(6), WithOverlapCharNum(5))
		acc := newResultAccumulator(t, buffer)

		buffer.Write([]byte("123456ABC"))
		acc.flushAndAccumulate()

		expectedChunks := []string{"123456", "23456A", "3456AB", "456ABC", "56ABC"}
		acc.check(1, expectedChunks)
	})

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

	t.Run("Feature/EventCountingLogic", func(t *testing.T) {
		t.Run("DelayedCountingInOverlap", func(t *testing.T) {
			buffer := NewContentBuffer(WithMaxChars(10), WithOverlapCharNum(3))
			buffer.overlapCountDelayed = true
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

	t.Run("Boundary/AdvancedUTF8", func(t *testing.T) {
		t.Run("IncompleteMultiByteAtEnd", func(t *testing.T) {
			buffer := NewContentBuffer(WithMaxChars(10), WithOverlapCharNum(0))
			acc := newResultAccumulator(t, buffer)

			buffer.Write([]byte("AB\xe4\xbd"))
			acc.flushAndAccumulate()

			expectedChunks := []string{"AB"}
			acc.check(1, expectedChunks)
			assert.True(t, utf8.ValidString(acc.allChunks[0]))
		})

		t.Run("InvalidByteAtChunkBoundary", func(t *testing.T) {
			buffer := NewContentBuffer(WithMaxChars(5), WithOverlapCharNum(2))
			acc := newResultAccumulator(t, buffer)

			buffer.Write([]byte("1234\xff56789"))
			acc.flushAndAccumulate()

			expectedChunks := []string{"12345", "45678", "789"}
			acc.check(1, expectedChunks)
			assert.True(t, utf8.ValidString(acc.allChunks[0]))
			assert.True(t, utf8.ValidString(acc.allChunks[1]))
		})

		t.Run("InvalidByteInOverlap", func(t *testing.T) {
			buffer := NewContentBuffer(WithMaxChars(10), WithOverlapCharNum(4))
			acc := newResultAccumulator(t, buffer)

			buffer.Write([]byte("01234567\xff89ABCDEFG"))
			acc.flushAndAccumulate()

			expectedChunks := []string{"0123456789", "6789ABCDEF", "CDEFG"}
			acc.check(1, expectedChunks)
		})

		t.Run("IncompleteSequenceAcrossWrites", func(t *testing.T) {
			buffer := NewContentBuffer(WithMaxChars(20), WithOverlapCharNum(0))
			acc := newResultAccumulator(t, buffer)

			buffer.Write([]byte("ABCD\xe4\xbd"))
			buffer.Write([]byte("\xa0EFGH"))

			acc.flushAndAccumulate()

			expectedChunks := []string{"ABCDEFGH"}
			acc.check(2, expectedChunks)
		})
	})

}

func stripInvalidUTF8Bytes(s string) string {
	if utf8.ValidString(s) {
		return s
	}

	var result strings.Builder
	result.Grow(len(s))
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			i += size
			continue
		}
		result.WriteRune(r)
		i += size
	}
	return result.String()
}

func FuzzContentBuffer(f *testing.F) {
	f.Add(10, 3, "Hello, world!", true)
	f.Add(10, 0, "Hello, world!", true)
	f.Add(20, 5, "AB\xe4\xbdCD", false) // Contains an incomplete UTF-8 sequence
	f.Add(5, 2, "\xff\xfe\xfd", true)   // Contains invalid bytes
	f.Add(15, 4, "这是中文事件\n另一行事件", true) // Contains multibyte characters and multiple events
	f.Add(8, 2, "1234567890", false)
	f.Add(12, 3, "abcdefghijklmnopqrstuvwxyz", true)
	f.Add(10, 3, "ababababababab", true)     // Repetitive string to test overlap logic
	f.Add(10, 3, "", true)                   // Input is an empty string
	f.Add(100, 20, "short", false)           // Input data is much smaller than maxChars
	f.Add(10, 9, "edge case", true)          // Edge case where overlapChars is close to the maxChars limit
	f.Add(5, 2, "tiny", false)               // Input data is smaller than maxChars but larger than overlapChars
	f.Add(10, 8, "123\xe4\xbd\xffENG", true) // Contains a mix of valid and invalid UTF-8 sequences

	// The Fuzz function receives inputs from the seed corpus and the fuzzing engine to execute the test logic.
	f.Fuzz(func(t *testing.T, maxChars int, overlapChars int, input string, overlapCountDelayed bool) {
		// Filter out invalid parameter combinations, as ContentBuffer does not support these cases.
		if maxChars <= 0 || overlapChars < 0 || overlapChars >= maxChars {
			return
		}
		t.Logf("[Fuzz test input] maxChars=%d, overlapChars=%d, overlapDelayed=%v, input=%q (%d bytes)",
			maxChars, overlapChars, overlapCountDelayed, input, len(input))

		r := rand.New(rand.NewSource(time.Now().UnixNano()))

		var events []string
		remainingInput := input
		// randomly split the input string into multiple small events.
		if len(remainingInput) > 0 {
			for utf8.RuneCountInString(remainingInput) > 1 {
				var cutPoints []int
				// Find all valid rune boundaries to split the string
				tempStr := remainingInput
				for i := range tempStr {
					if i > 0 {
						cutPoints = append(cutPoints, i)
					}
				}
				if len(cutPoints) == 0 {
					break
				}
				cutIndexInBytes := cutPoints[r.Intn(len(cutPoints))]

				event := remainingInput[:cutIndexInBytes]
				events = append(events, event)
				remainingInput = remainingInput[cutIndexInBytes:]
			}
			events = append(events, remainingInput)
		} else {
			events = append(events, "")
		}

		buffer := NewContentBuffer(WithMaxChars(maxChars), WithOverlapCharNum(overlapChars))
		buffer.overlapCountDelayed = overlapCountDelayed
		acc := newResultAccumulator(t, buffer)

		// Write the split events one by one into the buffer.
		realCount := 0
		for _, event := range events {
			realCount++
			buffer.Write([]byte(event))
		}

		acc.flushAndAccumulate()

		var reconstructed strings.Builder
		allChunks := acc.allChunks
		// Since the buffer cleans up invalid UTF-8 bytes, we need a clean version of the original input for comparison.
		cleanOriginalInput := stripInvalidUTF8Bytes(input)
		for i, chunk := range allChunks {
			assert.True(t, utf8.ValidString(chunk), "Fuzzing: Chunk %d should be valid UTF-8: %q", i, chunk)
		}

		// attempt to reconstruct the original string from the chunks, accounting for the overlap.
		if len(allChunks) > 0 {
			reconstructed.WriteString(allChunks[0])
			for i := 1; i < len(allChunks); i++ {
				prevChunk := allChunks[i-1]
				currentChunk := allChunks[i]

				if len(currentChunk) == 0 {
					continue
				}

				if overlapChars == 0 {
					reconstructed.WriteString(currentChunk)
					continue
				}

				// Based on the buffer's overlap logic, calculate the expected overlap.
				var intendedOverlap string
				prevRunes := []rune(prevChunk)

				if len(prevRunes) > overlapChars {
					overlapStartIndex := len(prevRunes) - overlapChars
					intendedOverlap = string(prevRunes[overlapStartIndex:])
				} else {
					// If the previous chunk is shorter than the overlap count, the whole chunk is considered the overlap.
					intendedOverlap = prevChunk
				}

				// Remove the overlapping part from the current chunk and then append the rest to the reconstructed string.
				if strings.HasPrefix(currentChunk, intendedOverlap) {
					reconstructed.WriteString(currentChunk[len(intendedOverlap):])
				} else {
					reconstructed.WriteString(currentChunk)
				}
			}
		}

		reconstructedStr := reconstructed.String()
		// The reconstructed string should be identical to the original input after it has been cleaned of invalid UTF-8 bytes.
		assert.Equal(t, cleanOriginalInput, reconstructedStr, "Fuzzing: Reconstructed string should match original input")
	})
}
