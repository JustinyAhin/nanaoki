//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

// Setsid keeps the refresh alive when the agent kills the hook's process group.
func detach(c *exec.Cmd) { c.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }
