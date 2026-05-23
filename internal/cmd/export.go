package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/tincke10/lane/internal/activator"
	"github.com/tincke10/lane/internal/registry"
)

// cmdExport is the hook-driven counterpart to `lane use`. It is intended
// to run on every shell prompt: it inspects cwd and LANE_ACTIVE_PROJECT,
// decides whether to activate, deactivate, switch, or stay, and emits
// only the minimal shell statements needed.
//
// The command is silent on benign failure (no readable registry, cwd
// unavailable). Breaking the user's prompt over a transient error would
// be worse than producing no output.
func cmdExport(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "lane: usage: lane export")
		return ExitUsage
	}

	cwd, err := os.Getwd()
	if err != nil {
		return ExitOK
	}

	reg, err := registry.Load()
	if err != nil {
		return ExitOK
	}

	current := os.Getenv(activator.ActiveProjectEnvVar)
	target := findProjectForPath(cwd, reg)

	if target == current {
		return ExitOK
	}

	if current != "" {
		writeDeactivate(stdout, reg, current)
	}
	if target != "" {
		writeActivate(stdout, reg, target)
	}
	return ExitOK
}

// findProjectForPath walks up from cwd looking for a directory that
// matches a registered project's Path. Returns the project name, or "".
// Nested projects naturally resolve to the deepest match because the
// walk starts at the leaf.
func findProjectForPath(cwd string, reg *registry.Registry) string {
	cwd = filepath.Clean(cwd)

	pathToName := make(map[string]string, len(reg.Projects))
	for name, p := range reg.Projects {
		pathToName[filepath.Clean(p.Path)] = name
	}

	for path := cwd; ; {
		if name, ok := pathToName[path]; ok {
			return name
		}
		parent := filepath.Dir(path)
		if parent == path {
			return ""
		}
		path = parent
	}
}

func writeActivate(w io.Writer, reg *registry.Registry, name string) {
	p, ok := reg.Get(name)
	if !ok {
		return
	}
	fmt.Fprint(w, activator.Activate(registry.NamedProject{Name: name, Project: p}))
}

func writeDeactivate(w io.Writer, reg *registry.Registry, name string) {
	p, ok := reg.Get(name)
	if !ok {
		// The recorded active project is no longer registered. Clear
		// the marker so the shell does not stay stuck on a ghost name.
		fmt.Fprintf(w, "unset %s\n", activator.ActiveProjectEnvVar)
		return
	}
	fmt.Fprint(w, activator.Deactivate(registry.NamedProject{Name: name, Project: p}))
}
