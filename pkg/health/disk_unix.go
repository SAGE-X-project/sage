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

//go:build !windows

package health

import (
	"fmt"
	"syscall"
)

// diskUsage returns the total and used bytes of the filesystem holding path.
func diskUsage(path string) (totalBytes, usedBytes uint64, err error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, err
	}
	if stat.Bsize <= 0 {
		return 0, 0, fmt.Errorf("invalid block size from filesystem stats")
	}
	// #nosec G115 - Bsize checked positive above
	bsize := uint64(stat.Bsize)
	totalBytes = stat.Blocks * bsize
	usedBytes = totalBytes - stat.Bfree*bsize
	return totalBytes, usedBytes, nil
}
