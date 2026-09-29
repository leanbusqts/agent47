//go:build linux

package initrepo

import (
	"syscall"
	"unsafe"
)

// The syscall boundary is intentionally isolated in this file. Go 1.26 does
// not expose linkat in syscall, while descriptor-relative links are required to
// prevent a renamed parent directory from redirecting a transaction.

const atRemoveDir = 0x200

func openat(dirfd int, path string, flags int, mode uint32) (int, error) {
	return syscall.Openat(dirfd, path, flags, mode)
}

func mkdirat(dirfd int, path string, mode uint32) error {
	return syscall.Mkdirat(dirfd, path, mode)
}

func renameat(oldDirfd int, oldPath string, newDirfd int, newPath string) error {
	return syscall.Renameat(oldDirfd, oldPath, newDirfd, newPath)
}

func linkat(oldDirfd int, oldPath string, newDirfd int, newPath string) error {
	oldPointer, err := syscall.BytePtrFromString(oldPath)
	if err != nil {
		return err
	}
	newPointer, err := syscall.BytePtrFromString(newPath)
	if err != nil {
		return err
	}
	_, _, errno := syscall.Syscall6(syscall.SYS_LINKAT,
		uintptr(oldDirfd), uintptr(unsafe.Pointer(oldPointer)),
		uintptr(newDirfd), uintptr(unsafe.Pointer(newPointer)), 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

func unlinkat(dirfd int, path string, flags int) error {
	pointer, err := syscall.BytePtrFromString(path)
	if err != nil {
		return err
	}
	_, _, errno := syscall.Syscall(syscall.SYS_UNLINKAT,
		uintptr(dirfd), uintptr(unsafe.Pointer(pointer)), uintptr(flags))
	if errno != 0 {
		return errno
	}
	return nil
}
