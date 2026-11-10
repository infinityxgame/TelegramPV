//go:build !windows

package server

import (
	"os/exec"
	"runtime"
)

func openURL(targetURL string) error {
	if runtime.GOOS == "darwin" {
		return exec.Command("open", targetURL).Start()
	}
	return exec.Command("xdg-open", targetURL).Start()
}
