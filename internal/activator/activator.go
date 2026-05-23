// Package activator emits POSIX shell statements that, when evaluated by
// the user's shell, set or unset the env vars associated with a Lane
// project. The pattern follows direnv / asdf: Lane prints, the shell evals.
package activator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tincke10/lane/internal/registry"
)

// ActiveProjectEnvVar is the env var Lane sets to mark which project is
// currently active in the shell. Read by status / doctor commands.
const ActiveProjectEnvVar = "LANE_ACTIVE_PROJECT"

// Activate returns POSIX shell statements that export the project's name
// (as ActiveProjectEnvVar) and its reserved port env vars. Port entries
// are emitted in alphabetical order by key for deterministic output.
func Activate(np registry.NamedProject) string {
	var b strings.Builder
	fmt.Fprintf(&b, "export %s=%s\n", ActiveProjectEnvVar, posixQuote(np.Name))

	for _, k := range sortedKeys(np.Project.Ports) {
		fmt.Fprintf(&b, "export %s=%d\n", k, np.Project.Ports[k])
	}
	return b.String()
}

// Deactivate returns POSIX shell statements that unset ActiveProjectEnvVar
// and every port env var associated with the project, in the same
// alphabetical order Activate uses.
func Deactivate(np registry.NamedProject) string {
	var b strings.Builder
	fmt.Fprintf(&b, "unset %s\n", ActiveProjectEnvVar)

	for _, k := range sortedKeys(np.Project.Ports) {
		fmt.Fprintf(&b, "unset %s\n", k)
	}
	return b.String()
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// posixQuote wraps s in single quotes, escaping embedded single quotes
// using the canonical POSIX technique: close, escaped-quote, reopen.
func posixQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
