package cmd

import (
	"flag"
	"fmt"
	"io"

	"github.com/tincke10/lane/internal/activator"
)

// parseShellFlag pulls --shell out of args and returns the chosen
// dialect along with the remaining positional arguments. The default
// is POSIX so existing bash/zsh users don't need to pass anything.
//
// fs writes its own parse errors to errOut so command help follows the
// dispatcher's stderr convention.
func parseShellFlag(name string, args []string, errOut io.Writer) (activator.Shell, []string, bool) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(errOut)
	shellStr := fs.String("shell", "posix", "output syntax: posix (bash/zsh) or fish")
	if err := fs.Parse(args); err != nil {
		return "", nil, false
	}
	switch *shellStr {
	case "posix":
		return activator.ShellPOSIX, fs.Args(), true
	case "fish":
		return activator.ShellFish, fs.Args(), true
	default:
		fmt.Fprintf(errOut, "lane: invalid --shell value %q (want posix or fish)\n", *shellStr)
		return "", nil, false
	}
}
