//go:build windows

package install

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
	"unsafe"

	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/runtime"
)

const (
	deferredUninstallEnv       = "AGENT47_INTERNAL_WINDOWS_UNINSTALL"
	deferredUninstallParentEnv = "AGENT47_INTERNAL_WINDOWS_UNINSTALL_PARENT"
	moveFileDelayUntilReboot   = 0x4
)

var moveFileExW = syscall.NewLazyDLL("kernel32.dll").NewProc("MoveFileExW")

func platformStartDeferredUninstall(cfg runtime.Config) error {
	tempDir, err := os.MkdirTemp("", "agent47-uninstall-")
	if err != nil {
		return err
	}
	helperPath := filepath.Join(tempDir, "afs-uninstall.exe")
	if err := copyDeferredExecutable(cfg.ExecutablePath, helperPath); err != nil {
		_ = os.RemoveAll(tempDir)
		return err
	}

	cmd := exec.Command(helperPath)
	cmd.Env = append(os.Environ(),
		deferredUninstallEnv+"=1",
		deferredUninstallParentEnv+"="+strconv.Itoa(os.Getpid()),
	)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		_ = os.RemoveAll(tempDir)
		return err
	}
	return cmd.Process.Release()
}

func RunDeferredUninstallIfRequested(ctx context.Context, cfg runtime.Config, out cli.Output) (bool, error) {
	if os.Getenv(deferredUninstallEnv) != "1" {
		return false, nil
	}
	parentPID, err := strconv.Atoi(os.Getenv(deferredUninstallParentEnv))
	if err != nil || parentPID <= 0 {
		return true, errors.New("invalid deferred uninstall parent process")
	}
	if err := waitForWindowsProcess(ctx, uint32(parentPID)); err != nil {
		return true, err
	}

	service, err := New(cfg, out)
	if err == nil {
		err = service.Uninstall(ctx, cfg)
	}
	if cleanupErr := scheduleHelperCleanup(cfg.ExecutablePath); cleanupErr != nil {
		out.Warn("Temporary uninstall helper will remain until system cleanup: %v", cleanupErr)
	}
	return true, err
}

func copyDeferredExecutable(source, target string) error {
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	dst, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o700)
	if err != nil {
		_ = src.Close()
		return err
	}
	_, copyErr := io.Copy(dst, src)
	return errors.Join(copyErr, dst.Close(), src.Close())
}

func waitForWindowsProcess(ctx context.Context, pid uint32) error {
	handle, err := syscall.OpenProcess(syscall.SYNCHRONIZE, false, pid)
	if err != nil {
		if errors.Is(err, syscall.Errno(87)) {
			return nil
		}
		return err
	}
	defer syscall.CloseHandle(handle)

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		event, err := syscall.WaitForSingleObject(handle, 250)
		if err != nil {
			return err
		}
		if event == syscall.WAIT_OBJECT_0 {
			return nil
		}
		if event != syscall.WAIT_TIMEOUT {
			return fmt.Errorf("unexpected Windows process wait result: %d", event)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	return errors.New("timed out waiting for uninstall parent process")
}

// scheduleHelperCleanup is the isolated Windows syscall boundary needed because
// a running executable cannot delete itself. Windows removes the helper and its
// now-empty temporary directory during the next boot.
func scheduleHelperCleanup(executablePath string) error {
	var errs []error
	for _, path := range []string{executablePath, filepath.Dir(executablePath)} {
		pathPtr, err := syscall.UTF16PtrFromString(path)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		result, _, callErr := moveFileExW.Call(uintptr(unsafe.Pointer(pathPtr)), 0, moveFileDelayUntilReboot)
		if result == 0 {
			errs = append(errs, callErr)
		}
	}
	return errors.Join(errs...)
}
