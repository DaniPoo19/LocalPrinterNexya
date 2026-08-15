//go:build !windows

package ui

import "os/exec"

func OpenAppWindow(url string) {
	_ = exec.Command("xdg-open", url).Start()
}
