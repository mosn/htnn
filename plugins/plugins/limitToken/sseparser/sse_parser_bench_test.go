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
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"
)

func generateSSEEvents(numEvents int) []byte {
	var buf bytes.Buffer

	for i := 0; i < numEvents; i++ {
		event := fmt.Sprintf("id: %d\nevent: message\ndata: This is a test event number %d to increase the payload size significantly.\n\n", i, i)
		buf.WriteString(event)
	}

	return buf.Bytes()
}

// GlobalMemorySampler handles memory sampling in a separate goroutine at fixed time intervals.
type GlobalMemorySampler struct {
	samples  []uint64
	stopChan chan struct{}
	wg       sync.WaitGroup
	mu       sync.RWMutex
}

func NewGlobalMemorySampler() *GlobalMemorySampler {
	return &GlobalMemorySampler{
		samples:  make([]uint64, 0, 1024), // Pre-allocate some capacity
		stopChan: make(chan struct{}),
	}
}

// Start launches the background sampling goroutine.
func (gs *GlobalMemorySampler) Start(interval time.Duration) {
	gs.wg.Add(1)
	go func() {
		defer gs.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				gs.mu.Lock()
				gs.samples = append(gs.samples, m.Alloc)
				gs.mu.Unlock()
			case <-gs.stopChan:
				return
			}
		}
	}()
}

// Stop signals the sampling goroutine to terminate and waits for it to finish.
func (gs *GlobalMemorySampler) Stop() {
	close(gs.stopChan)
	gs.wg.Wait()
}

// GetStats calculates and returns statistics from the collected samples.
func (gs *GlobalMemorySampler) GetStats() (peakAlloc, minAlloc, avgAlloc, stdDev uint64) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	if len(gs.samples) == 0 {
		return 0, 0, 0, 0
	}

	minAlloc = ^uint64(0)
	var totalAlloc uint64
	sampleCount := uint64(len(gs.samples))

	for _, alloc := range gs.samples {
		if alloc > peakAlloc {
			peakAlloc = alloc
		}
		if alloc < minAlloc {
			minAlloc = alloc
		}
		totalAlloc += alloc
	}

	if minAlloc == ^uint64(0) {
		minAlloc = 0
	}

	avgAlloc = totalAlloc / sampleCount

	var variance float64
	for _, v := range gs.samples {
		diff := float64(v) - float64(avgAlloc)
		variance += diff * diff
	}
	stdDev = uint64(math.Sqrt(variance / float64(sampleCount)))

	return peakAlloc, minAlloc, avgAlloc, stdDev
}

// BenchmarkConfig allows configuration of benchmark behavior.
type BenchmarkConfig struct {
	EnableMemorySampling bool
	NormalWriteSize      int
	BurstInterval        int
	BurstWriteSize       int
	PruneEventSizeLimit  int
	TotalOperations      int

	EnableFragmentation bool
	MaxFragmentSize     int
}

// DefaultBenchmarkConfig defines settings for HIGH CONCURRENCY, LOW unit load benchmarks.
func DefaultBenchmarkConfig() BenchmarkConfig {
	return BenchmarkConfig{
		EnableMemorySampling: true,
		NormalWriteSize:      1,
		BurstInterval:        100,
		BurstWriteSize:       200000,
		PruneEventSizeLimit:  50,
		TotalOperations:      200,
		EnableFragmentation:  false,
		MaxFragmentSize:      0,
	}
}

// DefaultBenchmarkConfigHigh defines settings for LOW CONCURRENCY, HIGH unit load benchmarks.
func DefaultBenchmarkConfigHigh() BenchmarkConfig {
	return BenchmarkConfig{
		EnableMemorySampling: true,
		NormalWriteSize:      10,
		BurstInterval:        3000,
		BurstWriteSize:       200000,
		PruneEventSizeLimit:  300,
		TotalOperations:      50000,
		EnableFragmentation:  false,
		MaxFragmentSize:      0,
	}
}

// appendFragmented receives a data chunk and feeds it to the parser in random fragments.
func appendFragmented(p *StreamEventParser, data []byte, maxFragmentSize int, r *rand.Rand) {
	offset := 0
	for offset < len(data) {
		fragmentSize := 1
		if maxFragmentSize > 1 {
			fragmentSize = r.Intn(maxFragmentSize-1) + 1
		}

		end := offset + fragmentSize
		if end > len(data) {
			end = len(data)
		}

		fragment := data[offset:end]
		p.Append(fragment)

		offset = end
	}
}

// runSSEParserBenchmark runs the benchmark with the new global time-based sampler.
func runSSEParserBenchmark(b *testing.B, pruneFunc func(p *StreamEventParser), config BenchmarkConfig, parallelism int) {
	normalChunk := generateSSEEvents(config.NormalWriteSize)
	burstChunk := generateSSEEvents(config.BurstWriteSize)

	b.ReportAllocs()

	var sampler *GlobalMemorySampler
	if config.EnableMemorySampling {
		sampler = NewGlobalMemorySampler()
		samplingInterval := 10 * time.Millisecond
		sampler.Start(samplingInterval)
	}

	var startMemStats runtime.MemStats
	runtime.ReadMemStats(&startMemStats)

	b.ResetTimer()
	b.SetParallelism(parallelism)
	b.RunParallel(func(pb *testing.PB) {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))

		for pb.Next() {
			parser := NewStreamEventParser()
			eventsSinceLastPrune := 0

			for j := 0; j < config.TotalOperations; j++ {
				isBurst := j > 0 && j%config.BurstInterval == 0

				var chunkToWrite []byte
				if isBurst {
					chunkToWrite = burstChunk
				} else {
					chunkToWrite = normalChunk
				}

				if config.EnableFragmentation && config.MaxFragmentSize > 0 {
					appendFragmented(parser, chunkToWrite, config.MaxFragmentSize, r)
				} else {
					parser.Append(chunkToWrite)
				}

				parsedCountInLoop := 0
				for {
					event, err := parser.Parse()
					if err != nil {
						b.Fatalf("Parse error: %v", err)
					}
					if event == nil {
						break
					}
					parsedCountInLoop++
				}

				eventsSinceLastPrune += parsedCountInLoop
				if eventsSinceLastPrune >= config.PruneEventSizeLimit {
					pruneFunc(parser)
					eventsSinceLastPrune = 0
				}
			}
		}
	})

	b.StopTimer()

	if config.EnableMemorySampling {
		sampler.Stop()
	}

	logGlobalResults(b, config, sampler, &startMemStats)
}

// logGlobalResults processes results from the global sampler and logs them.
func logGlobalResults(b *testing.B, config BenchmarkConfig, sampler *GlobalMemorySampler, startMemStats *runtime.MemStats) {
	var finalPeakAlloc, finalMinAlloc, finalAvgAlloc, finalStdDev uint64
	if config.EnableMemorySampling && sampler != nil {
		finalPeakAlloc, finalMinAlloc, finalAvgAlloc, finalStdDev = sampler.GetStats()
	}

	var endMemStats runtime.MemStats
	runtime.ReadMemStats(&endMemStats)

	totalAllocated := endMemStats.TotalAlloc - startMemStats.TotalAlloc
	totalGCPause := endMemStats.PauseTotalNs - startMemStats.PauseTotalNs
	gcCount := endMemStats.NumGC - startMemStats.NumGC

	var maxPause, avgPause, lastPause float64
	if gcCount > 0 {
		n := int(gcCount)
		if n > 256 {
			n = 256
		}

		var sum uint64
		for i := 0; i < n; i++ {
			idx := (startMemStats.NumGC + uint32(i)) % 256
			p := endMemStats.PauseNs[idx]
			if float64(p) > maxPause {
				maxPause = float64(p)
			}
			sum += p
		}
		avgPause = float64(sum) / float64(n)
		lastPause = float64(endMemStats.PauseNs[(endMemStats.NumGC-1)%256])
	}

	if config.EnableMemorySampling {
		b.Logf(
			"\n--- Global Memory Usage Profile (Time-based Sampling) ---"+
				"\nPeakAlloc: %.2f MB"+
				"\nAvgAlloc (Time-Weighted): %.2f MB"+
				"\nMinAlloc: %.2f MB"+
				"\nAvg StdDev: %.2f MB"+
				"\n--- Memory Pressure Profile ---"+
				"\nTotal Allocated: %.2f MB"+
				"\nGC Count: %d"+
				"\nTotal GC Pause: %.4f ms"+
				"\n--- Extended Memory Stats ---"+
				"\nHeapInuse: %.2f MB"+
				"\nHeapIdle: %.2f MB"+
				"\nHeapReleased: %.2f MB"+
				"\nHeapObjects: %d"+
				"\nMax GC Pause: %.4f ms"+
				"\nAvg GC Pause: %.4f ms"+
				"\nLast GC Pause: %.4f ms",
			float64(finalPeakAlloc)/1024/1024,
			float64(finalAvgAlloc)/1024/1024,
			float64(finalMinAlloc)/1024/1024,
			float64(finalStdDev)/1024/1024,
			float64(totalAllocated)/1024/1024,
			gcCount,
			float64(totalGCPause)/float64(time.Millisecond),
			float64(endMemStats.HeapInuse)/1024/1024,
			float64(endMemStats.HeapIdle)/1024/1024,
			float64(endMemStats.HeapReleased)/1024/1024,
			endMemStats.HeapObjects,
			maxPause/float64(time.Millisecond),
			avgPause/float64(time.Millisecond),
			lastPause/float64(time.Millisecond),
		)
	} else {
		b.Logf(
			"\n--- Basic Memory Stats (Sampling Disabled) ---"+
				"\nTotal Allocated: %.2f MB"+
				"\nGC Count: %d"+
				"\nTotal GC Pause: %.4f ms"+
				"\nHeapInuse: %.2f MB",
			float64(totalAllocated)/1024/1024,
			gcCount,
			float64(totalGCPause)/float64(time.Millisecond),
			float64(endMemStats.HeapInuse)/1024/1024,
		)
	}
}

const MAXPROCS = 40000

// --- Concurrent Fragmented Benchmark ---

func Benchmark_Prune_Concurrent_Fragmented(b *testing.B) {
	config := DefaultBenchmarkConfig()
	config.EnableFragmentation = true
	config.MaxFragmentSize = 32

	parallelism := runtime.GOMAXPROCS(0) * MAXPROCS
	runSSEParserBenchmark(b, func(p *StreamEventParser) {
		p.PruneParsedData()
	}, config, parallelism)
}

func Benchmark_Prune_HighLoad_Sequential_Fragmented(b *testing.B) {
	config := DefaultBenchmarkConfigHigh()
	config.EnableFragmentation = true
	config.MaxFragmentSize = 64

	runSSEParserBenchmark(b, func(p *StreamEventParser) {
		p.PruneParsedData()
	}, config, 1)
}

// --- Concurrent Benchmark ---

func Benchmark_Prune_Concurrent(b *testing.B) {
	config := DefaultBenchmarkConfig()
	parallelism := runtime.GOMAXPROCS(0) * MAXPROCS
	runSSEParserBenchmark(b, func(p *StreamEventParser) {
		p.PruneParsedData()
	}, config, parallelism)
}

//func Benchmark_Adaptive_Concurrent(b *testing.B) {
//  config := DefaultBenchmarkConfig()
//  parallelism := runtime.GOMAXPROCS(0) * MAXPROCS
//  runSSEParserBenchmark(b, func(p *StreamEventParser) {
//     p.PruneParsedDataAdaptive() // Per your original code, this benchmark tests the simple PruneParsedData
//  }, config, parallelism)
//}
//
//func Benchmark_PruneThreeIndex_Concurrent(b *testing.B) {
//  config := DefaultBenchmarkConfig()
//  parallelism := runtime.GOMAXPROCS(0) * MAXPROCS
//  runSSEParserBenchmark(b, func(p *StreamEventParser) {
//     p.PruneParsedDataThreeIndex()
//  }, config, parallelism)
//}

// --- HighLoad Sequential Benchmark ---

func Benchmark_Prune_HighLoad_Sequential(b *testing.B) {
	config := DefaultBenchmarkConfigHigh()
	runSSEParserBenchmark(b, func(p *StreamEventParser) {
		p.PruneParsedData()
	}, config, 1)
}

//func Benchmark_Adaptive_HighLoad_Sequential(b *testing.B) {
//  config := DefaultBenchmarkConfigHigh()
//  runSSEParserBenchmark(b, func(p *StreamEventParser) {
//     p.PruneParsedDataAdaptive()
//  }, config, 1)
//}
//
//func Benchmark_PruneThreeIndex_HighLoad_Sequential(b *testing.B) {
//  config := DefaultBenchmarkConfigHigh()
//  runSSEParserBenchmark(b, func(p *StreamEventParser) {
//     p.PruneParsedDataThreeIndex()
//  }, config, 1)
//}
