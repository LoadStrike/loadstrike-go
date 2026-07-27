package loadstrike

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	runtimeCacheDirectoryPermissions = os.FileMode(0o700)
	runtimeCacheRuntimePermissions   = os.FileMode(0o500)
	runtimeCacheSidecarPermissions   = os.FileMode(0o600)
)

type runtimeObjectIdentity struct {
	device uint64
	file   uint64
	links  uint64
}

type openedRuntimeObject struct {
	file     *os.File
	path     string
	identity runtimeObjectIdentity
	info     os.FileInfo
}

type resolvedRuntimeExecution struct {
	Path      string
	directory string
	closed    bool
}

func runtimeCacheRuntimeMode() os.FileMode {
	if runtime.GOOS == "windows" {
		// Windows immutability is enforced with the protected DACL. Creating a
		// file with no owner-write mode sets the DOS read-only attribute, which
		// can prevent secure cleanup without adding write permissions back.
		return runtimeCacheSidecarPermissions
	}
	return runtimeCacheRuntimePermissions
}

func runtimeCacheSidecarMode() os.FileMode {
	return runtimeCacheSidecarPermissions
}

func selectRuntimeCacheRoot(
	explicit string,
	userCacheDirectory func() (string, error),
) (string, error) {
	root := strings.TrimSpace(explicit)
	if root == "" {
		if userCacheDirectory == nil {
			return "", errors.New("resolve per-user runtime cache: cache directory provider is unavailable")
		}
		var err error
		root, err = userCacheDirectory()
		if err != nil {
			return "", fmt.Errorf("resolve per-user runtime cache: %w", err)
		}
		root = strings.TrimSpace(root)
		if root == "" {
			return "", errors.New("resolve per-user runtime cache: cache directory is empty")
		}
	}

	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve per-user runtime cache path: %w", err)
	}
	return filepath.Clean(absolute), nil
}

func ensureRuntimeCacheDirectory(root string, target string) error {
	root, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return fmt.Errorf("resolve runtime cache root: %w", err)
	}
	target, err = filepath.Abs(filepath.Clean(target))
	if err != nil {
		return fmt.Errorf("resolve runtime cache directory: %w", err)
	}
	if !runtimePathWithinRoot(root, target) {
		return fmt.Errorf("runtime cache directory %q is outside the per-user cache root", target)
	}

	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return fmt.Errorf("validate runtime cache root: %w", err)
	}
	if !runtimePathsEqual(filepath.Clean(resolvedRoot), root) {
		return errors.New("runtime cache root or its ancestry contains a link or reparse point")
	}
	rootObject, err := openStableRuntimeObject(root, true, false, false, false)
	if err != nil {
		return fmt.Errorf("validate runtime cache root: %w", err)
	}
	if err := rootObject.Close(); err != nil {
		return fmt.Errorf("close runtime cache root: %w", err)
	}

	if runtimePathsEqual(root, target) {
		return nil
	}
	relative, err := filepath.Rel(root, target)
	if err != nil {
		return fmt.Errorf("resolve runtime cache hierarchy: %w", err)
	}

	current := root
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "" || component == "." || component == ".." {
			return errors.New("runtime cache hierarchy contains an ambiguous path component")
		}
		current = filepath.Join(current, component)
		info, lstatErr := os.Lstat(current)
		switch {
		case lstatErr == nil:
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return fmt.Errorf("runtime cache hierarchy contains a linked or non-directory component: %s", current)
			}
			object, err := openStableRuntimeObject(
				current,
				true,
				false,
				true,
				false,
			)
			if err != nil {
				return fmt.Errorf("validate runtime cache directory %q: %w", current, err)
			}
			if err := protectOpenedRuntimeObject(
				object.file,
				true,
				runtimeCacheDirectoryPermissions,
			); err != nil {
				_ = object.Close()
				return fmt.Errorf("protect runtime cache directory %q: %w", current, err)
			}
			if err := object.Close(); err != nil {
				return fmt.Errorf("close runtime cache directory %q: %w", current, err)
			}
		case os.IsNotExist(lstatErr):
			if err := createProtectedRuntimeDirectory(current); err != nil {
				return fmt.Errorf("create runtime cache directory %q: %w", current, err)
			}
		default:
			return fmt.Errorf("inspect runtime cache directory %q: %w", current, lstatErr)
		}
		object, err := openStableRuntimeObject(
			current,
			true,
			false,
			true,
			true,
		)
		if err != nil {
			return fmt.Errorf("validate runtime cache directory %q: %w", current, err)
		}
		closeErr := object.Close()
		if closeErr != nil {
			return fmt.Errorf("close runtime cache directory %q: %w", current, closeErr)
		}
		if runtime.GOOS != "windows" {
			refreshed, err := os.Lstat(current)
			if err != nil {
				return fmt.Errorf("revalidate runtime cache directory %q: %w", current, err)
			}
			if refreshed.Mode().Perm() != runtimeCacheDirectoryPermissions {
				return fmt.Errorf(
					"runtime cache directory %q permissions are %04o, expected 0700",
					current,
					refreshed.Mode().Perm(),
				)
			}
		}
	}
	return nil
}

func openStableRuntimeObject(
	path string,
	wantDirectory bool,
	requireSingleLink bool,
	requirePrivateAccess bool,
	requireProtectedAccess bool,
) (*openedRuntimeObject, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("runtime cache path is a link: %s", path)
	}
	if wantDirectory && !before.IsDir() {
		return nil, fmt.Errorf("runtime cache path is not a directory: %s", path)
	}
	if !wantDirectory && !before.Mode().IsRegular() {
		return nil, fmt.Errorf("runtime cache path is not a regular file: %s", path)
	}

	file, err := openRuntimeObjectNoFollow(path, wantDirectory)
	if err != nil {
		return nil, err
	}
	closeOnError := true
	defer func() {
		if closeOnError {
			_ = file.Close()
		}
	}()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !os.SameFile(before, info) {
		return nil, fmt.Errorf("runtime cache path changed while it was opened: %s", path)
	}
	identity, err := runtimeObjectIdentityForFile(file)
	if err != nil {
		return nil, err
	}
	if requireSingleLink && identity.links != 1 {
		return nil, fmt.Errorf(
			"runtime cache file must have exactly one link, found %d: %s",
			identity.links,
			path,
		)
	}
	if err := validateRuntimeObjectSecurity(
		file,
		wantDirectory,
		requirePrivateAccess,
		requireProtectedAccess,
	); err != nil {
		return nil, err
	}
	after, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !os.SameFile(info, after) {
		return nil, fmt.Errorf("runtime cache path changed during validation: %s", path)
	}

	closeOnError = false
	return &openedRuntimeObject{
		file:     file,
		path:     path,
		identity: identity,
		info:     info,
	}, nil
}

func (o *openedRuntimeObject) ensureStable() error {
	if o == nil || o.file == nil {
		return errors.New("runtime cache object is not open")
	}
	current, err := o.file.Stat()
	if err != nil {
		return err
	}
	currentIdentity, err := runtimeObjectIdentityForFile(o.file)
	if err != nil {
		return err
	}
	if currentIdentity != o.identity ||
		current.Size() != o.info.Size() ||
		current.ModTime() != o.info.ModTime() ||
		current.Mode().Type() != o.info.Mode().Type() {
		return fmt.Errorf("runtime cache file changed while it was being read: %s", o.path)
	}
	atPath, err := os.Lstat(o.path)
	if err != nil {
		return err
	}
	if !os.SameFile(current, atPath) {
		return fmt.Errorf("runtime cache path changed while it was being read: %s", o.path)
	}
	return nil
}

func (o *openedRuntimeObject) Close() error {
	if o == nil || o.file == nil {
		return nil
	}
	err := o.file.Close()
	o.file = nil
	return err
}

func verifyRuntimeCacheFileMode(path string, info os.FileInfo, expected os.FileMode) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	permissions := info.Mode().Perm()
	if permissions == expected {
		return nil
	}
	if permissions&0o022 != 0 {
		return fmt.Errorf(
			"runtime cache file %q is writable by a group or other users (%04o)",
			path,
			permissions,
		)
	}
	return nil
}

func hashOpenedRuntimeObject(object *openedRuntimeObject) (string, error) {
	if _, err := object.file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	hasher := sha256.New()
	written, err := io.Copy(hasher, io.LimitReader(object.file, maxRuntimeArtifactBytes+1))
	if err != nil {
		return "", err
	}
	if written > maxRuntimeArtifactBytes {
		return "", fmt.Errorf("runtime artifact exceeds the %d byte safety limit", maxRuntimeArtifactBytes)
	}
	if err := object.ensureStable(); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func prepareRuntimeExecutionCopy(
	sourcePath string,
	expectedSHA256 string,
	afterSourceLstat func() error,
) (*resolvedRuntimeExecution, error) {
	if !isLowerHex64(expectedSHA256) {
		return nil, errors.New("runtime checksum must be exactly 64 lowercase hexadecimal characters")
	}
	before, err := os.Lstat(sourcePath)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, errors.New("cached runtime must be a non-linked regular file")
	}
	if afterSourceLstat != nil {
		if err := afterSourceLstat(); err != nil {
			return nil, fmt.Errorf("runtime cache swap test hook: %w", err)
		}
	}

	source, err := openStableRuntimeObject(
		sourcePath,
		false,
		true,
		false,
		false,
	)
	if err != nil {
		return nil, err
	}
	defer source.Close()
	if !os.SameFile(before, source.info) {
		return nil, errors.New("runtime cache path changed before it could be copied")
	}
	if err := verifyRuntimeCacheFileMode(
		sourcePath,
		source.info,
		runtimeCacheRuntimePermissions,
	); err != nil {
		return nil, err
	}
	if err := protectOpenedRuntimeObject(
		source.file,
		false,
		runtimeCacheRuntimePermissions,
	); err != nil {
		return nil, fmt.Errorf("protect cached runtime before execution copy: %w", err)
	}

	executionDirectory, err := os.MkdirTemp(
		filepath.Dir(sourcePath),
		".loadstrike-execution-*",
	)
	if err != nil {
		return nil, fmt.Errorf("create private runtime execution directory: %w", err)
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(executionDirectory)
		}
	}()
	if err := protectRuntimeObject(
		executionDirectory,
		true,
		runtimeCacheDirectoryPermissions,
	); err != nil {
		return nil, fmt.Errorf("protect private runtime execution directory: %w", err)
	}

	executionPath := filepath.Join(executionDirectory, filepath.Base(sourcePath))
	destination, err := os.OpenFile(
		executionPath,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		runtimeCacheSidecarPermissions,
	)
	if err != nil {
		return nil, fmt.Errorf("create private runtime execution file: %w", err)
	}
	destinationClosed := false
	defer func() {
		if !destinationClosed {
			_ = destination.Close()
		}
	}()

	if _, err := source.file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("rewind cached runtime: %w", err)
	}
	hasher := sha256.New()
	written, err := io.Copy(
		io.MultiWriter(destination, hasher),
		io.LimitReader(source.file, maxRuntimeArtifactBytes+1),
	)
	if err != nil {
		return nil, fmt.Errorf("copy verified runtime for execution: %w", err)
	}
	if written > maxRuntimeArtifactBytes {
		return nil, fmt.Errorf("runtime artifact exceeds the %d byte safety limit", maxRuntimeArtifactBytes)
	}
	actualSHA256 := hex.EncodeToString(hasher.Sum(nil))
	if actualSHA256 != expectedSHA256 {
		return nil, fmt.Errorf(
			"runtime checksum mismatch while preparing execution copy: expected %s, got %s",
			expectedSHA256,
			actualSHA256,
		)
	}
	if err := source.ensureStable(); err != nil {
		return nil, err
	}
	if err := destination.Sync(); err != nil {
		return nil, fmt.Errorf("sync private runtime execution file: %w", err)
	}
	if err := destination.Close(); err != nil {
		return nil, fmt.Errorf("close private runtime execution file: %w", err)
	}
	destinationClosed = true
	if err := protectRuntimeObject(
		executionPath,
		false,
		runtimeCacheRuntimePermissions,
	); err != nil {
		return nil, fmt.Errorf("protect private runtime execution file: %w", err)
	}
	if err := verifyRuntimeArtifactFile(executionPath, expectedSHA256); err != nil {
		return nil, fmt.Errorf("revalidate private runtime execution file: %w", err)
	}

	cleanup = false
	return &resolvedRuntimeExecution{
		Path:      executionPath,
		directory: executionDirectory,
	}, nil
}

func (e *resolvedRuntimeExecution) Close() error {
	if e == nil || e.closed {
		return nil
	}
	e.closed = true
	if e.directory == "" {
		return nil
	}
	parent := filepath.Dir(e.directory)
	name := filepath.Base(e.directory)
	if !strings.HasPrefix(name, ".loadstrike-execution-") ||
		!runtimePathWithinRoot(parent, e.directory) {
		return errors.New("refusing to remove an unrecognized runtime execution directory")
	}
	return os.RemoveAll(e.directory)
}

func runtimePathWithinRoot(root string, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relative == "." ||
		(relative != ".." &&
			!strings.HasPrefix(relative, ".."+string(filepath.Separator)) &&
			!filepath.IsAbs(relative))
}

func runtimePathsEqual(left string, right string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}
