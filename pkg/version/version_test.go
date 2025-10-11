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

package version

import (
	"runtime"
	"strings"
	"testing"
)

func TestGet(t *testing.T) {
	info := Get()

	if info.Version == "" {
		t.Error("Version should not be empty")
	}

	if info.GoVersion == "" {
		t.Error("GoVersion should not be empty")
	}

	if info.Platform == "" {
		t.Error("Platform should not be empty")
	}

	expectedPlatform := runtime.GOOS + "/" + runtime.GOARCH
	if info.Platform != expectedPlatform {
		t.Errorf("Expected platform %s, got %s", expectedPlatform, info.Platform)
	}
}

func TestString(t *testing.T) {
	// Save original values
	origVersion := Version
	origCommit := GitCommit
	origBranch := GitBranch
	origDate := BuildDate

	// Test without git info
	Version = "1.0.0"
	GitCommit = ""
	GitBranch = ""
	BuildDate = ""

	str := String()
	if !strings.Contains(str, "1.0.0") {
		t.Errorf("String should contain version 1.0.0, got: %s", str)
	}

	// Test with git info
	Version = "1.0.0"
	GitCommit = "abcdef1234567890"
	GitBranch = "main"
	BuildDate = "2025-01-11"

	str = String()
	if !strings.Contains(str, "1.0.0") {
		t.Errorf("String should contain version 1.0.0, got: %s", str)
	}
	if !strings.Contains(str, "abcdef1") {
		t.Errorf("String should contain commit hash prefix, got: %s", str)
	}
	if !strings.Contains(str, "main") {
		t.Errorf("String should contain branch name, got: %s", str)
	}

	// Restore original values
	Version = origVersion
	GitCommit = origCommit
	GitBranch = origBranch
	BuildDate = origDate
}

func TestShort(t *testing.T) {
	// Save original values
	origVersion := Version
	origCommit := GitCommit

	// Test without commit
	Version = "1.0.0"
	GitCommit = ""

	short := Short()
	if short != "1.0.0" {
		t.Errorf("Expected short version '1.0.0', got '%s'", short)
	}

	// Test with commit
	Version = "1.0.0"
	GitCommit = "abcdef1234567890"

	short = Short()
	expected := "1.0.0-abcdef1"
	if short != expected {
		t.Errorf("Expected short version '%s', got '%s'", expected, short)
	}

	// Restore original values
	Version = origVersion
	GitCommit = origCommit
}

func TestUserAgent(t *testing.T) {
	// Save original values
	origVersion := Version
	origCommit := GitCommit

	Version = "1.0.0"
	GitCommit = ""

	ua := UserAgent()
	expected := "SAGE/1.0.0"
	if ua != expected {
		t.Errorf("Expected UserAgent '%s', got '%s'", expected, ua)
	}

	// Test with commit
	GitCommit = "abcdef1234567890"

	ua = UserAgent()
	expected = "SAGE/1.0.0-abcdef1"
	if ua != expected {
		t.Errorf("Expected UserAgent '%s', got '%s'", expected, ua)
	}

	// Restore original values
	Version = origVersion
	GitCommit = origCommit
}

func TestGetModuleVersion(t *testing.T) {
	// This test just ensures the function doesn't panic
	version := GetModuleVersion()
	if version == "" {
		t.Error("GetModuleVersion should not return empty string")
	}
}

func TestPrintVersion(t *testing.T) {
	// This test just ensures the function doesn't panic
	PrintVersion()
}

func TestPrintVersionJSON(t *testing.T) {
	// This test just ensures the function doesn't panic
	PrintVersionJSON()
}

func TestVersionConstants(t *testing.T) {
	// Ensure Version is set
	if Version == "" {
		t.Error("Version constant should be set")
	}

	// GoVersion should always be set by runtime
	if GoVersion == "" {
		t.Error("GoVersion should be set by runtime.Version()")
	}

	// Check that GoVersion starts with "go"
	if !strings.HasPrefix(GoVersion, "go") {
		t.Errorf("GoVersion should start with 'go', got: %s", GoVersion)
	}
}

func TestInfoStruct(t *testing.T) {
	info := Info{
		Version:   "1.0.0",
		GitCommit: "abc123",
		GitBranch: "main",
		BuildDate: "2025-01-11",
		GoVersion: "go1.23.0",
		Platform:  "linux/amd64",
	}

	if info.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", info.Version)
	}

	if info.GitCommit != "abc123" {
		t.Errorf("Expected commit abc123, got %s", info.GitCommit)
	}

	if info.GitBranch != "main" {
		t.Errorf("Expected branch main, got %s", info.GitBranch)
	}

	if info.BuildDate != "2025-01-11" {
		t.Errorf("Expected date 2025-01-11, got %s", info.BuildDate)
	}

	if info.GoVersion != "go1.23.0" {
		t.Errorf("Expected Go version go1.23.0, got %s", info.GoVersion)
	}

	if info.Platform != "linux/amd64" {
		t.Errorf("Expected platform linux/amd64, got %s", info.Platform)
	}
}
