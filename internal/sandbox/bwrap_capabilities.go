package sandbox

import (
	"os/exec"
	"strings"
)

func probeBwrapCapabilities(path string) (BwrapCapabilities, bool) {
	if path == "" {
		return BwrapCapabilities{}, false
	}
	output, err := exec.Command(path, "--help").CombinedOutput()
	if err != nil {
		return BwrapCapabilities{}, false
	}
	help := string(output)
	contains := func(flag string) bool { return strings.Contains(help, flag) }
	caps := BwrapCapabilities{
		UnshareUser:   contains("--unshare-user"),
		NewSession:    contains("--new-session"),
		DieWithParent: contains("--die-with-parent"),
		MountProc:     contains("--proc"),
		MountTmpfs:    contains("--tmpfs"),
		MountBind:     contains("--bind") && contains("--ro-bind"),
		NetworkNS:     contains("--unshare-net"),
	}
	return caps, true
}
