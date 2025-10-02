// Copyright (C) 2025 sage-x-project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// SPDX-License-Identifier: LGPL-3.0-or-later


package random

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"
)

// Fuzzer is the main random test framework
type Fuzzer struct {
	generator *TestCaseGenerator
	executor  *TestExecutor
	reporter  *ResultReporter
	config    *FuzzerConfig

	// Statistics
	totalTests     atomic.Int64
	passedTests    atomic.Int64
	failedTests    atomic.Int64
	skippedTests   atomic.Int64
	totalDuration  atomic.Int64 // in nanoseconds
}

// FuzzerConfig contains configuration for the fuzzer
type FuzzerConfig struct {
	Iterations      int
	Parallel        int
	Timeout         time.Duration
	Seed            int64
	Categories      []TestCategory
	VerboseMode     bool
	ReportPath      string
	StopOnFirstFail bool
}

// TestCategory represents different test categories
type TestCategory string

const (
	CategoryRFC9421     TestCategory = "rfc9421"
	CategoryCrypto      TestCategory = "crypto"
	CategoryDID         TestCategory = "did"
	CategoryBlockchain  TestCategory = "blockchain"
	CategorySession     TestCategory = "session"
	CategoryHPKE        TestCategory = "hpke"
	CategoryIntegration TestCategory = "integration"
)

// NewFuzzer creates a new fuzzer instance
func NewFuzzer(config *FuzzerConfig) *Fuzzer {
	if config == nil {
		config = &FuzzerConfig{
			Iterations: 100,
			Parallel:   1,
			Timeout:    30 * time.Second,
			Seed:       time.Now().UnixNano(),
			Categories: []TestCategory{
				CategoryRFC9421,
				CategoryCrypto,
				CategoryDID,
			},
		}
	}

	return &Fuzzer{
		generator: NewTestCaseGenerator(config.Seed),
		executor:  NewTestExecutor(config.Timeout),
		reporter:  NewResultReporter(config.ReportPath),
		config:    config,
	}
}

// Run executes the fuzzing tests
func (f *Fuzzer) Run(ctx context.Context) (*FuzzReport, error) {
	startTime := time.Now()

	// Create worker pool
	var wg sync.WaitGroup
	testChan := make(chan TestCase, f.config.Parallel)
	resultChan := make(chan TestResult, f.config.Iterations)

	// Start workers
	for i := 0; i < f.config.Parallel; i++ {
		wg.Add(1)
		go f.worker(ctx, &wg, testChan, resultChan)
	}

	// Generate and send test cases
	go func() {
		defer close(testChan)
		for i := 0; i < f.config.Iterations; i++ {
			select {
			case <-ctx.Done():
				return
			default:
				testCase := f.generator.Generate(f.config.Categories)
				testChan <- testCase
			}
		}
	}()

	// Collect results
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Process results
	var results []TestResult
	for result := range resultChan {
		results = append(results, result)
		f.updateStatistics(result)

		if f.config.StopOnFirstFail && !result.Passed {
			// Cancel context to stop all workers
			break
		}
	}

	// Generate report
	duration := time.Since(startTime)
	report := &FuzzReport{
		StartTime:     startTime,
		EndTime:       time.Now(),
		Duration:      duration,
		TotalTests:    f.totalTests.Load(),
		PassedTests:   f.passedTests.Load(),
		FailedTests:   f.failedTests.Load(),
		SkippedTests:  f.skippedTests.Load(),
		SuccessRate:   float64(f.passedTests.Load()) / float64(f.totalTests.Load()) * 100,
		Configuration: f.config,
		Results:       results,
		Statistics:    f.calculateStatistics(results),
	}

	// Save report
	if err := f.reporter.Save(report); err != nil {
		return report, fmt.Errorf("failed to save report: %w", err)
	}

	return report, nil
}

// worker processes test cases
func (f *Fuzzer) worker(ctx context.Context, wg *sync.WaitGroup, testChan <-chan TestCase, resultChan chan<- TestResult) {
	defer wg.Done()

	for testCase := range testChan {
		select {
		case <-ctx.Done():
			return
		default:
			result := f.executor.Execute(ctx, testCase)
			resultChan <- result
		}
	}
}

// updateStatistics updates test statistics
func (f *Fuzzer) updateStatistics(result TestResult) {
	f.totalTests.Add(1)

	switch {
	case result.Passed:
		f.passedTests.Add(1)
	case result.Skipped:
		f.skippedTests.Add(1)
	default:
		f.failedTests.Add(1)
	}

	f.totalDuration.Add(result.Duration.Nanoseconds())
}

// calculateStatistics calculates detailed statistics
func (f *Fuzzer) calculateStatistics(results []TestResult) Statistics {
	stats := Statistics{
		CategoryStats: make(map[TestCategory]*CategoryStatistics),
	}

	// Group by category
	categoryResults := make(map[TestCategory][]TestResult)
	for _, result := range results {
		categoryResults[result.TestCase.Category] = append(
			categoryResults[result.TestCase.Category],
			result,
		)
	}

	// Calculate per-category statistics
	for category, catResults := range categoryResults {
		catStats := &CategoryStatistics{
			Category:     category,
			TotalTests:   len(catResults),
			PassedTests:  0,
			FailedTests:  0,
			SkippedTests: 0,
		}

		var totalDuration time.Duration
		for _, result := range catResults {
			if result.Passed {
				catStats.PassedTests++
			} else if result.Skipped {
				catStats.SkippedTests++
			} else {
				catStats.FailedTests++
			}
			totalDuration += result.Duration
		}

		catStats.SuccessRate = float64(catStats.PassedTests) / float64(catStats.TotalTests) * 100
		catStats.AverageDuration = totalDuration / time.Duration(len(catResults))

		stats.CategoryStats[category] = catStats
	}

	// Calculate overall statistics
	if len(results) > 0 {
		stats.AverageDuration = time.Duration(f.totalDuration.Load()) / time.Duration(len(results))
		stats.TestsPerSecond = float64(len(results)) / time.Duration(f.totalDuration.Load()).Seconds()
	}

	return stats
}

// GenerateRandomBytes generates random bytes
func GenerateRandomBytes(size int) ([]byte, error) {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	return bytes, nil
}

// GenerateRandomString generates a random hex string
func GenerateRandomString(length int) string {
	bytes, _ := GenerateRandomBytes(length / 2)
	return hex.EncodeToString(bytes)
}

// GenerateRandomInt generates a random integer within range
func GenerateRandomInt(min, max int64) int64 {
	if min >= max {
		return min
	}

	n, _ := rand.Int(rand.Reader, big.NewInt(max-min))
	return n.Int64() + min
}

// GenerateRandomBool generates a random boolean
func GenerateRandomBool() bool {
	b := make([]byte, 1)
	rand.Read(b)
	return b[0]&1 == 1
}