package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// BenchmarkResult represents a single benchmark result
type BenchmarkResult struct {
	Name        string  `json:"name"`
	Iterations  int     `json:"iterations"`
	NsPerOp     float64 `json:"ns_per_op"`
	MBPerSec    float64 `json:"mb_per_sec,omitempty"`
	AllocsPerOp int     `json:"allocs_per_op"`
	BytesPerOp  int     `json:"bytes_per_op"`
}

// BenchmarkReport represents the full benchmark report
type BenchmarkReport struct {
	Timestamp string            `json:"timestamp"`
	GoVersion string            `json:"go_version"`
	OS        string            `json:"os"`
	Arch      string            `json:"arch"`
	Results   []BenchmarkResult `json:"results"`
}

func main() {
	inputFile := flag.String("input", "", "Input benchmark results file")
	outputFile := flag.String("output", "", "Output JSON file")
	flag.Parse()

	if *inputFile == "" || *outputFile == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s -input <file> -output <file>\n", os.Args[0])
		os.Exit(1)
	}

	// Read input file
	file, err := os.Open(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening input file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Parse benchmark results
	results := parseBenchmarkResults(file)

	// Create report
	report := BenchmarkReport{
		Timestamp: time.Now().Format(time.RFC3339),
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		Results:   results,
	}

	// Write JSON output
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outputFile, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Parsed %d benchmark results\n", len(results))
	fmt.Printf("Output written to %s\n", *outputFile)
}

func parseBenchmarkResults(file *os.File) []BenchmarkResult {
	var results []BenchmarkResult

	// Regular expressions for parsing
	benchmarkRe := regexp.MustCompile(`^Benchmark(\S+)-\d+\s+(\d+)\s+(\d+\.?\d*)\s+ns/op`)
	memRe := regexp.MustCompile(`(\d+)\s+B/op\s+(\d+)\s+allocs/op`)
	mbPerSecRe := regexp.MustCompile(`(\d+\.?\d*)\s+MB/s`)

	scanner := bufio.NewScanner(file)
	var currentBench *BenchmarkResult

	for scanner.Scan() {
		line := scanner.Text()

		// Check for benchmark result line
		if matches := benchmarkRe.FindStringSubmatch(line); matches != nil {
			if currentBench != nil {
				results = append(results, *currentBench)
			}

			iterations, _ := strconv.Atoi(matches[2])
			nsPerOp, _ := strconv.ParseFloat(matches[3], 64)

			currentBench = &BenchmarkResult{
				Name:       "Benchmark" + matches[1],
				Iterations: iterations,
				NsPerOp:    nsPerOp,
			}

			// Check for memory stats on same line
			if memMatches := memRe.FindStringSubmatch(line); memMatches != nil {
				currentBench.BytesPerOp, _ = strconv.Atoi(memMatches[1])
				currentBench.AllocsPerOp, _ = strconv.Atoi(memMatches[2])
			}

			// Check for MB/s on same line
			if mbMatches := mbPerSecRe.FindStringSubmatch(line); mbMatches != nil {
				currentBench.MBPerSec, _ = strconv.ParseFloat(mbMatches[1], 64)
			}
		}
	}

	// Add last benchmark
	if currentBench != nil {
		results = append(results, *currentBench)
	}

	// Average results with same name (from multiple runs)
	results = averageDuplicates(results)

	return results
}

func averageDuplicates(results []BenchmarkResult) []BenchmarkResult {
	grouped := make(map[string][]BenchmarkResult)

	for _, result := range results {
		grouped[result.Name] = append(grouped[result.Name], result)
	}

	var averaged []BenchmarkResult
	for name, group := range grouped {
		if len(group) == 1 {
			averaged = append(averaged, group[0])
			continue
		}

		// Calculate averages
		var totalNs, totalMB float64
		var totalAllocs, totalBytes, totalIters int

		for _, result := range group {
			totalNs += result.NsPerOp
			totalMB += result.MBPerSec
			totalAllocs += result.AllocsPerOp
			totalBytes += result.BytesPerOp
			totalIters += result.Iterations
		}

		count := float64(len(group))
		averaged = append(averaged, BenchmarkResult{
			Name:        name,
			Iterations:  int(float64(totalIters) / count),
			NsPerOp:     totalNs / count,
			MBPerSec:    totalMB / count,
			AllocsPerOp: int(float64(totalAllocs) / count),
			BytesPerOp:  int(float64(totalBytes) / count),
		})
	}

	return averaged
}
