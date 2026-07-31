package lockfile

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ignoreDirs is pruned entirely (fs.SkipDir) during the source walk — build
// artifacts, dependency trees, and VCS/editor metadata that were never part
// of a mod's actual source. .code-review-graph is this monorepo's own
// tooling artifact, observed present inside barrakuda-mod-creator itself.
var ignoreDirs = map[string]bool{
	".git": true, ".github": true, ".hg": true, ".svn": true,
	"node_modules": true, "vendor": true, "dist": true, "build": true,
	"out": true, "target": true, ".venv": true, "venv": true,
	"__pycache__": true, ".pytest_cache": true, ".idea": true, ".vscode": true,
	".code-review-graph": true,
}

var ignoreFiles = map[string]bool{
	".DS_Store": true, "Thumbs.db": true,
}

const lockignoreFile = "lockignore"

// walkSource walks dir recursively, hashing every file not in exclude, not
// under an ignored directory, and not matched by an optional
// .barrakuda/lockignore (or root lockignore). Returned files are sorted by
// path for deterministic output. Symlinks are skipped (not followed) —
// cheap way to avoid cycles and cross-platform inconsistency, since Windows
// symlinks require elevated privileges most mod repos won't have anyway.
func walkSource(dir string, exclude map[string]bool) ([]FileHash, error) {
	patterns := loadLockignore(dir)

	var files []FileHash
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == dir {
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		name := d.Name()
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			rel = path
		}
		relSlash := filepath.ToSlash(rel)

		if d.IsDir() {
			if ignoreDirs[name] {
				return fs.SkipDir
			}
			return nil
		}

		if exclude[filepath.Clean(path)] {
			return nil
		}
		if ignoreFiles[name] {
			return nil
		}
		if matchesAny(patterns, name, relSlash) {
			return nil
		}

		hash, hashErr := hashFile(path)
		if hashErr != nil {
			return hashErr
		}
		files = append(files, FileHash{Path: relSlash, Hash: hash})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

// loadLockignore reads an optional glob-per-line exclusion file — first
// .barrakuda/lockignore, then a root lockignore. Blank lines and lines
// starting with '#' are skipped. Absent file is not an error: most mod
// repos have nothing to exclude.
func loadLockignore(dir string) []string {
	for _, candidate := range []string{
		filepath.Join(dir, ".barrakuda", lockignoreFile),
		filepath.Join(dir, lockignoreFile),
	} {
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		var patterns []string
		for line := range strings.SplitSeq(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			patterns = append(patterns, line)
		}
		return patterns
	}
	return nil
}

// matchesAny reports whether name (basename) or relSlash (path relative to
// the mod root, forward-slash) matches any lockignore pattern. filepath.Match
// semantics only — no negation, no recursive **, matching the simple glob
// most mod repos will ever need (nothing uses .gitignore today, so there's
// no existing convention to be compatible with).
func matchesAny(patterns []string, name, relSlash string) bool {
	for _, p := range patterns {
		if ok, _ := filepath.Match(p, name); ok {
			return true
		}
		if ok, _ := filepath.Match(p, relSlash); ok {
			return true
		}
	}
	return false
}
