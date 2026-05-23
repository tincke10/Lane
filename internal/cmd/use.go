package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/tincke10/lane/internal/activator"
	"github.com/tincke10/lane/internal/ports"
	"github.com/tincke10/lane/internal/registry"
)

func cmdUse(args []string, stdout, stderr io.Writer) int {
	shell, rest, ok := parseShellFlag("use", args, stderr)
	if !ok {
		return ExitUsage
	}
	if len(rest) != 1 {
		fmt.Fprintln(stderr, "lane: usage: lane use [--shell posix|fish] <name>")
		return ExitUsage
	}
	name := rest[0]

	reg, err := registry.Load()
	if err != nil {
		fmt.Fprintf(stderr, "lane: load registry: %v\n", err)
		return ExitError
	}
	project, ok := reg.Get(name)
	if !ok {
		fmt.Fprintf(stderr, "lane: project %q not found\n", name)
		return ExitError
	}

	if collisions := ports.CheckCollisions(project.Ports); len(collisions) > 0 {
		fmt.Fprintf(stderr, "lane: port collision detected for %q — not activating:\n", name)
		for _, c := range collisions {
			fmt.Fprintf(stderr, "  %s=%d is in use\n", c.Name, c.Port)
		}
		return ExitError
	}

	fmt.Fprint(stdout, activator.Activate(registry.NamedProject{
		Name:    name,
		Project: project,
	}, shell))
	return ExitOK
}

func cmdUnuse(args []string, stdout, stderr io.Writer) int {
	shell, rest, ok := parseShellFlag("unuse", args, stderr)
	if !ok {
		return ExitUsage
	}
	if len(rest) != 0 {
		fmt.Fprintln(stderr, "lane: usage: lane unuse [--shell posix|fish]")
		return ExitUsage
	}

	active := os.Getenv(activator.ActiveProjectEnvVar)
	if active == "" {
		// No active project. Emit a no-op unset so eval is safe and the
		// shell ends up with LANE_ACTIVE_PROJECT cleared either way.
		fmt.Fprint(stdout, activator.UnsetActiveProject(shell))
		return ExitOK
	}

	reg, err := registry.Load()
	if err != nil {
		fmt.Fprintf(stderr, "lane: load registry: %v\n", err)
		return ExitError
	}
	project, ok := reg.Get(active)
	if !ok {
		fmt.Fprint(stdout, activator.UnsetActiveProject(shell))
		return ExitOK
	}

	fmt.Fprint(stdout, activator.Deactivate(registry.NamedProject{
		Name:    active,
		Project: project,
	}, shell))
	return ExitOK
}
