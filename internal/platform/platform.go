// Package platform provides cross-platform compatibility utilities.
package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	appDirName       = "mothx"
	legacyAppDirName = "vibecoding"
)

// ──────────────────────────────────────────────────────────────────────────────
// OS detection
// ──────────────────────────────────────────────────────────────────────────────

// OS returns the current operating system: "windows", "darwin", "linux", etc.
func OS() string {
	return runtime.GOOS
}

// IsWindows returns true if running on Windows.
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// IsMacOS returns true if running on macOS.
func IsMacOS() bool {
	return runtime.GOOS == "darwin"
}

// IsLinux returns true if running on Linux.
func IsLinux() bool {
	return runtime.GOOS == "linux"
}

// IsFreeBSD returns true if running on FreeBSD.
func IsFreeBSD() bool {
	return runtime.GOOS == "freebsd"
}

// IsOpenBSD returns true if running on OpenBSD.
func IsOpenBSD() bool {
	return runtime.GOOS == "openbsd"
}

// IsNetBSD returns true if running on NetBSD.
func IsNetBSD() bool {
	return runtime.GOOS == "netbsd"
}

// IsDragonflyBSD returns true if running on DragonFly BSD.
func IsDragonflyBSD() bool {
	return runtime.GOOS == "dragonfly"
}

// IsBSD returns true if running on any BSD variant (FreeBSD, OpenBSD, NetBSD, DragonFly BSD).
func IsBSD() bool {
	switch runtime.GOOS {
	case "freebsd", "openbsd", "netbsd", "dragonfly":
		return true
	default:
		return false
	}
}

// IsSolaris returns true if running on Solaris or illumos.
func IsSolaris() bool {
	switch runtime.GOOS {
	case "solaris", "illumos":
		return true
	default:
		return false
	}
}

// IsAIX returns true if running on AIX.
func IsAIX() bool {
	return runtime.GOOS == "aix"
}

// IsPlan9 returns true if running on Plan 9.
func IsPlan9() bool {
	return runtime.GOOS == "plan9"
}

// IsUnix returns true if running on a Unix-like OS (Linux, macOS, BSD, Solaris, illumos, AIX).
func IsUnix() bool {
	switch runtime.GOOS {
	case "windows", "plan9":
		return false
	default:
		return true
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Architecture detection
// ──────────────────────────────────────────────────────────────────────────────

// Arch returns the current architecture: "amd64", "arm64", "386", etc.
func Arch() string {
	return runtime.GOARCH
}

// IsAMD64 returns true if running on amd64 (x86-64).
func IsAMD64() bool {
	return runtime.GOARCH == "amd64"
}

// IsARM64 returns true if running on arm64 (AArch64).
func IsARM64() bool {
	return runtime.GOARCH == "arm64"
}

// IsARM returns true if running on 32-bit ARM.
func IsARM() bool {
	return runtime.GOARCH == "arm"
}

// Is386 returns true if running on 32-bit x86.
func Is386() bool {
	return runtime.GOARCH == "386"
}

// Is64Bit returns true if the architecture is 64-bit.
func Is64Bit() bool {
	switch runtime.GOARCH {
	case "amd64", "arm64", "ppc64", "ppc64le", "mips64", "mips64le",
		"s390x", "riscv64", "loong64":
		return true
	default:
		return false
	}
}

// IsLittleEndian returns true if the architecture is little-endian.
// Returns true for most modern architectures; false for s390x, ppc64 (BE), mips64 (BE).
func IsLittleEndian() bool {
	switch runtime.GOARCH {
	case "amd64", "arm64", "arm", "386", "riscv64", "loong64",
		"mips64le", "mipsle", "ppc64le":
		return true
	default:
		return false
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Directory helpers
// ──────────────────────────────────────────────────────────────────────────────

// HomeDir returns the user's home directory.
func HomeDir() string {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return home
	}
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		return cwd
	}
	return string(os.PathSeparator)
}

// ConfigDir returns the platform-specific configuration directory.
func ConfigDir() string {
	if dir := os.Getenv("MOTHX_DIR"); dir != "" {
		return dir
	}
	if dir := os.Getenv("VIBECODING_DIR"); dir != "" {
		return dir
	}

	return configDirForOS(runtime.GOOS, HomeDir(), os.Getenv("APPDATA"))
}

func configDirForOS(goos, home, appData string) string {
	switch goos {
	case "windows":
		if appData != "" {
			return filepath.Join(appData, appDirName)
		}
		return filepath.Join(home, "AppData", "Roaming", appDirName)
	default: // unix-like and others
		return filepath.Join(home, "."+appDirName)
	}
}

// ConfigDirOverridden reports whether the user selected a custom config dir.
func ConfigDirOverridden() bool {
	return os.Getenv("MOTHX_DIR") != "" || os.Getenv("VIBECODING_DIR") != ""
}

// LegacyConfigDir returns the pre-MothX default global configuration directory.
func LegacyConfigDir() string {
	return legacyConfigDirForOS(runtime.GOOS, HomeDir(), os.Getenv("APPDATA"))
}

func legacyConfigDirForOS(goos, home, appData string) string {
	switch goos {
	case "windows":
		if appData != "" {
			return filepath.Join(appData, legacyAppDirName)
		}
		return filepath.Join(home, "AppData", "Roaming", legacyAppDirName)
	default:
		return filepath.Join(home, "."+legacyAppDirName)
	}
}

// DataDir returns the platform-specific data directory.
func DataDir() string {
	return ConfigDir()
}

// CacheDir returns the platform-specific cache directory.
func CacheDir() string {
	switch runtime.GOOS {
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData != "" {
			return filepath.Join(localAppData, appDirName, "cache")
		}
		return filepath.Join(HomeDir(), "AppData", "Local", appDirName, "cache")
	case "darwin":
		return filepath.Join(HomeDir(), "Library", "Caches", appDirName)
	default: // linux, BSD, Solaris, illumos, AIX, and others
		cacheHome := os.Getenv("XDG_CACHE_HOME")
		if cacheHome != "" {
			return filepath.Join(cacheHome, appDirName)
		}
		return filepath.Join(HomeDir(), ".cache", appDirName)
	}
}

// OpenFile opens path with the platform default application.
func OpenFile(path string) error {
	var candidates [][]string
	switch runtime.GOOS {
	case "darwin":
		candidates = [][]string{{"open", path}}
	case "windows":
		candidates = [][]string{{"cmd", "/c", "start", "", path}}
	default:
		candidates = [][]string{{"xdg-open", path}, {"gio", "open", path}}
	}
	for _, candidate := range candidates {
		if _, err := exec.LookPath(candidate[0]); err != nil {
			continue
		}
		if err := exec.Command(candidate[0], candidate[1:]...).Start(); err != nil {
			return err
		}
		return nil
	}
	return &exec.Error{Name: candidates[0][0], Err: exec.ErrNotFound}
}

// SessionDir returns the platform-specific session directory.
func SessionDir() string {
	return filepath.Join(ConfigDir(), "sessions")
}

// SkillsDir returns the platform-specific skills directory.
func SkillsDir() string {
	return filepath.Join(ConfigDir(), "skills")
}

// ──────────────────────────────────────────────────────────────────────────────
// Shell helpers
// ──────────────────────────────────────────────────────────────────────────────

// DefaultShell returns the default shell for the current platform.
func DefaultShell() string {
	if shell := os.Getenv("SHELL"); isExecutableAbsolutePath(shell) {
		return shell
	}

	switch runtime.GOOS {
	case "windows":
		// Try PowerShell first, then cmd
		if _, err := exec.LookPath("powershell.exe"); err == nil {
			return "powershell.exe"
		}
		return "cmd.exe"
	case "darwin":
		return "/bin/zsh"
	case "linux":
		return "/bin/bash"
	case "plan9":
		return "/bin/rc"
	default: // BSD, Solaris, illumos, AIX, and others
		return "/bin/sh"
	}
}

func isExecutableAbsolutePath(path string) bool {
	if path == "" || !filepath.IsAbs(path) {
		return false
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&0111 != 0
}

// ShellArgs returns the arguments to execute a command in the shell.
func ShellArgs(shell, command string) []string {
	normalizedShell := strings.ToLower(shell)
	switch {
	case strings.Contains(normalizedShell, "busybox"):
		return []string{"sh", "-c", command}
	case strings.Contains(normalizedShell, "powershell"):
		return []string{"-NoProfile", "-NonInteractive", "-Command", command}
	case strings.Contains(normalizedShell, "cmd"):
		return []string{"/c", command}
	case strings.Contains(normalizedShell, "rc"):
		return []string{"-c", command}
	default: // bash, zsh, sh, ksh, csh, etc.
		return []string{"-c", command}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Path helpers
// ──────────────────────────────────────────────────────────────────────────────

// PathSeparator returns the platform-specific path separator.
func PathSeparator() string {
	return string(os.PathSeparator)
}

// JoinPath joins path elements using the platform-specific separator.
func JoinPath(elem ...string) string {
	return filepath.Join(elem...)
}

// NormalizePath normalizes a path for the current platform.
// Converts forward slashes to backslashes on Windows.
func NormalizePath(path string) string {
	return filepath.FromSlash(path)
}

// ExpandHome expands ~ to the user's home directory.
func ExpandHome(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}

	home := HomeDir()
	if home == "" {
		return path
	}

	if path == "~" {
		return home
	}

	if len(path) > 1 && (path[1] == '/' || path[1] == '\\') {
		return filepath.Join(home, path[2:])
	}

	return path
}

// ──────────────────────────────────────────────────────────────────────────────
// Platform-specific paths and environment
// ──────────────────────────────────────────────────────────────────────────────

// CommonPaths returns platform-specific common system paths.
func CommonPaths() map[string]string {
	switch runtime.GOOS {
	case "windows":
		return map[string]string{
			"home":         HomeDir(),
			"temp":         os.TempDir(),
			"appData":      os.Getenv("APPDATA"),
			"localApp":     os.Getenv("LOCALAPPDATA"),
			"programFiles": os.Getenv("ProgramFiles"),
		}
	case "darwin":
		return map[string]string{
			"home":       HomeDir(),
			"temp":       os.TempDir(),
			"appSupport": filepath.Join(HomeDir(), "Library", "Application Support"),
			"caches":     filepath.Join(HomeDir(), "Library", "Caches"),
		}
	case "plan9":
		return map[string]string{
			"home": HomeDir(),
			"temp": os.TempDir(),
		}
	default: // linux, BSD, Solaris, illumos, AIX, and others
		return map[string]string{
			"home":   HomeDir(),
			"temp":   os.TempDir(),
			"cache":  filepath.Join(HomeDir(), ".cache"),
			"config": filepath.Join(HomeDir(), ".config"),
			"local":  filepath.Join(HomeDir(), ".local"),
		}
	}
}

// SandboxPaths returns paths that should be accessible in sandbox mode.
func SandboxPaths() []string {
	switch runtime.GOOS {
	case "windows":
		return []string{
			"C:\\Windows",
			"C:\\Program Files",
			"C:\\Program Files (x86)",
		}
	case "darwin":
		return []string{
			"/usr",
			"/lib",
			"/bin",
			"/sbin",
			"/System",
			"/Library",
		}
	case "linux":
		return []string{
			"/usr",
			"/lib",
			"/lib64",
			"/bin",
			"/sbin",
			"/etc/ld.so.cache",
			"/etc/ssl",
			"/etc/ca-certificates",
			"/dev/null",
			"/dev/urandom",
			"/dev/zero",
			"/proc/self",
			"/proc/meminfo",
			"/proc/cpuinfo",
		}
	default: // BSD, Solaris, illumos, AIX, Plan9, and others
		return []string{}
	}
}

// DeniedPaths returns paths that should be denied in sandbox mode.
func DeniedPaths() []string {
	switch runtime.GOOS {
	case "windows":
		return []string{
			filepath.Join(HomeDir(), "Documents"),
			filepath.Join(HomeDir(), "Desktop"),
		}
	case "plan9":
		return []string{}
	default: // linux, macOS, BSD, Solaris, illumos, AIX, and others
		return []string{
			"/etc/shadow",
			"/etc/gshadow",
			"/etc/passwd",
			"/root",
			"/home",
		}
	}
}

// DefaultEnvVars returns environment variables to pass through sandbox.
func DefaultEnvVars() []string {
	common := []string{
		"PATH",
		"HOME",
		"USER",
		"LANG",
		"LC_ALL",
		"TERM",
	}

	switch runtime.GOOS {
	case "windows":
		return append(common,
			"APPDATA",
			"LOCALAPPDATA",
			"COMPUTERNAME",
			"USERPROFILE",
			"SYSTEMROOT",
		)
	case "darwin":
		return append(common,
			"SHELL",
			"TMPDIR",
		)
	case "plan9":
		return []string{
			"path",
			"home",
			"user",
			"service",
		}
	default: // linux, BSD, Solaris, illumos, AIX, and others
		return append(common,
			"SHELL",
			"GOPATH",
			"GOROOT",
			"GOPROXY",
			"GOMODCACHE",
			"NODE_PATH",
		)
	}
}

// TempDir returns the platform-specific temp directory.
func TempDir() string {
	return os.TempDir()
}

// ExecutableExt returns the platform-specific executable extension.
func ExecutableExt() string {
	if IsWindows() {
		return ".exe"
	}
	return ""
}

// IsExecutable checks if a file is executable on the current platform.
func IsExecutable(info os.FileMode) bool {
	if IsWindows() {
		// On Windows, check file extension
		return true // Simplified for now
	}
	return info&0111 != 0
}
