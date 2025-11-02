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

// Package version provides version information for SAGE.
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

// Build information. Populated at build-time via ldflags.
var (
	// Version is the semantic version (set via ldflags or VERSION file).
	Version = "1.5.2"

	// GitCommit is the git commit hash (set via ldflags).
	GitCommit = ""

	// GitBranch is the git branch (set via ldflags).
	GitBranch = ""

	// BuildDate is the build date (set via ldflags).
	BuildDate = ""

	// GoVersion is the Go version used to build.
	GoVersion = runtime.Version()
)

// Info contains version information.
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit,omitempty"`
	GitBranch string `json:"git_branch,omitempty"`
	BuildDate string `json:"build_date,omitempty"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// Get returns the version information.
func Get() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		GitBranch: GitBranch,
		BuildDate: BuildDate,
		GoVersion: GoVersion,
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String returns the version information as a formatted string.
func String() string {
	info := Get()
	if info.GitCommit != "" {
		return fmt.Sprintf("%s (commit: %s, branch: %s, built: %s, go: %s, platform: %s)",
			info.Version,
			info.GitCommit[:7],
			info.GitBranch,
			info.BuildDate,
			info.GoVersion,
			info.Platform,
		)
	}
	return fmt.Sprintf("%s (go: %s, platform: %s)",
		info.Version,
		info.GoVersion,
		info.Platform,
	)
}

// Short returns a short version string.
func Short() string {
	if GitCommit != "" {
		return fmt.Sprintf("%s-%s", Version, GitCommit[:7])
	}
	return Version
}

// UserAgent returns a User-Agent string for SAGE.
func UserAgent() string {
	return fmt.Sprintf("SAGE/%s", Short())
}

// GetModuleVersion attempts to get version from Go module info.
// This works when SAGE is used as a library dependency.
func GetModuleVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return Version
	}

	// Try to find SAGE module in dependencies
	for _, dep := range info.Deps {
		if dep.Path == "github.com/sage-x-project/sage" {
			if dep.Version != "" && dep.Version != "(devel)" {
				return dep.Version
			}
		}
	}

	// Fallback to main module version
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}

	return Version
}

// PrintVersion prints version information to stdout.
func PrintVersion() {
	fmt.Println(String())
}

// PrintVersionJSON prints version information as JSON.
func PrintVersionJSON() {
	info := Get()
	fmt.Printf(`{
  "version": "%s",
  "git_commit": "%s",
  "git_branch": "%s",
  "build_date": "%s",
  "go_version": "%s",
  "platform": "%s"
}
`, info.Version, info.GitCommit, info.GitBranch, info.BuildDate, info.GoVersion, info.Platform)
}
