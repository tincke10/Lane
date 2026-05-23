package cmd_test

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tincke10/lane/internal/cmd"
)

// runner returns a closure that executes a Lane subcommand against an
// isolated XDG_CONFIG_HOME, so tests don't touch the user's real registry.
func runner(t *testing.T) func(args ...string) (stdout, stderr string, code int) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	return func(args ...string) (string, string, int) {
		var out, errb bytes.Buffer
		code := cmd.Run(args, &out, &errb)
		return out.String(), errb.String(), code
	}
}

func laravelProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	composer := `{"require":{"laravel/framework":"^11.0"}}`
	if err := os.WriteFile(filepath.Join(dir, "composer.json"), []byte(composer), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// -- dispatcher ---------------------------------------------------------

func TestRun_NoArgs_ReturnsUsage(t *testing.T) {
	run := runner(t)
	_, stderr, code := run()
	if code != cmd.ExitUsage {
		t.Errorf("code = %d, want %d", code, cmd.ExitUsage)
	}
	if !strings.Contains(stderr, "Usage") {
		t.Errorf("stderr missing usage: %s", stderr)
	}
}

func TestRun_Help_ReturnsZero(t *testing.T) {
	run := runner(t)
	stdout, _, code := run("help")
	if code != cmd.ExitOK {
		t.Errorf("code = %d, want %d", code, cmd.ExitOK)
	}
	if !strings.Contains(stdout, "Usage") {
		t.Errorf("stdout missing usage: %s", stdout)
	}
}

func TestRun_UnknownCommand_ReturnsUsage(t *testing.T) {
	run := runner(t)
	_, stderr, code := run("blarg")
	if code != cmd.ExitUsage {
		t.Errorf("code = %d, want %d", code, cmd.ExitUsage)
	}
	if !strings.Contains(stderr, "unknown") {
		t.Errorf("stderr missing 'unknown': %s", stderr)
	}
}

// -- init ---------------------------------------------------------------

func TestInit_LaravelProject_RegistersWithStackAndPorts(t *testing.T) {
	run := runner(t)
	proj := laravelProject(t)

	stdout, stderr, code := run("init", "--path", proj, "--name", "myapp")
	if code != cmd.ExitOK {
		t.Fatalf("code=%d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "myapp") {
		t.Errorf("stdout missing myapp: %s", stdout)
	}
	if !strings.Contains(stdout, "APP_PORT=") {
		t.Errorf("stdout missing APP_PORT: %s", stdout)
	}
	if !strings.Contains(stdout, "stack:") {
		t.Errorf("stdout missing stack: %s", stdout)
	}
}

func TestInit_DuplicateName_ReturnsError(t *testing.T) {
	run := runner(t)
	proj := laravelProject(t)

	run("init", "--path", proj, "--name", "myapp")
	_, stderr, code := run("init", "--path", proj, "--name", "myapp")
	if code != cmd.ExitError {
		t.Errorf("code = %d, want %d", code, cmd.ExitError)
	}
	if !strings.Contains(stderr, "already registered") {
		t.Errorf("stderr missing 'already registered': %s", stderr)
	}
}

func TestInit_DefaultNameFromBasename(t *testing.T) {
	run := runner(t)
	// Create a project under a known basename
	dir := t.TempDir()
	projDir := filepath.Join(dir, "the-project")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "composer.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, code := run("init", "--path", projDir)
	if code != cmd.ExitOK {
		t.Fatalf("code=%d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "the-project") {
		t.Errorf("expected default name 'the-project' in output: %s", stdout)
	}
}

func TestInit_NonexistentPath_ReturnsError(t *testing.T) {
	run := runner(t)
	_, stderr, code := run("init", "--path", "/no/such/path/lane-fake", "--name", "x")
	if code != cmd.ExitError {
		t.Errorf("code = %d, want %d", code, cmd.ExitError)
	}
	if stderr == "" {
		t.Error("expected stderr message")
	}
}

// -- list / rm ----------------------------------------------------------

func TestList_EmptyRegistry_PrintsHint(t *testing.T) {
	run := runner(t)
	stdout, _, code := run("list")
	if code != cmd.ExitOK {
		t.Errorf("code = %d, want %d", code, cmd.ExitOK)
	}
	if !strings.Contains(stdout, "no projects") {
		t.Errorf("stdout = %s", stdout)
	}
}

func TestList_AfterInit_ShowsProject(t *testing.T) {
	run := runner(t)
	proj := laravelProject(t)
	run("init", "--path", proj, "--name", "myapp")

	stdout, _, code := run("list")
	if code != cmd.ExitOK {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(stdout, "myapp") {
		t.Errorf("stdout missing myapp: %s", stdout)
	}
	if !strings.Contains(stdout, "APP_PORT") {
		t.Errorf("stdout missing APP_PORT: %s", stdout)
	}
	if !strings.Contains(stdout, "laravel") {
		t.Errorf("stdout missing stack 'laravel': %s", stdout)
	}
}

func TestRm_ExistingProject_Removes(t *testing.T) {
	run := runner(t)
	proj := laravelProject(t)
	run("init", "--path", proj, "--name", "myapp")

	_, _, code := run("rm", "myapp")
	if code != cmd.ExitOK {
		t.Fatalf("code=%d", code)
	}

	listOut, _, _ := run("list")
	if !strings.Contains(listOut, "no projects") {
		t.Errorf("after rm, list should be empty: %s", listOut)
	}
}

func TestRm_Missing_ReturnsError(t *testing.T) {
	run := runner(t)
	_, stderr, code := run("rm", "ghost")
	if code != cmd.ExitError {
		t.Errorf("code=%d, want %d", code, cmd.ExitError)
	}
	if !strings.Contains(stderr, "not found") {
		t.Errorf("stderr missing 'not found': %s", stderr)
	}
}

// -- use / unuse --------------------------------------------------------

func TestUse_RegisteredProject_PrintsExports(t *testing.T) {
	run := runner(t)
	proj := laravelProject(t)
	run("init", "--path", proj, "--name", "myapp")

	stdout, _, code := run("use", "myapp")
	if code != cmd.ExitOK {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(stdout, "export LANE_ACTIVE_PROJECT='myapp'") {
		t.Errorf("stdout missing LANE_ACTIVE_PROJECT: %s", stdout)
	}
	if !strings.Contains(stdout, "export APP_PORT=") {
		t.Errorf("stdout missing APP_PORT: %s", stdout)
	}
}

func TestUse_UnknownProject_ReturnsError(t *testing.T) {
	run := runner(t)
	stdout, stderr, code := run("use", "ghost")
	if code != cmd.ExitError {
		t.Errorf("code = %d, want %d", code, cmd.ExitError)
	}
	if stdout != "" {
		t.Errorf("stdout should be empty on error (eval safety), got: %s", stdout)
	}
	if !strings.Contains(stderr, "not found") {
		t.Errorf("stderr missing 'not found': %s", stderr)
	}
}

func TestUse_Collision_EmptyStdoutAndErrorExit(t *testing.T) {
	run := runner(t)
	proj := laravelProject(t)
	run("init", "--path", proj, "--name", "myapp")

	listOut, _, _ := run("list")
	port := parsePort(t, listOut, "APP_PORT")

	l := mustBind(t, port)
	defer l.Close()

	stdout, stderr, code := run("use", "myapp")
	if code != cmd.ExitError {
		t.Errorf("code = %d, want %d", code, cmd.ExitError)
	}
	if stdout != "" {
		t.Errorf("stdout MUST be empty on collision (eval safety), got: %q", stdout)
	}
	if !strings.Contains(stderr, "collision") {
		t.Errorf("stderr missing 'collision': %s", stderr)
	}
}

func TestUnuse_WithActiveProject_EmitsUnsets(t *testing.T) {
	run := runner(t)
	proj := laravelProject(t)
	run("init", "--path", proj, "--name", "myapp")

	t.Setenv("LANE_ACTIVE_PROJECT", "myapp")

	stdout, _, code := run("unuse")
	if code != cmd.ExitOK {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(stdout, "unset LANE_ACTIVE_PROJECT") {
		t.Errorf("stdout missing unset LANE_ACTIVE_PROJECT: %s", stdout)
	}
	if !strings.Contains(stdout, "unset APP_PORT") {
		t.Errorf("stdout missing unset APP_PORT: %s", stdout)
	}
}

func TestUnuse_NoActiveProject_EmitsOnlyMarkerUnset(t *testing.T) {
	run := runner(t)
	t.Setenv("LANE_ACTIVE_PROJECT", "")

	stdout, _, code := run("unuse")
	if code != cmd.ExitOK {
		t.Fatalf("code=%d", code)
	}
	want := "unset LANE_ACTIVE_PROJECT\n"
	if stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}
}

func TestUnuse_ActiveProjectMissingFromRegistry_StillClears(t *testing.T) {
	run := runner(t)
	t.Setenv("LANE_ACTIVE_PROJECT", "ghost-project")

	stdout, _, code := run("unuse")
	if code != cmd.ExitOK {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(stdout, "unset LANE_ACTIVE_PROJECT") {
		t.Errorf("stdout missing marker unset: %s", stdout)
	}
}

// -- helpers ------------------------------------------------------------

func parsePort(t *testing.T, output, envVar string) int {
	t.Helper()
	needle := envVar + "="
	idx := strings.Index(output, needle)
	if idx < 0 {
		t.Fatalf("env var %s not in output: %s", envVar, output)
	}
	rest := output[idx+len(needle):]
	end := strings.IndexAny(rest, "\n ")
	if end < 0 {
		end = len(rest)
	}
	var n int
	if _, err := fmt.Sscanf(rest[:end], "%d", &n); err != nil {
		t.Fatalf("parse port from %q: %v", rest[:end], err)
	}
	return n
}

func mustBind(t *testing.T, port int) net.Listener {
	t.Helper()
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("bind port %d: %v", port, err)
	}
	return l
}
