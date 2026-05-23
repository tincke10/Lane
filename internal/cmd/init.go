package cmd

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/tincke10/lane/internal/convention"
	"github.com/tincke10/lane/internal/ports"
	"github.com/tincke10/lane/internal/registry"
	"github.com/tincke10/lane/internal/stack"
)

func cmdInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	nameFlag := fs.String("name", "", "project name (default: basename of path)")
	pathFlag := fs.String("path", "", "project path (default: cwd)")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}

	path, err := resolvePath(*pathFlag)
	if err != nil {
		fmt.Fprintf(stderr, "lane: %v\n", err)
		return ExitError
	}

	name := *nameFlag
	if name == "" {
		name = filepath.Base(path)
	}

	reg, err := registry.Load()
	if err != nil {
		fmt.Fprintf(stderr, "lane: load registry: %v\n", err)
		return ExitError
	}
	if _, exists := reg.Get(name); exists {
		fmt.Fprintf(stderr, "lane: project %q already registered\n", name)
		return ExitError
	}

	detected, err := stack.Detect(path)
	if err != nil {
		fmt.Fprintf(stderr, "lane: detect stack: %v\n", err)
		return ExitError
	}

	reserved := allReservedPorts(reg)
	bases := portBasesFor(detected)

	allocated := make(map[string]int, len(bases))
	for _, env := range sortedEnvKeys(bases) {
		port, err := ports.Allocate(bases[env], reserved)
		if err != nil {
			fmt.Fprintf(stderr, "lane: allocate %s: %v\n", env, err)
			return ExitError
		}
		allocated[env] = port
		reserved[port] = struct{}{}
	}

	project := registry.Project{
		Path:    path,
		Stack:   detected,
		Ports:   allocated,
		Created: time.Now().UTC(),
	}
	if err := reg.Add(name, project); err != nil {
		fmt.Fprintf(stderr, "lane: add: %v\n", err)
		return ExitError
	}
	if err := reg.Save(); err != nil {
		fmt.Fprintf(stderr, "lane: save: %v\n", err)
		return ExitError
	}

	fmt.Fprintf(stdout, "registered %q at %s\n", name, path)
	if len(detected) > 0 {
		fmt.Fprintf(stdout, "  stack: %v\n", detected)
	}
	for _, env := range sortedEnvKeys(allocated) {
		fmt.Fprintf(stdout, "  %s=%d\n", env, allocated[env])
	}

	// A bad compose file should never block init — we silently skip the
	// convention check on read error and report missing references as a
	// non-fatal warning.
	if missing, err := convention.Validate(path, sortedEnvKeys(allocated)); err == nil && len(missing) > 0 {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "  warning: docker-compose.yml does not reference these env vars:")
		for _, v := range missing {
			fmt.Fprintf(stdout, "    - %s (allocated %d, but compose will ignore it)\n", v, allocated[v])
		}
		fmt.Fprintln(stdout, "  use the ${VAR:-default}:default pattern in your compose ports to opt in.")
	}
	return ExitOK
}

func resolvePath(p string) (string, error) {
	if p == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("getwd: %w", err)
		}
		p = cwd
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	return abs, nil
}
