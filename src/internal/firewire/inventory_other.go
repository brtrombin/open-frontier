//go:build !windows

package firewire

import "errors"

// Collect only works on Windows; on Linux, tools/phase1-collect.sh does this job.
func Collect() (Inventory, error) {
	return Inventory{}, errors.New("the firewire command runs on Windows; on Linux use tools/phase1-collect.sh")
}
