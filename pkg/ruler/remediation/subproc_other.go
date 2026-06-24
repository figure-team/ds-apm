//go:build !linux

package remediation

import "os/exec"

// configureSubprocess is a no-op off Linux. The full process-group +
// parent-death containment is Linux-specific (PR_SET_PDEATHSIG); this
// deployment targets Linux. Off-Linux builds still run bash under the ctx
// timeout, just without group-level containment.
func configureSubprocess(cmd *exec.Cmd) {}

// killProcessTree kills the lead process off Linux (no process group set).
func killProcessTree(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
