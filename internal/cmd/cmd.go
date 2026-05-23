// Package cmd wires the Lane CLI: a thin dispatcher over the internal
// packages (registry, ports, stack, activator). All subcommand handlers
// receive stdout/stderr writers so they can be tested without poking
// global state.
package cmd

import (
	"fmt"
	"io"
	"sort"

	"github.com/tincke10/lane/internal/registry"
)

// Exit codes follow common UNIX conventions.
const (
	ExitOK    = 0
	ExitError = 1
	ExitUsage = 2
)

const usage = `lane — port-collision-free project switcher

Usage:
  lane init [--name NAME] [--path PATH]   register the project in PATH (default: cwd)
  lane use [--shell posix|fish] <name>     print shell exports for <name>
  lane unuse [--shell posix|fish]          print shell unsets for the active project
  lane list                                list registered projects
  lane rm <name>                           remove a project from the registry
  lane doctor                              diagnose registry health and port conflicts
  lane hook <bash|zsh|fish>                print shell hook for auto-activation on cd
  lane export [--shell posix|fish]         hook-driven activation diff (called from prompt)
  lane serve [extras...]                   run 'php artisan serve' on Lane's $APP_PORT
  lane vite  [extras...]                   run 'npx vite' on Lane's $VITE_PORT
  lane help                                show this help

Activation pattern (manual, bash/zsh):
  eval "$(lane use my-project)"
  eval "$(lane unuse)"

Activation pattern (manual, fish):
  lane use --shell fish my-project | source
  lane unuse --shell fish | source

Auto-activation (add to your rc file once):
  eval "$(lane hook zsh)"            # zsh
  eval "$(lane hook bash)"           # bash
  lane hook fish | source             # fish (in ~/.config/fish/config.fish)
`

// Run dispatches a Lane subcommand. args is the program args WITHOUT the
// executable name (i.e. os.Args[1:]). Returns the process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return ExitUsage
	}

	switch args[0] {
	case "init":
		return cmdInit(args[1:], stdout, stderr)
	case "use":
		return cmdUse(args[1:], stdout, stderr)
	case "unuse":
		return cmdUnuse(args[1:], stdout, stderr)
	case "list", "ls":
		return cmdList(args[1:], stdout, stderr)
	case "rm", "remove":
		return cmdRm(args[1:], stdout, stderr)
	case "doctor":
		return cmdDoctor(args[1:], stdout, stderr)
	case "hook":
		return cmdHook(args[1:], stdout, stderr)
	case "export":
		return cmdExport(args[1:], stdout, stderr)
	case "serve":
		return cmdServe(args[1:], stdout, stderr)
	case "vite":
		return cmdVite(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return ExitOK
	default:
		fmt.Fprintf(stderr, "lane: unknown command %q\n\n%s", args[0], usage)
		return ExitUsage
	}
}

// allReservedPorts returns every port currently claimed across the registry,
// so init can allocate without overlapping existing projects.
func allReservedPorts(r *registry.Registry) map[int]struct{} {
	out := make(map[int]struct{})
	for _, p := range r.Projects {
		for _, port := range p.Ports {
			out[port] = struct{}{}
		}
	}
	return out
}

// portBasesFor maps detected stack markers to the env vars Lane allocates,
// each paired with its preferred starting port. The same env var produced
// by multiple markers is allowed; later wins, but ordering is deterministic
// because the caller iterates sorted stack markers.
func portBasesFor(stackMarkers []string) map[string]int {
	out := make(map[string]int)
	for _, m := range stackMarkers {
		switch m {
		case "laravel":
			out["APP_PORT"] = 8080
		case "vite":
			out["VITE_PORT"] = 5173
		case "mysql":
			out["FORWARD_DB_PORT"] = 33060
		case "postgres":
			out["FORWARD_DB_PORT"] = 54320
		case "redis":
			out["FORWARD_REDIS_PORT"] = 63790
		}
	}
	return out
}

// sortedEnvKeys returns the keys of m sorted alphabetically so allocation
// order is reproducible across runs.
func sortedEnvKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
