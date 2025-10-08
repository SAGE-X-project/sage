// SAGE - Secure Agent Guarantee Engine
// Copyright (C) 2025 SAGE-X-project
//
// This file is part of SAGE.
//
// SAGE is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// SAGE is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with SAGE. If not, see <https://www.gnu.org/licenses/>.


package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/sage-x-project/sage/tests/random"
)

func main() {
	// Parse command line flags
	var (
		iterations      = flag.Int("iterations", 100, "Number of test iterations")
		parallel        = flag.Int("parallel", 1, "Number of parallel workers")
		timeout         = flag.Duration("timeout", 30*time.Second, "Timeout for each test")
		seed            = flag.Int64("seed", 0, "Random seed (0 for current time)")
		categories      = flag.String("categories", "all", "Test categories (comma-separated or 'all')")
		verbose         = flag.Bool("verbose", false, "Enable verbose output")
		report          = flag.String("report", "", "Report file path (json/html/md/txt)")
		stopOnFirstFail = flag.Bool("stop-on-fail", false, "Stop on first failure")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "SAGE Random Test Framework\n")
		fmt.Fprintf(os.Stderr, "==========================\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nCategories:\n")
		fmt.Fprintf(os.Stderr, "  rfc9421     - RFC 9421 signature tests\n")
		fmt.Fprintf(os.Stderr, "  crypto      - Cryptographic operation tests\n")
		fmt.Fprintf(os.Stderr, "  did         - DID management tests\n")
		fmt.Fprintf(os.Stderr, "  blockchain  - Blockchain integration tests\n")
		fmt.Fprintf(os.Stderr, "  session     - Session management tests\n")
		fmt.Fprintf(os.Stderr, "  hpke        - HPKE encryption tests\n")
		fmt.Fprintf(os.Stderr, "  integration - End-to-end integration tests\n")
		fmt.Fprintf(os.Stderr, "  all         - All categories\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  %s -iterations=1000 -parallel=10\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -categories=rfc9421,crypto -report=report.html\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -iterations=100 -stop-on-fail -verbose\n", os.Args[0])
	}

	flag.Parse()

	// Parse categories
	testCategories := parseCategories(*categories)
	if len(testCategories) == 0 {
		log.Fatal("No valid test categories specified")
	}

	// Determine seed
	if *seed == 0 {
		*seed = time.Now().UnixNano()
	}

	// Determine report path
	reportPath := *report
	if reportPath == "" {
		// Generate default report path
		timestamp := time.Now().Format("20060102-150405")
		reportPath = filepath.Join("reports", fmt.Sprintf("random-test-%s.json", timestamp))
	}

	// Ensure reports directory exists
	reportDir := filepath.Dir(reportPath)
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		log.Fatalf("Failed to create reports directory: %v", err)
	}

	// Create fuzzer configuration
	config := &random.FuzzerConfig{
		Iterations:      *iterations,
		Parallel:        *parallel,
		Timeout:         *timeout,
		Seed:            *seed,
		Categories:      testCategories,
		VerboseMode:     *verbose,
		ReportPath:      reportPath,
		StopOnFirstFail: *stopOnFirstFail,
	}

	// Print configuration
	fmt.Println("SAGE Random Test Framework")
	fmt.Println("==========================")
	fmt.Printf("Iterations:       %d\n", config.Iterations)
	fmt.Printf("Parallel Workers: %d\n", config.Parallel)
	fmt.Printf("Timeout:          %v\n", config.Timeout)
	fmt.Printf("Seed:             %d\n", config.Seed)
	fmt.Printf("Categories:       %v\n", config.Categories)
	fmt.Printf("Report Path:      %s\n", config.ReportPath)
	fmt.Printf("Verbose Mode:     %v\n", config.VerboseMode)
	fmt.Printf("Stop on Fail:     %v\n", config.StopOnFirstFail)
	fmt.Println()

	// Create fuzzer
	fuzzer := random.NewFuzzer(config)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nInterrupt received, stopping tests...")
		cancel()
	}()

	// Run tests
	fmt.Println("Starting random tests...")
	startTime := time.Now()

	testReport, err := fuzzer.Run(ctx)
	if err != nil {
		log.Printf("Error during test execution: %v", err)
	}

	// Print results
	fmt.Println()
	reporter := random.NewResultReporter(reportPath)
	reporter.PrintSummary(testReport)

	// Print report location
	fmt.Printf("\nDetailed report saved to: %s\n", reportPath)

	// Determine exit code based on success rate
	exitCode := 0
	if testReport.SuccessRate < 95.0 {
		exitCode = 1
		fmt.Printf("\nWARNING: Success rate (%.2f%%) is below 95%%\n", testReport.SuccessRate)
	}

	if testReport.SuccessRate < 90.0 {
		exitCode = 2
		fmt.Printf("\nFAILURE: Success rate (%.2f%%) is below 90%%\n", testReport.SuccessRate)
	}

	// Print evaluation score estimate
	evaluationScore := calculateEvaluationScore(testReport)
	fmt.Printf("\nEstimated Evaluation Score: %.1f/5.0", evaluationScore)
	if evaluationScore >= 4.5 {
		fmt.Println(" (EXCELLENT - Full bonus points expected)")
	} else if evaluationScore >= 4.0 {
		fmt.Println(" (GOOD - Partial bonus points expected)")
	} else if evaluationScore >= 3.0 {
		fmt.Println(" (ACCEPTABLE - No bonus points)")
	} else {
		fmt.Println(" (NEEDS IMPROVEMENT)")
	}

	fmt.Printf("\nTotal execution time: %v\n", time.Since(startTime))

	os.Exit(exitCode)
}

// parseCategories parses the categories string into TestCategory slice
func parseCategories(categoriesStr string) []random.TestCategory {
	var categories []random.TestCategory

	if categoriesStr == "all" || categoriesStr == "" {
		return []random.TestCategory{
			random.CategoryRFC9421,
			random.CategoryCrypto,
			random.CategoryDID,
			random.CategoryBlockchain,
			random.CategorySession,
			random.CategoryHPKE,
			random.CategoryIntegration,
		}
	}

	parts := strings.Split(categoriesStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(strings.ToLower(part))
		switch part {
		case "rfc9421":
			categories = append(categories, random.CategoryRFC9421)
		case "crypto":
			categories = append(categories, random.CategoryCrypto)
		case "did":
			categories = append(categories, random.CategoryDID)
		case "blockchain":
			categories = append(categories, random.CategoryBlockchain)
		case "session":
			categories = append(categories, random.CategorySession)
		case "hpke":
			categories = append(categories, random.CategoryHPKE)
		case "integration":
			categories = append(categories, random.CategoryIntegration)
		default:
			fmt.Fprintf(os.Stderr, "Warning: Unknown category '%s' ignored\n", part)
		}
	}

	return categories
}

// calculateEvaluationScore estimates the evaluation score based on test results
func calculateEvaluationScore(report *random.FuzzReport) float64 {
	baseScore := 0.0

	// Base score from success rate (0-3 points)
	if report.SuccessRate >= 98.0 {
		baseScore = 3.0
	} else if report.SuccessRate >= 95.0 {
		baseScore = 2.5
	} else if report.SuccessRate >= 90.0 {
		baseScore = 2.0
	} else if report.SuccessRate >= 80.0 {
		baseScore = 1.5
	} else if report.SuccessRate >= 70.0 {
		baseScore = 1.0
	} else {
		baseScore = 0.5
	}

	// Bonus for test volume (0-1 point)
	volumeBonus := 0.0
	if report.TotalTests >= 10000 {
		volumeBonus = 1.0
	} else if report.TotalTests >= 5000 {
		volumeBonus = 0.8
	} else if report.TotalTests >= 1000 {
		volumeBonus = 0.6
	} else if report.TotalTests >= 500 {
		volumeBonus = 0.4
	} else if report.TotalTests >= 100 {
		volumeBonus = 0.2
	}

	// Bonus for no critical defects (0-1 point)
	defectBonus := 1.0
	criticalCount := 0
	for _, defect := range report.Defects {
		if defect.Severity == "CRITICAL" {
			criticalCount++
		}
	}
	if criticalCount > 0 {
		defectBonus = 0.0
	} else if len(report.Defects) > 10 {
		defectBonus = 0.5
	} else if len(report.Defects) > 5 {
		defectBonus = 0.7
	}

	return baseScore + volumeBonus + defectBonus
}
