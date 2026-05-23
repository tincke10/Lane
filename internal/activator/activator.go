// Package activator emits shell statements that, when evaluated by the
// user's shell, set or unset the env vars associated with a Lane
// project. The pattern follows direnv / asdf: Lane prints, the shell
// evals (or `| source`s in the case of fish).
package activator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tincke10/lane/internal/registry"
)

// Shell selects the output syntax dialect.
type Shell string

const (
	// ShellPOSIX targets bash and zsh: `export NAME=value` / `unset NAME`.
	ShellPOSIX Shell = "posix"
	// ShellFish targets fish: `set -gx NAME value` / `set -e NAME`.
	ShellFish Shell = "fish"
)

// ActiveProjectEnvVar is the env var Lane sets to mark which project is
// currently active in the shell. Read by `lane unuse`, `lane doctor`,
// and the auto-activation hook.
const ActiveProjectEnvVar = "LANE_ACTIVE_PROJECT"

// Activate returns shell statements that export the project's name (as
// ActiveProjectEnvVar) and its reserved port env vars in the requested
// shell dialect. Port entries are emitted in alphabetical order by key
// for deterministic output.
func Activate(np registry.NamedProject, sh Shell) string {
	var b strings.Builder
	writeExport(&b, sh, ActiveProjectEnvVar, quoteString(sh, np.Name))

	for _, k := range sortedKeys(np.Project.Ports) {
		writeExport(&b, sh, k, fmt.Sprintf("%d", np.Project.Ports[k]))
	}
	return b.String()
}

// Deactivate returns shell statements that unset ActiveProjectEnvVar
// and every port env var associated with the project, in the same
// alphabetical order Activate uses.
func Deactivate(np registry.NamedProject, sh Shell) string {
	var b strings.Builder
	writeUnset(&b, sh, ActiveProjectEnvVar)

	for _, k := range sortedKeys(np.Project.Ports) {
		writeUnset(&b, sh, k)
	}
	return b.String()
}

// UnsetActiveProject returns a single shell statement that erases the
// ActiveProjectEnvVar marker. Used by the hook to clear an orphaned
// active project when its registry entry no longer exists.
func UnsetActiveProject(sh Shell) string {
	var b strings.Builder
	writeUnset(&b, sh, ActiveProjectEnvVar)
	return b.String()
}

func writeExport(b *strings.Builder, sh Shell, name, valueLiteral string) {
	switch sh {
	case ShellFish:
		fmt.Fprintf(b, "set -gx %s %s\n", name, valueLiteral)
	default:
		fmt.Fprintf(b, "export %s=%s\n", name, valueLiteral)
	}
}

func writeUnset(b *strings.Builder, sh Shell, name string) {
	switch sh {
	case ShellFish:
		fmt.Fprintf(b, "set -e %s\n", name)
	default:
		fmt.Fprintf(b, "unset %s\n", name)
	}
}

// quoteString wraps s in single quotes using the dialect's escape rules.
//
//	POSIX: end-quote, escaped-quote, reopen — 'foo'\''s bar'
//	Fish:  backslash-escape inside single quotes — 'foo\'s bar'
func quoteString(sh Shell, s string) string {
	switch sh {
	case ShellFish:
		return "'" + strings.ReplaceAll(s, "'", `\'`) + "'"
	default:
		return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
	}
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
