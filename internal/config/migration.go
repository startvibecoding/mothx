package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/startvibecoding/mothx/internal/platform"
)

const (
	ProjectDirName       = ".mothx"
	LegacyProjectDirName = ".vibe"
)

type DirMigration struct {
	Scope    string
	OldPath  string
	NewPath  string
	Migrated bool
	Skipped  bool
	Err      error
}

var migrationReports = struct {
	sync.Mutex
	seen map[string]bool
}{seen: map[string]bool{}}

// AutoMigrateLegacyDirs moves legacy VibeCoding directories to their MothX
// names when the destination does not already exist.
func AutoMigrateLegacyDirs(cwd string) []DirMigration {
	var results []DirMigration
	if r, ok := AutoMigrateGlobalDir(); ok {
		results = append(results, r)
	}
	if r, ok := AutoMigrateProjectDir(cwd); ok {
		results = append(results, r)
	}
	reportDirMigrations(results)
	return results
}

// AutoMigrateGlobalDir migrates the default global directory from .vibecoding
// to .mothx. Custom directories from MOTHX_DIR/VIBECODING_DIR are left alone.
func AutoMigrateGlobalDir() (DirMigration, bool) {
	if platform.ConfigDirOverridden() {
		return DirMigration{}, false
	}
	return migrateLegacyDir("global", platform.LegacyConfigDir(), platform.ConfigDir())
}

// AutoMigrateProjectDir migrates cwd/.vibe to cwd/.mothx when needed.
func AutoMigrateProjectDir(cwd string) (DirMigration, bool) {
	if cwd == "" {
		cwd = "."
	}
	return migrateLegacyDir(
		"project",
		filepath.Join(cwd, LegacyProjectDirName),
		filepath.Join(cwd, ProjectDirName),
	)
}

// ProjectPath returns a project-level path under .mothx in the current working
// directory. It also performs the one-time legacy .vibe migration.
func ProjectPath(elem ...string) string {
	return ProjectPathFor(".", elem...)
}

// ProjectPathFor returns a project-level path under cwd/.mothx. It also
// performs the one-time legacy .vibe migration for cwd.
func ProjectPathFor(cwd string, elem ...string) string {
	r, ok := AutoMigrateProjectDir(cwd)
	if ok {
		reportDirMigrations([]DirMigration{r})
	}
	if cwd == "" {
		cwd = "."
	}
	parts := append([]string{cwd, ProjectDirName}, elem...)
	return filepath.Join(parts...)
}

func migrateLegacyDir(scope, oldPath, newPath string) (DirMigration, bool) {
	result := DirMigration{Scope: scope, OldPath: oldPath, NewPath: newPath}
	if oldPath == "" || newPath == "" || filepath.Clean(oldPath) == filepath.Clean(newPath) {
		return result, false
	}

	oldInfo, err := os.Stat(oldPath)
	if err != nil {
		if os.IsNotExist(err) {
			return result, false
		}
		result.Err = err
		return result, true
	}
	if !oldInfo.IsDir() {
		result.Skipped = true
		result.Err = fmt.Errorf("legacy path is not a directory")
		return result, true
	}

	if _, err := os.Stat(newPath); err == nil {
		result.Skipped = true
		return result, true
	} else if !os.IsNotExist(err) {
		result.Err = err
		return result, true
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		result.Err = err
		return result, true
	}
	result.Migrated = true
	return result, true
}

func reportDirMigrations(results []DirMigration) {
	for _, r := range results {
		key := r.Scope + "\x00" + filepath.Clean(r.OldPath) + "\x00" + filepath.Clean(r.NewPath)
		migrationReports.Lock()
		if migrationReports.seen[key] {
			migrationReports.Unlock()
			continue
		}
		migrationReports.seen[key] = true
		migrationReports.Unlock()

		switch {
		case r.Migrated:
			fmt.Fprintf(os.Stderr, "MothX automatic migration completed: %s -> %s\n", r.OldPath, r.NewPath)
		case r.Skipped && r.Err != nil:
			fmt.Fprintf(os.Stderr, "Warning: MothX automatic migration skipped for %s -> %s: %v\n", r.OldPath, r.NewPath, r.Err)
		case r.Skipped:
			fmt.Fprintf(os.Stderr, "Warning: MothX automatic migration skipped for %s because %s already exists\n", r.OldPath, r.NewPath)
		case r.Err != nil:
			fmt.Fprintf(os.Stderr, "Warning: MothX automatic migration failed for %s -> %s: %v\n", r.OldPath, r.NewPath, r.Err)
		}
	}
}
