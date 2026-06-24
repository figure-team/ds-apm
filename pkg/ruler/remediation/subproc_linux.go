//go:build linux

package remediation

import (
	"os/exec"
	"syscall"
)

// configureSubprocess puts the child in its own process group and asks the
// kernel to SIGKILL it if this process dies (PR_SET_PDEATHSIG). The process
// group lets us kill the whole script tree, not just the lead pid; Pdeathsig is
// defense-in-depth against an orphaned bash if the server is hard-killed.
func configureSubprocess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:   true,
		Pdeathsig: syscall.SIGKILL,
	}
}

// killProcessTree SIGKILLs the child's entire process group. The child is the
// group leader (Setpgid), so signalling the negative pid reaches every
// descendant in the group, not just the lead process.
func killProcessTree(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
		// Group gone or already reaped; fall back to the lead pid.
		return cmd.Process.Kill()
	}
	return nil
}
