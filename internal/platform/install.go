package platform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// RecommendedPackages is the full runtime dependency set install-deps installs.
var RecommendedPackages = []string{"openvpn", "iproute2", "procps", "ca-certificates"}

// Install installs the given logical packages via the detected package manager,
// streaming command output to stdout/stderr.
func Install(ctx context.Context, logical []string) error {
	if len(logical) == 0 {
		return nil
	}
	pm := DetectPkgManager()
	if pm == nil {
		return fmt.Errorf("no supported package manager found (need apt-get/apk/dnf/yum)")
	}
	fmt.Printf("Using %s to install: %v\n", pm.Name, logical)
	for _, step := range pm.steps(logical) {
		fmt.Printf("+ %v\n", step)
		cmd := exec.CommandContext(ctx, step[0], step[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%v failed: %w", step, err)
		}
	}
	return nil
}
