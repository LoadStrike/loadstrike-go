//go:build !windows

package loadstrike

import "os"

func atomicReplaceFile(sourcePath string, targetPath string) error {
	return os.Rename(sourcePath, targetPath)
}

func syncRuntimeCacheDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
