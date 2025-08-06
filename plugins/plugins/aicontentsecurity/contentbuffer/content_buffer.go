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

// chunkBoundary stores metadata for a single data chunk.
type chunkBoundary struct {
	start      int
	end        int
	writeTimes int
}

// SplitResult encapsulates the result of a split operation.
type SplitResult struct {
	Chunks          []string
	CompletedEvents int
}

// ContentBuffer implements the logic for streaming splitting by the number of runes.
type ContentBuffer struct {
	maxChars            int // The maximum number of runes per chunk.
	overlapCharNum      int // The number of overlapping runes between chunks.
	overlapCountDelayed bool

	buffer      []byte          // Internal byte buffer.
	boundaries  []chunkBoundary // Boundary information of completed chunks.
	currStart   int             // The starting byte index of the chunk currently being built.
	currChars   int             // The number of runes in the chunk currently being built.
	outputIndex int             // The index of the next chunk to be returned.

	currEventCounter    int // Event counter for the current chunk.
	overlapEventCounter int // Event counter for the current overlap area (delays the next chunk's count).

	initialCapacity int     // The initial capacity of the buffer.
	shrinkFactor    float64 // The threshold factor that triggers buffer shrinkage.
	resizeFactor    float64

	counter CharCounter
}

// BufferOption is a function type for configuring a ContentBuffer.
type BufferOption func(*ContentBuffer)

// WithInitialCapacity sets the initial capacity of the buffer.
func WithInitialCapacity(capacity int) BufferOption {
	return func(c *ContentBuffer) {
		if capacity > 0 {
			c.initialCapacity = capacity
		}
	}
}

func WithMaxChars(maxChars int) BufferOption {
	return func(c *ContentBuffer) {
		c.maxChars = maxChars
	}
}

func WithOverlapCharNum(overlapCharNum int) BufferOption {
	return func(c *ContentBuffer) {
		c.overlapCharNum = overlapCharNum
	}
}

// NewContentBuffer creates and initializes a new ContentBuffer.
func NewContentBuffer(opts ...BufferOption) *ContentBuffer {
	c := &ContentBuffer{
		maxChars:            100,
		overlapCharNum:      0,
		boundaries:          make([]chunkBoundary, 0, 64),
		counter:             Utf8RuneCounter{},
		currStart:           0,
		currChars:           0,
		overlapCountDelayed: true,
		initialCapacity:     2049,
		shrinkFactor:        2,
		resizeFactor:        1.3,
	}

	for _, opt := range opts {
		opt(c)
	}

	c.buffer = make([]byte, 0, c.initialCapacity)
	return c
}

func (c *ContentBuffer) inOverlap() bool {
	return c.maxChars-c.currChars < c.overlapCharNum
}

// startNewChunk finalizes the current chunk and starts a new one.
func (c *ContentBuffer) startNewChunk(disableOverlap bool) {
	if c.currChars == 0 {
		return
	}
	end := len(c.buffer)
	c.boundaries = append(c.boundaries, chunkBoundary{
		start:      c.currStart,
		end:        end,
		writeTimes: c.currEventCounter,
	})
	c.currEventCounter = c.overlapEventCounter
	c.overlapEventCounter = 0

	if c.overlapCharNum > 0 && !disableOverlap {
		overlapStart := c.counter.TailStartIndex(c.buffer, c.overlapCharNum) // Better implementation?
		c.currStart = overlapStart
		c.currChars = c.overlapCharNum
	} else {
		c.currStart = len(c.buffer)
		c.currChars = 0
	}
}

// shrinkIfNeeded checks if the buffer's capacity needs to be reduced and performs the shrink if necessary.
func (c *ContentBuffer) shrinkIfNeeded() {
	currentCap := cap(c.buffer)
	currentLen := len(c.buffer)

	// If the buffer is empty and its capacity is greater than the initial capacity,
	// shrink it back to the initial capacity.
	if currentLen == 0 && currentCap > c.initialCapacity {
		c.buffer = make([]byte, 0, c.initialCapacity)
		return
	}

	// Only consider shrinking when the capacity is greater than the initial capacity.
	if currentCap > c.initialCapacity {
		targetShrinkCapacity := int(float64(currentLen) * c.shrinkFactor)
		if targetShrinkCapacity < c.initialCapacity {
			targetShrinkCapacity = c.initialCapacity
		}

		if currentCap > targetShrinkCapacity {
			newBuf := make([]byte, currentLen, int(float64(currentLen)*c.resizeFactor))
			copy(newBuf, c.buffer)
			c.buffer = newBuf
		}
	}
}

// Write adds data to the buffer.
func (c *ContentBuffer) Write(data []byte) {
	i := 0
	for i < len(data) {
		_, size, err := c.counter.DecodeChar(data[i:])
		if err != nil {
			// As a fault-tolerance strategy, skip invalid UTF-8 bytes.
			i++
			continue
		}

		c.buffer = append(c.buffer, data[i:i+size]...)
		c.currChars++
		i += size

		if c.currChars == c.maxChars {
			// Processing is complete and the buffered text has reached the upper limit.
			c.startNewChunk(false)
			if i == len(data) && c.overlapCharNum == 0 {
				c.boundaries[len(c.boundaries)-1].writeTimes++
				return
			}
		}
	}

	if c.inOverlap() && c.overlapCountDelayed {
		c.overlapEventCounter++
	} else {
		c.currEventCounter++
	}
}

// Flush commits the currently ongoing chunk.
func (c *ContentBuffer) Flush() {
	if c.currChars > 0 {
		// Avoid missing event counts.
		c.currEventCounter += c.overlapEventCounter
		c.startNewChunk(true)
	}
}

// GetCompletedResult returns all completed chunks that have not yet been retrieved.
func (c *ContentBuffer) GetCompletedResult() SplitResult {
	if c.outputIndex >= len(c.boundaries) {
		if c.currChars == 0 {
			counter := c.currEventCounter
			c.currEventCounter = 0
			return SplitResult{Chunks: nil, CompletedEvents: counter}
		}
		return SplitResult{Chunks: nil}
	}

	newBoundaries := c.boundaries[c.outputIndex:]
	chunks := make([]string, len(newBoundaries))

	eventCount := 0
	for i, boundary := range newBoundaries {
		chunks[i] = string(c.buffer[boundary.start:boundary.end])
		eventCount += boundary.writeTimes
	}

	// Clean completed result
	if c.currStart > 0 {
		remainingSize := len(c.buffer) - c.currStart
		copy(c.buffer, c.buffer[c.currStart:])
		c.buffer = c.buffer[:remainingSize]
		c.boundaries = c.boundaries[:0]
		c.currStart = 0
		c.outputIndex = 0
	}
	c.shrinkIfNeeded()

	return SplitResult{Chunks: chunks, CompletedEvents: eventCount}
}
