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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"
)

// FuzzReport represents a complete fuzzing test report
type FuzzReport struct {
	StartTime     time.Time               `json:"start_time"`
	EndTime       time.Time               `json:"end_time"`
	Duration      time.Duration           `json:"duration"`
	TotalTests    int64                   `json:"total_tests"`
	PassedTests   int64                   `json:"passed_tests"`
	FailedTests   int64                   `json:"failed_tests"`
	SkippedTests  int64                   `json:"skipped_tests"`
	SuccessRate   float64                 `json:"success_rate"`
	Configuration *FuzzerConfig           `json:"configuration"`
	Results       []TestResult            `json:"results"`
	Statistics    Statistics              `json:"statistics"`
	Defects       []Defect                `json:"defects,omitempty"`
	Summary       string                  `json:"summary"`
}

// Statistics contains statistical analysis of test results
type Statistics struct {
	AverageDuration time.Duration                        `json:"average_duration"`
	TestsPerSecond  float64                              `json:"tests_per_second"`
	CategoryStats   map[TestCategory]*CategoryStatistics `json:"category_stats"`
	ErrorFrequency  map[string]int                       `json:"error_frequency"`
}

// CategoryStatistics contains statistics for a specific test category
type CategoryStatistics struct {
	Category        TestCategory  `json:"category"`
	TotalTests      int           `json:"total_tests"`
	PassedTests     int           `json:"passed_tests"`
	FailedTests     int           `json:"failed_tests"`
	SkippedTests    int           `json:"skipped_tests"`
	SuccessRate     float64       `json:"success_rate"`
	AverageDuration time.Duration `json:"average_duration"`
	CommonErrors    []string      `json:"common_errors,omitempty"`
}

// Defect represents a test defect or failure
type Defect struct {
	TestID      string       `json:"test_id"`
	Category    TestCategory `json:"category"`
	Error       string       `json:"error"`
	ErrorDetail string       `json:"error_detail"`
	Input       interface{}  `json:"input"`
	Severity    string       `json:"severity"`
	Timestamp   time.Time    `json:"timestamp"`
}

// ResultReporter handles test result reporting
type ResultReporter struct {
	reportPath string
	format     ReportFormat
}

// ReportFormat defines the output format for reports
type ReportFormat string

const (
	FormatJSON     ReportFormat = "json"
	FormatHTML     ReportFormat = "html"
	FormatMarkdown ReportFormat = "markdown"
	FormatText     ReportFormat = "text"
)

// NewResultReporter creates a new result reporter
func NewResultReporter(reportPath string) *ResultReporter {
	format := FormatJSON

	// Determine format from file extension
	switch filepath.Ext(reportPath) {
	case ".html":
		format = FormatHTML
	case ".md":
		format = FormatMarkdown
	case ".txt":
		format = FormatText
	default:
		format = FormatJSON
	}

	return &ResultReporter{
		reportPath: reportPath,
		format:     format,
	}
}

// Save saves the report to file
func (r *ResultReporter) Save(report *FuzzReport) error {
	// Analyze defects
	report.Defects = r.analyzeDefects(report.Results)

	// Generate summary
	report.Summary = r.generateSummary(report)

	// Ensure directory exists
	dir := filepath.Dir(r.reportPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create report directory: %w", err)
	}

	switch r.format {
	case FormatJSON:
		return r.saveJSON(report)
	case FormatHTML:
		return r.saveHTML(report)
	case FormatMarkdown:
		return r.saveMarkdown(report)
	case FormatText:
		return r.saveText(report)
	default:
		return r.saveJSON(report)
	}
}

// saveJSON saves report as JSON
func (r *ResultReporter) saveJSON(report *FuzzReport) error {
	file, err := os.Create(r.reportPath)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(report); err != nil {
		return fmt.Errorf("failed to encode report: %w", err)
	}

	return nil
}

// saveHTML saves report as HTML
func (r *ResultReporter) saveHTML(report *FuzzReport) error {
	htmlTemplate := `<!DOCTYPE html>
<html>
<head>
    <title>SAGE Random Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        h1 { color: #333; }
        .summary { background: #f0f0f0; padding: 15px; border-radius: 5px; margin: 20px 0; }
        .stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; }
        .stat-card { background: white; border: 1px solid #ddd; padding: 15px; border-radius: 5px; }
        .success { color: green; }
        .failure { color: red; }
        .warning { color: orange; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { padding: 10px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background: #f5f5f5; }
        .defect { background: #ffe0e0; }
        .progress-bar { width: 100%; height: 20px; background: #f0f0f0; border-radius: 10px; overflow: hidden; }
        .progress-fill { height: 100%; background: linear-gradient(90deg, #4CAF50, #45a049); }
    </style>
</head>
<body>
    <h1>SAGE Random Test Report</h1>

    <div class="summary">
        <h2>Executive Summary</h2>
        <p>{{.Summary}}</p>
        <p>Test Duration: {{.Duration}}</p>
        <p>Success Rate: <span class="{{if ge .SuccessRate 90.0}}success{{else if ge .SuccessRate 70.0}}warning{{else}}failure{{end}}">{{printf "%.2f" .SuccessRate}}%</span></p>
    </div>

    <div class="stats">
        <div class="stat-card">
            <h3>Total Tests</h3>
            <p style="font-size: 24px; font-weight: bold;">{{.TotalTests}}</p>
        </div>
        <div class="stat-card">
            <h3>Passed</h3>
            <p style="font-size: 24px; font-weight: bold; color: green;">{{.PassedTests}}</p>
        </div>
        <div class="stat-card">
            <h3>Failed</h3>
            <p style="font-size: 24px; font-weight: bold; color: red;">{{.FailedTests}}</p>
        </div>
        <div class="stat-card">
            <h3>Tests/Second</h3>
            <p style="font-size: 24px; font-weight: bold;">{{printf "%.2f" .Statistics.TestsPerSecond}}</p>
        </div>
    </div>

    <h2>Category Breakdown</h2>
    <table>
        <thead>
            <tr>
                <th>Category</th>
                <th>Total</th>
                <th>Passed</th>
                <th>Failed</th>
                <th>Success Rate</th>
                <th>Avg Duration</th>
            </tr>
        </thead>
        <tbody>
            {{range $cat, $stats := .Statistics.CategoryStats}}
            <tr>
                <td>{{$cat}}</td>
                <td>{{$stats.TotalTests}}</td>
                <td class="success">{{$stats.PassedTests}}</td>
                <td class="failure">{{$stats.FailedTests}}</td>
                <td>{{printf "%.2f" $stats.SuccessRate}}%</td>
                <td>{{$stats.AverageDuration}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>

    {{if .Defects}}
    <h2>Defects Found</h2>
    <table>
        <thead>
            <tr>
                <th>Test ID</th>
                <th>Category</th>
                <th>Error</th>
                <th>Severity</th>
            </tr>
        </thead>
        <tbody>
            {{range .Defects}}
            <tr class="defect">
                <td>{{.TestID}}</td>
                <td>{{.Category}}</td>
                <td>{{.Error}}</td>
                <td>{{.Severity}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{end}}

    <div style="margin-top: 40px; padding-top: 20px; border-top: 1px solid #ddd; text-align: center; color: #666;">
        Generated: {{.EndTime.Format "2006-01-02 15:04:05"}}
    </div>
</body>
</html>`

	tmpl, err := template.New("report").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}

	file, err := os.Create(r.reportPath)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer file.Close()

	return tmpl.Execute(file, report)
}

// saveMarkdown saves report as Markdown
func (r *ResultReporter) saveMarkdown(report *FuzzReport) error {
	file, err := os.Create(r.reportPath)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer file.Close()

	fmt.Fprintf(file, "# SAGE Random Test Report\n\n")
	fmt.Fprintf(file, "## Executive Summary\n\n")
	fmt.Fprintf(file, "%s\n\n", report.Summary)
	fmt.Fprintf(file, "- **Duration**: %v\n", report.Duration)
	fmt.Fprintf(file, "- **Total Tests**: %d\n", report.TotalTests)
	fmt.Fprintf(file, "- **Passed**: %d\n", report.PassedTests)
	fmt.Fprintf(file, "- **Failed**: %d\n", report.FailedTests)
	fmt.Fprintf(file, "- **Success Rate**: %.2f%%\n\n", report.SuccessRate)

	fmt.Fprintf(file, "## Performance Metrics\n\n")
	fmt.Fprintf(file, "- **Average Duration**: %v\n", report.Statistics.AverageDuration)
	fmt.Fprintf(file, "- **Tests per Second**: %.2f\n\n", report.Statistics.TestsPerSecond)

	fmt.Fprintf(file, "## Category Results\n\n")
	fmt.Fprintf(file, "| Category | Total | Passed | Failed | Success Rate | Avg Duration |\n")
	fmt.Fprintf(file, "|----------|-------|--------|--------|--------------|-------------|\n")

	// Sort categories for consistent output
	var categories []TestCategory
	for cat := range report.Statistics.CategoryStats {
		categories = append(categories, cat)
	}
	sort.Slice(categories, func(i, j int) bool {
		return string(categories[i]) < string(categories[j])
	})

	for _, cat := range categories {
		stats := report.Statistics.CategoryStats[cat]
		fmt.Fprintf(file, "| %s | %d | %d | %d | %.2f%% | %v |\n",
			cat, stats.TotalTests, stats.PassedTests, stats.FailedTests,
			stats.SuccessRate, stats.AverageDuration)
	}

	if len(report.Defects) > 0 {
		fmt.Fprintf(file, "\n## Defects Found\n\n")
		fmt.Fprintf(file, "| Test ID | Category | Error | Severity |\n")
		fmt.Fprintf(file, "|---------|----------|-------|----------|\n")
		for _, defect := range report.Defects {
			fmt.Fprintf(file, "| %s | %s | %s | %s |\n",
				defect.TestID, defect.Category,
				strings.ReplaceAll(defect.Error, "|", "\\|"),
				defect.Severity)
		}
	}

	fmt.Fprintf(file, "\n---\n\n")
	fmt.Fprintf(file, "_Generated: %s_\n", report.EndTime.Format("2006-01-02 15:04:05"))

	return nil
}

// saveText saves report as plain text
func (r *ResultReporter) saveText(report *FuzzReport) error {
	file, err := os.Create(r.reportPath)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer file.Close()

	fmt.Fprintln(file, "================================================================================")
	fmt.Fprintln(file, "                        SAGE RANDOM TEST REPORT")
	fmt.Fprintln(file, "================================================================================")
	fmt.Fprintln(file)
	fmt.Fprintln(file, report.Summary)
	fmt.Fprintln(file)
	fmt.Fprintf(file, "Test Duration:  %v\n", report.Duration)
	fmt.Fprintf(file, "Total Tests:    %d\n", report.TotalTests)
	fmt.Fprintf(file, "Passed:         %d\n", report.PassedTests)
	fmt.Fprintf(file, "Failed:         %d\n", report.FailedTests)
	fmt.Fprintf(file, "Success Rate:   %.2f%%\n", report.SuccessRate)
	fmt.Fprintln(file)
	fmt.Fprintln(file, "CATEGORY BREAKDOWN:")
	fmt.Fprintln(file, "-------------------")

	for cat, stats := range report.Statistics.CategoryStats {
		fmt.Fprintf(file, "\n%s:\n", cat)
		fmt.Fprintf(file, "  Total:        %d\n", stats.TotalTests)
		fmt.Fprintf(file, "  Passed:       %d\n", stats.PassedTests)
		fmt.Fprintf(file, "  Failed:       %d\n", stats.FailedTests)
		fmt.Fprintf(file, "  Success Rate: %.2f%%\n", stats.SuccessRate)
	}

	if len(report.Defects) > 0 {
		fmt.Fprintln(file)
		fmt.Fprintln(file, "DEFECTS FOUND:")
		fmt.Fprintln(file, "--------------")
		for i, defect := range report.Defects {
			fmt.Fprintf(file, "\n%d. %s (Category: %s, Severity: %s)\n",
				i+1, defect.TestID, defect.Category, defect.Severity)
			fmt.Fprintf(file, "   Error: %s\n", defect.Error)
		}
	}

	fmt.Fprintln(file)
	fmt.Fprintln(file, "================================================================================")
	fmt.Fprintf(file, "Generated: %s\n", report.EndTime.Format("2006-01-02 15:04:05"))

	return nil
}

// analyzeDefects analyzes test results to identify defects
func (r *ResultReporter) analyzeDefects(results []TestResult) []Defect {
	var defects []Defect

	for _, result := range results {
		if !result.Passed && !result.Skipped && result.Error != nil {
			severity := "LOW"

			// Determine severity based on error type
			errorStr := result.Error.Error()
			if strings.Contains(errorStr, "panic") || strings.Contains(errorStr, "fatal") {
				severity = "CRITICAL"
			} else if strings.Contains(errorStr, "timeout") || strings.Contains(errorStr, "connection") {
				severity = "HIGH"
			} else if strings.Contains(errorStr, "validation") || strings.Contains(errorStr, "verification") {
				severity = "MEDIUM"
			}

			defects = append(defects, Defect{
				TestID:      result.TestCase.ID,
				Category:    result.TestCase.Category,
				Error:       errorStr,
				ErrorDetail: result.ErrorDetail,
				Input:       result.TestCase.Input,
				Severity:    severity,
				Timestamp:   result.ExecutedAt,
			})
		}
	}

	// Sort defects by severity
	sort.Slice(defects, func(i, j int) bool {
		severityOrder := map[string]int{"CRITICAL": 0, "HIGH": 1, "MEDIUM": 2, "LOW": 3}
		return severityOrder[defects[i].Severity] < severityOrder[defects[j].Severity]
	})

	return defects
}

// generateSummary generates a text summary of the report
func (r *ResultReporter) generateSummary(report *FuzzReport) string {
	status := "FAILED"
	if report.SuccessRate >= 95.0 {
		status = "EXCELLENT"
	} else if report.SuccessRate >= 90.0 {
		status = "PASSED"
	} else if report.SuccessRate >= 80.0 {
		status = "WARNING"
	}

	defectCount := len(report.Defects)
	criticalCount := 0
	for _, d := range report.Defects {
		if d.Severity == "CRITICAL" {
			criticalCount++
		}
	}

	summary := fmt.Sprintf(
		"Random testing completed with status: %s. "+
		"Executed %d tests in %v with %.2f%% success rate. "+
		"Found %d defects (%d critical). "+
		"Performance: %.2f tests/second.",
		status, report.TotalTests, report.Duration,
		report.SuccessRate, defectCount, criticalCount,
		report.Statistics.TestsPerSecond,
	)

	return summary
}

// PrintSummary prints a summary to stdout
func (r *ResultReporter) PrintSummary(report *FuzzReport) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("SAGE RANDOM TEST SUMMARY")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("\n%s\n\n", report.Summary)

	fmt.Printf("%-20s %d\n", "Total Tests:", report.TotalTests)
	fmt.Printf("%-20s %d\n", "Passed:", report.PassedTests)
	fmt.Printf("%-20s %d\n", "Failed:", report.FailedTests)
	fmt.Printf("%-20s %.2f%%\n", "Success Rate:", report.SuccessRate)
	fmt.Printf("%-20s %v\n", "Duration:", report.Duration)
	fmt.Printf("%-20s %.2f\n", "Tests/Second:", report.Statistics.TestsPerSecond)

	if len(report.Defects) > 0 {
		fmt.Printf("\nWARNING: Found %d defects - see %s for details\n",
			len(report.Defects), r.reportPath)
	}

	fmt.Println(strings.Repeat("=", 80))
}