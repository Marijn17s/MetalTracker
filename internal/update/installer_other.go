//go:build !windows

package update

import (
	"fmt"
	"os/exec"
	"runtime"
)

func startInstaller(path string) error {
	var command *exec.Cmd
	if runtime.GOOS == "darwin" {
		command = exec.Command("open", path)
	} else {
		command = exec.Command(path)
	}
	if err := command.Start(); err != nil {
		return fmt.Errorf("start installer: %w", err)
	}
	return nil
}