//go:build !linux && !darwin

package registry011

import "os"

func retentionLock(*os.File) error   { return ErrRetentionPlatform }
func retentionUnlock(*os.File) error { return ErrRetentionPlatform }
