package activator_test

import (
	"strings"
	"testing"

	"github.com/tincke10/lane/internal/activator"
	"github.com/tincke10/lane/internal/registry"
)

func sample(name string, ports map[string]int) registry.NamedProject {
	return registry.NamedProject{
		Name: name,
		Project: registry.Project{
			Path:  "/home/dev/" + name,
			Ports: ports,
		},
	}
}

// -- POSIX --------------------------------------------------------------

func TestActivate_POSIX_EmptyPorts_EmitsOnlyActiveProject(t *testing.T) {
	got := activator.Activate(sample("proyecto-a", map[string]int{}), activator.ShellPOSIX)
	want := "export LANE_ACTIVE_PROJECT='proyecto-a'\n"
	if got != want {
		t.Errorf("Activate:\n got %q\nwant %q", got, want)
	}
}

func TestActivate_POSIX_MultiplePorts_SortedAlphabetically(t *testing.T) {
	got := activator.Activate(sample("proyecto-a", map[string]int{
		"VITE_PORT":       5174,
		"APP_PORT":        8081,
		"FORWARD_DB_PORT": 33061,
	}), activator.ShellPOSIX)

	want := strings.Join([]string{
		"export LANE_ACTIVE_PROJECT='proyecto-a'",
		"export APP_PORT=8081",
		"export FORWARD_DB_PORT=33061",
		"export VITE_PORT=5174",
		"",
	}, "\n")

	if got != want {
		t.Errorf("Activate:\n got %q\nwant %q", got, want)
	}
}

func TestActivate_POSIX_ProjectNameWithSingleQuote_Escapes(t *testing.T) {
	got := activator.Activate(sample("foo's bar", map[string]int{}), activator.ShellPOSIX)
	want := `export LANE_ACTIVE_PROJECT='foo'\''s bar'` + "\n"
	if got != want {
		t.Errorf("Activate:\n got %q\nwant %q", got, want)
	}
}

func TestDeactivate_POSIX_EmitsUnsetsInSortedOrder(t *testing.T) {
	got := activator.Deactivate(sample("p", map[string]int{
		"VITE_PORT": 5174,
		"APP_PORT":  8081,
	}), activator.ShellPOSIX)

	want := strings.Join([]string{
		"unset LANE_ACTIVE_PROJECT",
		"unset APP_PORT",
		"unset VITE_PORT",
		"",
	}, "\n")

	if got != want {
		t.Errorf("Deactivate:\n got %q\nwant %q", got, want)
	}
}

// -- Fish ---------------------------------------------------------------

func TestActivate_Fish_EmptyPorts_EmitsOnlyActiveProject(t *testing.T) {
	got := activator.Activate(sample("proyecto-a", map[string]int{}), activator.ShellFish)
	want := "set -gx LANE_ACTIVE_PROJECT 'proyecto-a'\n"
	if got != want {
		t.Errorf("Activate fish:\n got %q\nwant %q", got, want)
	}
}

func TestActivate_Fish_MultiplePorts_SortedAlphabetically(t *testing.T) {
	got := activator.Activate(sample("proyecto-a", map[string]int{
		"VITE_PORT": 5174,
		"APP_PORT":  8081,
	}), activator.ShellFish)

	want := strings.Join([]string{
		"set -gx LANE_ACTIVE_PROJECT 'proyecto-a'",
		"set -gx APP_PORT 8081",
		"set -gx VITE_PORT 5174",
		"",
	}, "\n")

	if got != want {
		t.Errorf("Activate fish:\n got %q\nwant %q", got, want)
	}
}

func TestActivate_Fish_ProjectNameWithSingleQuote_UsesBackslashEscape(t *testing.T) {
	got := activator.Activate(sample("foo's bar", map[string]int{}), activator.ShellFish)
	want := `set -gx LANE_ACTIVE_PROJECT 'foo\'s bar'` + "\n"
	if got != want {
		t.Errorf("Activate fish:\n got %q\nwant %q", got, want)
	}
}

func TestDeactivate_Fish_EmitsSetEraseInSortedOrder(t *testing.T) {
	got := activator.Deactivate(sample("p", map[string]int{
		"VITE_PORT": 5174,
		"APP_PORT":  8081,
	}), activator.ShellFish)

	want := strings.Join([]string{
		"set -e LANE_ACTIVE_PROJECT",
		"set -e APP_PORT",
		"set -e VITE_PORT",
		"",
	}, "\n")

	if got != want {
		t.Errorf("Deactivate fish:\n got %q\nwant %q", got, want)
	}
}

// -- Shared -------------------------------------------------------------

func TestActivate_IsDeterministic(t *testing.T) {
	np := sample("p", map[string]int{
		"APP_PORT":  8081,
		"VITE_PORT": 5174,
	})
	for _, sh := range []activator.Shell{activator.ShellPOSIX, activator.ShellFish} {
		first := activator.Activate(np, sh)
		for i := 0; i < 25; i++ {
			if got := activator.Activate(np, sh); got != first {
				t.Fatalf("Activate(%s) not deterministic on call %d", sh, i)
			}
		}
	}
}

func TestActiveProjectEnvVarConstant(t *testing.T) {
	if activator.ActiveProjectEnvVar != "LANE_ACTIVE_PROJECT" {
		t.Errorf("ActiveProjectEnvVar = %q, want LANE_ACTIVE_PROJECT", activator.ActiveProjectEnvVar)
	}
}

func TestUnsetActiveProject_POSIX(t *testing.T) {
	if got := activator.UnsetActiveProject(activator.ShellPOSIX); got != "unset LANE_ACTIVE_PROJECT\n" {
		t.Errorf("got %q", got)
	}
}

func TestUnsetActiveProject_Fish(t *testing.T) {
	if got := activator.UnsetActiveProject(activator.ShellFish); got != "set -e LANE_ACTIVE_PROJECT\n" {
		t.Errorf("got %q", got)
	}
}

func TestActivate_TrailingNewline(t *testing.T) {
	for _, sh := range []activator.Shell{activator.ShellPOSIX, activator.ShellFish} {
		out := activator.Activate(sample("p", map[string]int{"APP_PORT": 8080}), sh)
		if !strings.HasSuffix(out, "\n") {
			t.Errorf("Activate(%s) missing trailing newline: %q", sh, out)
		}
	}
}
