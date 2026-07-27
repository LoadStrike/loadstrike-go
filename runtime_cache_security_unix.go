//go:build !windows

package loadstrike

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func openRuntimeObjectNoFollow(path string, directory bool) (*os.File, error) {
	flags := unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW
	if directory {
		flags |= unix.O_DIRECTORY
	}
	descriptor, err := unix.Open(path, flags, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(descriptor), path), nil
}

func runtimeObjectIdentityForFile(file *os.File) (runtimeObjectIdentity, error) {
	var status unix.Stat_t
	if err := unix.Fstat(int(file.Fd()), &status); err != nil {
		return runtimeObjectIdentity{}, err
	}
	return runtimeObjectIdentity{
		device: uint64(status.Dev),
		file:   uint64(status.Ino),
		links:  uint64(status.Nlink),
	}, nil
}

func validateRuntimeObjectSecurity(
	file *os.File,
	directory bool,
	requirePrivateAccess bool,
	requireProtectedAccess bool,
) error {
	var status unix.Stat_t
	if err := unix.Fstat(int(file.Fd()), &status); err != nil {
		return err
	}
	if status.Uid != uint32(os.Geteuid()) {
		return fmt.Errorf("runtime cache object is not owned by the current user: %s", file.Name())
	}
	if status.Mode&0o022 != 0 {
		return fmt.Errorf("runtime cache object is writable by a group or other users: %s", file.Name())
	}
	return nil
}

func protectRuntimeObject(path string, directory bool, permissions os.FileMode) error {
	return os.Chmod(path, permissions)
}

func protectOpenedRuntimeObject(
	file *os.File,
	directory bool,
	permissions os.FileMode,
) error {
	return file.Chmod(permissions)
}

func createProtectedRuntimeDirectory(path string) error {
	return os.Mkdir(path, runtimeCacheDirectoryPermissions)
}
