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

//go:build windows

package health

import "golang.org/x/sys/windows"

// diskUsage returns the total and used bytes of the volume holding path.
func diskUsage(path string) (totalBytes, usedBytes uint64, err error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, err
	}
	var freeToCaller, total, free uint64
	if err := windows.GetDiskFreeSpaceEx(p, &freeToCaller, &total, &free); err != nil {
		return 0, 0, err
	}
	return total, total - free, nil
}
