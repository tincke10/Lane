// Package stack inspects a project directory to detect which technologies
// it uses (laravel, vite, docker, mysql, etc.). The result feeds Lane's
// init flow so the registry knows which env vars to allocate.
package stack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Detect inspects the top-level of dir and returns a sorted set of
// technology markers. Detect does NOT recurse. When a known config file
// is present but malformed, the base marker (e.g. "php", "node") is still
// reported and the deeper marker (e.g. "laravel", "vite") is skipped.
func Detect(dir string) ([]string, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("stat project path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", dir)
	}

	markers := make(map[string]struct{})
	detectComposer(dir, markers)
	detectPackageJSON(dir, markers)
	detectPython(dir, markers)
	detectDocker(dir, markers)

	out := make([]string, 0, len(markers))
	for m := range markers {
		out = append(out, m)
	}
	sort.Strings(out)
	return out, nil
}

func detectComposer(dir string, m map[string]struct{}) {
	data, err := os.ReadFile(filepath.Join(dir, "composer.json"))
	if err != nil {
		return
	}
	m["php"] = struct{}{}

	var c struct {
		Require map[string]string `json:"require"`
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return
	}
	if _, ok := c.Require["laravel/framework"]; ok {
		m["laravel"] = struct{}{}
	}
}

func detectPackageJSON(dir string, m map[string]struct{}) {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return
	}
	m["node"] = struct{}{}

	var p struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &p); err != nil {
		return
	}
	hasDep := func(name string) bool {
		_, inDeps := p.Dependencies[name]
		_, inDevDeps := p.DevDependencies[name]
		return inDeps || inDevDeps
	}
	if hasDep("vite") {
		m["vite"] = struct{}{}
	}
	if hasDep("next") {
		m["nextjs"] = struct{}{}
	}
}

func detectPython(dir string, m map[string]struct{}) {
	// Read requirements.txt and pyproject.toml content so we can also
	// detect the framework (flask, django), not just the language. The
	// detection is substring-based and case-insensitive; good enough for
	// MVP without pulling in a TOML/requirements-spec parser.
	hasPython := false
	for _, name := range []string{"pyproject.toml", "requirements.txt"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		hasPython = true
		lower := bytes.ToLower(data)
		if bytes.Contains(lower, []byte("flask")) {
			m["flask"] = struct{}{}
		}
		if bytes.Contains(lower, []byte("django")) {
			m["django"] = struct{}{}
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "manage.py")); err == nil {
		hasPython = true
		m["django"] = struct{}{}
	}
	if hasPython {
		m["python"] = struct{}{}
	}
}

func detectDocker(dir string, m map[string]struct{}) {
	candidates := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
	}
	var data []byte
	for _, name := range candidates {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err == nil {
			data = b
			break
		}
	}
	if data == nil {
		return
	}
	m["docker"] = struct{}{}

	lower := bytes.ToLower(data)
	for _, svc := range []string{"mysql", "postgres", "redis"} {
		if bytes.Contains(lower, []byte(svc)) {
			m[svc] = struct{}{}
		}
	}
}
