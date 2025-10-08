package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

// BenchmarkResult represents a single benchmark result
type BenchmarkResult struct {
	Name         string  `json:"name"`
	Iterations   int     `json:"iterations"`
	NsPerOp      float64 `json:"ns_per_op"`
	MBPerSec     float64 `json:"mb_per_sec,omitempty"`
	AllocsPerOp  int     `json:"allocs_per_op"`
	BytesPerOp   int     `json:"bytes_per_op"`
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
	inputFile := flag.String("input", "benchmark_results.json", "Input benchmark results file")
	outputFile := flag.String("output", "benchmark_analysis.md", "Output analysis file")
	compareFile := flag.String("compare", "", "Previous benchmark results for comparison")
	flag.Parse()

	// Read benchmark results
	data, err := os.ReadFile(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input file: %v\n", err)
		os.Exit(1)
	}

	var report BenchmarkReport
	if err := json.Unmarshal(data, &report); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	// Generate analysis
	analysis := generateAnalysis(report)

	// If comparison file provided, generate comparison
	if *compareFile != "" {
		compareData, err := os.ReadFile(*compareFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading comparison file: %v\n", err)
		} else {
			var compareReport BenchmarkReport
			if err := json.Unmarshal(compareData, &compareReport); err == nil {
				comparison := generateComparison(report, compareReport)
				analysis += "\n\n" + comparison
			}
		}
	}

	// Write output
	if err := os.WriteFile(*outputFile, []byte(analysis), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Analysis written to %s\n", *outputFile)
}

func generateAnalysis(report BenchmarkReport) string {
	var sb strings.Builder

	sb.WriteString("# SAGE Benchmark Analysis\n\n")
	sb.WriteString(fmt.Sprintf("**Generated**: %s\n", report.Timestamp))
	sb.WriteString(fmt.Sprintf("**Go Version**: %s\n", report.GoVersion))
	sb.WriteString(fmt.Sprintf("**Platform**: %s/%s\n\n", report.OS, report.Arch))

	// Group results by category
	categories := make(map[string][]BenchmarkResult)
	for _, result := range report.Results {
		category := extractCategory(result.Name)
		categories[category] = append(categories[category], result)
	}

	// Sort categories
	var categoryNames []string
	for name := range categories {
		categoryNames = append(categoryNames, name)
	}
	sort.Strings(categoryNames)

	// Generate tables for each category
	for _, category := range categoryNames {
		sb.WriteString(fmt.Sprintf("## %s\n\n", category))
		sb.WriteString("| Benchmark | ns/op | MB/s | Allocs/op | Bytes/op |\n")
		sb.WriteString("|-----------|-------|------|-----------|----------|\n")

		results := categories[category]
		sort.Slice(results, func(i, j int) bool {
			return results[i].Name < results[j].Name
		})

		for _, result := range results {
			name := strings.TrimPrefix(result.Name, "Benchmark"+category+"_")
			mbPerSec := "-"
			if result.MBPerSec > 0 {
				mbPerSec = fmt.Sprintf("%.2f", result.MBPerSec)
			}

			sb.WriteString(fmt.Sprintf("| %s | %.2f | %s | %d | %d |\n",
				name,
				result.NsPerOp,
				mbPerSec,
				result.AllocsPerOp,
				result.BytesPerOp,
			))
		}
		sb.WriteString("\n")
	}

	// Summary statistics
	sb.WriteString("## Summary Statistics\n\n")

	totalOps := 0
	totalAllocs := 0
	for _, result := range report.Results {
		totalOps += result.Iterations
		totalAllocs += result.AllocsPerOp * result.Iterations
	}

	sb.WriteString(fmt.Sprintf("- **Total Benchmarks**: %d\n", len(report.Results)))
	sb.WriteString(fmt.Sprintf("- **Total Operations**: %d\n", totalOps))
	sb.WriteString(fmt.Sprintf("- **Total Allocations**: %d\n\n", totalAllocs))

	// Find fastest/slowest operations
	fastest, slowest := findExtremes(report.Results)
	sb.WriteString(fmt.Sprintf("**Fastest Operation**: %s (%.2f ns/op)\n", fastest.Name, fastest.NsPerOp))
	sb.WriteString(fmt.Sprintf("**Slowest Operation**: %s (%.2f ns/op)\n\n", slowest.Name, slowest.NsPerOp))

	return sb.String()
}

func generateComparison(current, previous BenchmarkReport) string {
	var sb strings.Builder

	sb.WriteString("## Performance Comparison\n\n")
	sb.WriteString(fmt.Sprintf("Comparing current (%s) vs previous (%s)\n\n", current.Timestamp, previous.Timestamp))

	// Create lookup map for previous results
	prevMap := make(map[string]BenchmarkResult)
	for _, result := range previous.Results {
		prevMap[result.Name] = result
	}

	sb.WriteString("| Benchmark | Current (ns/op) | Previous (ns/op) | Change | Status |\n")
	sb.WriteString("|-----------|-----------------|------------------|--------|--------|\n")

	for _, curr := range current.Results {
		prev, exists := prevMap[curr.Name]
		if !exists {
			sb.WriteString(fmt.Sprintf("| %s | %.2f | - | NEW | ✨ |\n", curr.Name, curr.NsPerOp))
			continue
		}

		change := ((curr.NsPerOp - prev.NsPerOp) / prev.NsPerOp) * 100
		status := "✅"
		if change > 10 {
			status = "⚠️"
		} else if change > 20 {
			status = "❌"
		}

		sb.WriteString(fmt.Sprintf("| %s | %.2f | %.2f | %.1f%% | %s |\n",
			curr.Name,
			curr.NsPerOp,
			prev.NsPerOp,
			change,
			status,
		))
	}

	sb.WriteString("\n")
	sb.WriteString("Legend:\n")
	sb.WriteString("- ✅ Performance within 10% (acceptable)\n")
	sb.WriteString("- ⚠️ Performance degraded 10-20% (review needed)\n")
	sb.WriteString("- ❌ Performance degraded >20% (action required)\n")
	sb.WriteString("- ✨ New benchmark\n\n")

	return sb.String()
}

func extractCategory(name string) string {
	name = strings.TrimPrefix(name, "Benchmark")
	parts := strings.Split(name, "_")
	if len(parts) > 0 {
		return parts[0]
	}
	return "Other"
}

func findExtremes(results []BenchmarkResult) (fastest, slowest BenchmarkResult) {
	if len(results) == 0 {
		return
	}

	fastest = results[0]
	slowest = results[0]

	for _, result := range results {
		if result.NsPerOp < fastest.NsPerOp {
			fastest = result
		}
		if result.NsPerOp > slowest.NsPerOp {
			slowest = result
		}
	}

	return
}
