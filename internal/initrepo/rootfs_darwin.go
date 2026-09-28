//go:build darwin

package initrepo

import (
	"syscall"
	"unsafe"
)

// The syscall boundary is intentionally isolated in this file. Go 1.22 does
// not expose the Darwin *at calls, while descriptor-relative operations are
// required to keep repository writes confined during directory swaps.

const (
	atRemoveDir    = 0x80
	darwinOpenat   = 463
	darwinRenameat = 465
	darwinMkdirat  = 475
	darwinLinkat   = 471
	darwinUnlinkat = 472
)

func openat(dirfd int, path string, flags int, mode uint32) (int, error) {
	pointer, err := syscall.BytePtrFromString(path)
	if err != nil {
		return 0, err
	}
	fd, _, errno := syscall.Syscall6(darwinOpenat,
		uintptr(dirfd), uintptr(unsafe.Pointer(pointer)), uintptr(flags), uintptr(mode), 0, 0)
	if errno != 0 {
		return 0, errno
	}
	return int(fd), nil
}

func mkdirat(dirfd int, path string, mode uint32) error {
	pointer, err := syscall.BytePtrFromString(path)
	if err != nil {
		return err
	}
	_, _, errno := syscall.Syscall6(darwinMkdirat,
		uintptr(dirfd), uintptr(unsafe.Pointer(pointer)), uintptr(mode), 0, 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

func renameat(oldDirfd int, oldPath string, newDirfd int, newPath string) error {
	oldPointer, err := syscall.BytePtrFromString(oldPath)
	if err != nil {
		return err
	}
	newPointer, err := syscall.BytePtrFromString(newPath)
	if err != nil {
		return err
	}
	_, _, errno := syscall.Syscall6(darwinRenameat,
		uintptr(oldDirfd), uintptr(unsafe.Pointer(oldPointer)),
		uintptr(newDirfd), uintptr(unsafe.Pointer(newPointer)), 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
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
	_, _, errno := syscall.Syscall6(darwinLinkat,
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
	_, _, errno := syscall.Syscall(darwinUnlinkat,
		uintptr(dirfd), uintptr(unsafe.Pointer(pointer)), uintptr(flags))
	if errno != 0 {
		return errno
	}
	return nil
}
