package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/tincke10/lane/internal/registry"
)

func cmdList(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "lane: usage: lane list")
		return ExitUsage
	}

	reg, err := registry.Load()
	if err != nil {
		fmt.Fprintf(stderr, "lane: load registry: %v\n", err)
		return ExitError
	}

	projects := reg.List()
	if len(projects) == 0 {
		fmt.Fprintln(stdout, "no projects registered. run `lane init` inside a project.")
		return ExitOK
	}

	for _, np := range projects {
		fmt.Fprintf(stdout, "%s\n", np.Name)
		fmt.Fprintf(stdout, "  path:  %s\n", np.Project.Path)
		if len(np.Project.Stack) > 0 {
			fmt.Fprintf(stdout, "  stack: %s\n", strings.Join(np.Project.Stack, ", "))
		}
		for _, env := range sortedEnvKeys(np.Project.Ports) {
			fmt.Fprintf(stdout, "  %s=%d\n", env, np.Project.Ports[env])
		}
	}
	return ExitOK
}

func cmdRm(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "lane: usage: lane rm <name>")
		return ExitUsage
	}
	name := args[0]

	reg, err := registry.Load()
	if err != nil {
		fmt.Fprintf(stderr, "lane: load registry: %v\n", err)
		return ExitError
	}
	if err := reg.Remove(name); err != nil {
		fmt.Fprintf(stderr, "lane: %v\n", err)
		return ExitError
	}
	if err := reg.Save(); err != nil {
		fmt.Fprintf(stderr, "lane: save: %v\n", err)
		return ExitError
	}
	fmt.Fprintf(stdout, "removed %q\n", name)
	return ExitOK
}
