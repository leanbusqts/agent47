package platform

import "runtime"

func OS() string {
	return runtime.GOOS
}

func IsWindows() bool {
	return OS() == "windows"
}
