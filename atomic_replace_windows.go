//go:build windows

package loadstrike

import "golang.org/x/sys/windows"

func atomicReplaceFile(sourcePath string, targetPath string) error {
	source, err := windows.UTF16PtrFromString(sourcePath)
	if err != nil {
		return err
	}
	target, err := windows.UTF16PtrFromString(targetPath)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(
		source,
		target,
		windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH,
	)
}

func syncRuntimeCacheDirectory(string) error {
	// MOVEFILE_WRITE_THROUGH flushes the rename operation before returning on Windows.
	return nil
}
