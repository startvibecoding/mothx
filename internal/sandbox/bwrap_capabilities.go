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
		UnsharePID:    contains("--unshare-pid"),
		UnshareIPC:    contains("--unshare-ipc"),
		UnshareUTS:    contains("--unshare-uts"),
		NewSession:    contains("--new-session"),
		DieWithParent: contains("--die-with-parent"),
		MountProc:     contains("--proc"),
		MountDev:      contains("--dev"),
		MountTmpfs:    contains("--tmpfs"),
		TmpfsSize:     contains("--size"),
		MountBind:     contains("--bind") && contains("--ro-bind"),
		ChangeDir:     contains("--chdir"),
		Hostname:      contains("--hostname"),
	}
	return caps, true
}
