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

func TestActivate_EmptyPorts_EmitsOnlyActiveProject(t *testing.T) {
	got := activator.Activate(sample("proyecto-a", map[string]int{}))
	want := "export LANE_ACTIVE_PROJECT='proyecto-a'\n"
	if got != want {
		t.Errorf("Activate:\n got %q\nwant %q", got, want)
	}
}

func TestActivate_MultiplePorts_SortedAlphabetically(t *testing.T) {
	got := activator.Activate(sample("proyecto-a", map[string]int{
		"VITE_PORT":       5174,
		"APP_PORT":        8081,
		"FORWARD_DB_PORT": 33061,
	}))

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

func TestActivate_ProjectNameWithSingleQuote_Escapes(t *testing.T) {
	got := activator.Activate(sample("foo's bar", map[string]int{}))
	// POSIX escape: end quote, escaped quote, start quote → '\''
	want := `export LANE_ACTIVE_PROJECT='foo'\''s bar'` + "\n"
	if got != want {
		t.Errorf("Activate:\n got %q\nwant %q", got, want)
	}
}

func TestActivate_IsDeterministic(t *testing.T) {
	np := sample("proyecto-a", map[string]int{
		"APP_PORT":  8081,
		"VITE_PORT": 5174,
	})
	first := activator.Activate(np)
	for i := 0; i < 50; i++ {
		if got := activator.Activate(np); got != first {
			t.Fatalf("Activate not deterministic on call %d:\n got %q\nfirst %q", i, got, first)
		}
	}
}

func TestActivate_ExposesActiveProjectEnvVarConstant(t *testing.T) {
	if activator.ActiveProjectEnvVar != "LANE_ACTIVE_PROJECT" {
		t.Errorf("ActiveProjectEnvVar = %q, want %q", activator.ActiveProjectEnvVar, "LANE_ACTIVE_PROJECT")
	}
}

func TestDeactivate_EmitsUnsetsInSortedOrder(t *testing.T) {
	got := activator.Deactivate(sample("proyecto-a", map[string]int{
		"VITE_PORT": 5174,
		"APP_PORT":  8081,
	}))

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

func TestDeactivate_EmptyPorts_StillUnsetsActiveProject(t *testing.T) {
	got := activator.Deactivate(sample("anything", map[string]int{}))
	want := "unset LANE_ACTIVE_PROJECT\n"
	if got != want {
		t.Errorf("Deactivate:\n got %q\nwant %q", got, want)
	}
}

func TestActivate_TrailingNewline(t *testing.T) {
	// Important: shell `eval` is tolerant either way, but a trailing newline
	// keeps output composable with other shell-emitting tools.
	out := activator.Activate(sample("p", map[string]int{"APP_PORT": 8080}))
	if !strings.HasSuffix(out, "\n") {
		t.Errorf("Activate output should end with newline, got %q", out)
	}
}
