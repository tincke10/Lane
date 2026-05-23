// Package convention validates that a project's docker-compose file
// references the env vars Lane allocates. Lane's port injection only
// works when the user's compose file actually reads ${APP_PORT:-80}
// or similar — without that, Lane is silently doing nothing useful.
package convention

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

var composeCandidates = []string{
	"docker-compose.yml",
	"docker-compose.yaml",
	"compose.yml",
	"compose.yaml",
}

// Validate reads dir's docker-compose file (the first that exists among
// the canonical filenames) and returns the subset of expected env vars
// that are NOT referenced anywhere inside.
//
// When no compose file is present, Validate returns nil, nil — the
// convention check is silently skipped for non-docker stacks.
func Validate(dir string, expected []string) ([]string, error) {
	data, found, err := readCompose(dir)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}

	missing := make([]string, 0)
	for _, v := range expected {
		if !referencesVar(data, v) {
			missing = append(missing, v)
		}
	}
	sort.Strings(missing)
	return missing, nil
}

func readCompose(dir string) ([]byte, bool, error) {
	for _, name := range composeCandidates {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err == nil {
			return data, true, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, false, fmt.Errorf("read %s: %w", name, err)
		}
	}
	return nil, false, nil
}

// referencesVar reports whether data contains a shell-style reference
// to env var `name`. Accepted forms:
//
//	${NAME}            standard brace form
//	${NAME:-default}   brace with default
//	${NAME:?error}     brace with required-error
//	$NAME              bare form, followed by a non-word boundary
//
// A prefix collision like $NAME_EXTRA must NOT match NAME, hence the
// word boundary in the bare-form branch and the `[}:]` restriction on
// the brace-form branch.
func referencesVar(data []byte, name string) bool {
	q := regexp.QuoteMeta(name)
	pattern := `\$(?:\{` + q + `[}:]|` + q + `\b)`
	return regexp.MustCompile(pattern).Match(data)
}
