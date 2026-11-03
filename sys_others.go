//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func showNotice(msg string) {
	fmt.Fprintf(os.Stderr, "[TelegramPV] %s\n", msg)
}

func isTelegramDLRunning() bool {
	out, err := exec.Command("pgrep", "-f", "TelegramDL").Output()
	if err == nil && len(strings.TrimSpace(string(out))) > 0 {
		return true
	}
	out2, err2 := exec.Command("pgrep", "-f", "tgdown").Output()
	if err2 == nil && len(strings.TrimSpace(string(out2))) > 0 {
		return true
	}
	return false
}
