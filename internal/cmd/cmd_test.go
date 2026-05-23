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

func TestInit_ComposeWithoutVarReferences_PrintsWarning(t *testing.T) {
	run := runner(t)
	dir := t.TempDir()
	// Compose mentions Lane-relevant services (so init allocates ports for them)
	// but hardcodes the ports — Lane's exports would be ignored.
	composeBody := `services:
  app:
    image: nginx
    ports:
      - "80:80"
  db:
    image: mysql:8
    ports:
      - "3306:3306"
`
	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(composeBody), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "composer.json"), []byte(`{"require":{"laravel/framework":"^11"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, _, code := run("init", "--path", dir, "--name", "warnme")
	if code != cmd.ExitOK {
		t.Fatalf("code=%d stdout=%s", code, stdout)
	}
	if !strings.Contains(stdout, "warning") {
		t.Errorf("stdout missing convention warning: %s", stdout)
	}
	if !strings.Contains(stdout, "APP_PORT") {
		t.Errorf("warning should name APP_PORT: %s", stdout)
	}
	if !strings.Contains(stdout, "FORWARD_DB_PORT") {
		t.Errorf("warning should name FORWARD_DB_PORT: %s", stdout)
	}
}

func TestInit_ComposeWithVarReferences_NoWarning(t *testing.T) {
	run := runner(t)
	dir := t.TempDir()
	composeBody := `services:
  app:
    image: nginx
    ports:
      - "${APP_PORT:-80}:80"
  db:
    image: mysql:8
    ports:
      - "${FORWARD_DB_PORT:-3306}:3306"
`
	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(composeBody), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "composer.json"), []byte(`{"require":{"laravel/framework":"^11"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, _, code := run("init", "--path", dir, "--name", "clean")
	if code != cmd.ExitOK {
		t.Fatalf("code=%d", code)
	}
	if strings.Contains(stdout, "warning") {
		t.Errorf("expected no warning when compose references vars: %s", stdout)
	}
}

func TestInit_NoCompose_NoWarning(t *testing.T) {
	run := runner(t)
	proj := laravelProject(t) // no docker-compose

	stdout, _, code := run("init", "--path", proj, "--name", "nocompose")
	if code != cmd.ExitOK {
		t.Fatalf("code=%d", code)
	}
	if strings.Contains(stdout, "warning") {
		t.Errorf("expected no warning when project has no compose: %s", stdout)
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

// -- doctor -------------------------------------------------------------

func TestDoctor_EmptyRegistry_ExitsOK(t *testing.T) {
	run := runner(t)
	stdout, _, code := run("doctor")
	if code != cmd.ExitOK {
		t.Errorf("code = %d, want %d", code, cmd.ExitOK)
	}
	if !strings.Contains(stdout, "no projects") {
		t.Errorf("stdout = %s", stdout)
	}
}

func TestDoctor_HealthyProject_ExitsOK(t *testing.T) {
	run := runner(t)
	proj := laravelProject(t)
	run("init", "--path", proj, "--name", "myapp")

	stdout, _, code := run("doctor")
	if code != cmd.ExitOK {
		t.Errorf("code = %d, want %d — stdout=%s", code, cmd.ExitOK, stdout)
	}
	if !strings.Contains(stdout, "[ok]") {
		t.Errorf("stdout missing '[ok]' tag: %s", stdout)
	}
	if !strings.Contains(stdout, "myapp") {
		t.Errorf("stdout missing project name: %s", stdout)
	}
	if !strings.Contains(stdout, "1 ok") {
		t.Errorf("stdout missing summary '1 ok': %s", stdout)
	}
}

func TestDoctor_MissingPath_ExitsNonZero(t *testing.T) {
	run := runner(t)
	proj := laravelProject(t)
	run("init", "--path", proj, "--name", "myapp")

	// Remove the project directory after registration.
	if err := os.RemoveAll(proj); err != nil {
		t.Fatal(err)
	}

	stdout, _, code := run("doctor")
	if code != cmd.ExitError {
		t.Errorf("code = %d, want %d — stdout=%s", code, cmd.ExitError, stdout)
	}
	if !strings.Contains(stdout, "[error]") {
		t.Errorf("stdout missing '[error]' tag: %s", stdout)
	}
	if !strings.Contains(stdout, "path does not exist") {
		t.Errorf("stdout missing 'path does not exist': %s", stdout)
	}
}

func TestDoctor_BoundPort_WarnsButExitsOK(t *testing.T) {
	run := runner(t)
	proj := laravelProject(t)
	run("init", "--path", proj, "--name", "myapp")

	listOut, _, _ := run("list")
	port := parsePort(t, listOut, "APP_PORT")
	l := mustBind(t, port)
	defer l.Close()

	stdout, _, code := run("doctor")
	if code != cmd.ExitOK {
		t.Errorf("code = %d, want %d (warns should not fail) — stdout=%s", code, cmd.ExitOK, stdout)
	}
	if !strings.Contains(stdout, "in use") {
		t.Errorf("stdout missing 'in use' warning: %s", stdout)
	}
}

func TestDoctor_OrphanedActiveProject_Warns(t *testing.T) {
	run := runner(t)
	t.Setenv("LANE_ACTIVE_PROJECT", "ghost-app")

	stdout, _, code := run("doctor")
	if code != cmd.ExitOK {
		t.Errorf("code = %d, want %d — stdout=%s", code, cmd.ExitOK, stdout)
	}
	if !strings.Contains(stdout, "LANE_ACTIVE_PROJECT") {
		t.Errorf("stdout missing global LANE_ACTIVE_PROJECT warning: %s", stdout)
	}
}

func TestDoctor_BadArgs_ReturnsUsage(t *testing.T) {
	run := runner(t)
	_, stderr, code := run("doctor", "extra")
	if code != cmd.ExitUsage {
		t.Errorf("code = %d, want %d", code, cmd.ExitUsage)
	}
	if !strings.Contains(stderr, "usage") {
		t.Errorf("stderr missing usage: %s", stderr)
	}
}

// -- hook ---------------------------------------------------------------

func TestHook_Bash_EmitsPromptCommandInstallation(t *testing.T) {
	run := runner(t)
	stdout, _, code := run("hook", "bash")
	if code != cmd.ExitOK {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(stdout, "PROMPT_COMMAND") {
		t.Errorf("bash hook missing PROMPT_COMMAND wiring: %s", stdout)
	}
	if !strings.Contains(stdout, "lane export") {
		t.Errorf("bash hook missing 'lane export' call: %s", stdout)
	}
}

func TestHook_Zsh_EmitsPrecmdInstallation(t *testing.T) {
	run := runner(t)
	stdout, _, code := run("hook", "zsh")
	if code != cmd.ExitOK {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(stdout, "precmd_functions") {
		t.Errorf("zsh hook missing precmd_functions wiring: %s", stdout)
	}
	if !strings.Contains(stdout, "lane export") {
		t.Errorf("zsh hook missing 'lane export' call: %s", stdout)
	}
}

func TestHook_UnsupportedShell_ReturnsUsage(t *testing.T) {
	run := runner(t)
	_, stderr, code := run("hook", "tcsh")
	if code != cmd.ExitUsage {
		t.Errorf("code=%d, want %d", code, cmd.ExitUsage)
	}
	if !strings.Contains(stderr, "unsupported") {
		t.Errorf("stderr missing 'unsupported': %s", stderr)
	}
}

func TestHook_NoArgs_ReturnsUsage(t *testing.T) {
	run := runner(t)
	_, stderr, code := run("hook")
	if code != cmd.ExitUsage {
		t.Errorf("code=%d, want %d", code, cmd.ExitUsage)
	}
	if !strings.Contains(stderr, "usage") {
		t.Errorf("stderr missing 'usage': %s", stderr)
	}
}

// -- export -------------------------------------------------------------

func TestExport_InsideProject_EmitsActivate(t *testing.T) {
	run := runner(t)
	proj := laravelProject(t)
	run("init", "--path", proj, "--name", "myapp")

	t.Chdir(proj)

	stdout, _, code := run("export")
	if code != cmd.ExitOK {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(stdout, "export LANE_ACTIVE_PROJECT='myapp'") {
		t.Errorf("export missing activate: %s", stdout)
	}
}

func TestExport_InSubdirOfProject_EmitsActivate(t *testing.T) {
	run := runner(t)
	proj := laravelProject(t)
	sub := filepath.Join(proj, "deep", "subdir")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	run("init", "--path", proj, "--name", "myapp")

	t.Chdir(sub)

	stdout, _, code := run("export")
	if code != cmd.ExitOK {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(stdout, "myapp") {
		t.Errorf("export from subdir failed to find parent project: %s", stdout)
	}
}

func TestExport_OutsideProject_NoActiveProject_EmitsNothing(t *testing.T) {
	run := runner(t)
	outside := t.TempDir()
	t.Chdir(outside)

	stdout, _, code := run("export")
	if code != cmd.ExitOK {
		t.Fatalf("code=%d", code)
	}
	if stdout != "" {
		t.Errorf("export outside any project should be empty, got: %q", stdout)
	}
}

func TestExport_AlreadyInActiveProject_EmitsNothing(t *testing.T) {
	run := runner(t)
	proj := laravelProject(t)
	run("init", "--path", proj, "--name", "myapp")

	t.Chdir(proj)
	t.Setenv("LANE_ACTIVE_PROJECT", "myapp")

	stdout, _, code := run("export")
	if code != cmd.ExitOK {
		t.Fatalf("code=%d", code)
	}
	if stdout != "" {
		t.Errorf("export when already active should be empty, got: %q", stdout)
	}
}

func TestExport_LeavingProject_EmitsDeactivate(t *testing.T) {
	run := runner(t)
	proj := laravelProject(t)
	run("init", "--path", proj, "--name", "myapp")

	outside := t.TempDir()
	t.Chdir(outside)
	t.Setenv("LANE_ACTIVE_PROJECT", "myapp")

	stdout, _, code := run("export")
	if code != cmd.ExitOK {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(stdout, "unset LANE_ACTIVE_PROJECT") {
		t.Errorf("export leaving project should emit unset: %s", stdout)
	}
	if !strings.Contains(stdout, "unset APP_PORT") {
		t.Errorf("export leaving project should unset port vars: %s", stdout)
	}
}

func TestExport_SwitchingProjects_EmitsDeactivateThenActivate(t *testing.T) {
	run := runner(t)
	projA := laravelProject(t)
	run("init", "--path", projA, "--name", "app-a")
	projB := laravelProject(t)
	run("init", "--path", projB, "--name", "app-b")

	t.Chdir(projB)
	t.Setenv("LANE_ACTIVE_PROJECT", "app-a")

	stdout, _, code := run("export")
	if code != cmd.ExitOK {
		t.Fatalf("code=%d", code)
	}
	unsetIdx := strings.Index(stdout, "unset LANE_ACTIVE_PROJECT")
	exportIdx := strings.Index(stdout, "export LANE_ACTIVE_PROJECT='app-b'")
	if unsetIdx == -1 || exportIdx == -1 {
		t.Fatalf("expected both unset+export, got: %s", stdout)
	}
	if unsetIdx >= exportIdx {
		t.Errorf("unset must precede export in switch output, got: %s", stdout)
	}
}

func TestExport_OrphanedActiveProject_EmitsBareUnset(t *testing.T) {
	run := runner(t)
	outside := t.TempDir()
	t.Chdir(outside)
	t.Setenv("LANE_ACTIVE_PROJECT", "ghost-app")

	stdout, _, code := run("export")
	if code != cmd.ExitOK {
		t.Fatalf("code=%d", code)
	}
	if stdout != "unset LANE_ACTIVE_PROJECT\n" {
		t.Errorf("orphan should clear marker only, got: %q", stdout)
	}
}

func TestExport_BadArgs_ReturnsUsage(t *testing.T) {
	run := runner(t)
	_, stderr, code := run("export", "extra")
	if code != cmd.ExitUsage {
		t.Errorf("code=%d, want %d", code, cmd.ExitUsage)
	}
	if !strings.Contains(stderr, "usage") {
		t.Errorf("stderr missing usage: %s", stderr)
	}
}

// -- fish shell ---------------------------------------------------------

func TestHook_Fish_EmitsOnPromptEvent(t *testing.T) {
	run := runner(t)
	stdout, _, code := run("hook", "fish")
	if code != cmd.ExitOK {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(stdout, "fish_prompt") {
		t.Errorf("fish hook missing fish_prompt event: %s", stdout)
	}
	if !strings.Contains(stdout, "lane export --shell fish") {
		t.Errorf("fish hook missing fish export call: %s", stdout)
	}
	if !strings.Contains(stdout, "| source") {
		t.Errorf("fish hook missing source pipe: %s", stdout)
	}
}

func TestUse_FishShell_EmitsFishSyntax(t *testing.T) {
	run := runner(t)
	proj := laravelProject(t)
	run("init", "--path", proj, "--name", "myapp")

	stdout, _, code := run("use", "--shell", "fish", "myapp")
	if code != cmd.ExitOK {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(stdout, "set -gx LANE_ACTIVE_PROJECT 'myapp'") {
		t.Errorf("missing fish-style LANE_ACTIVE_PROJECT: %s", stdout)
	}
	if !strings.Contains(stdout, "set -gx APP_PORT") {
		t.Errorf("missing fish-style APP_PORT: %s", stdout)
	}
	if strings.Contains(stdout, "export ") {
		t.Errorf("fish output should not contain 'export ': %s", stdout)
	}
}

func TestUnuse_FishShell_EmitsFishSyntax(t *testing.T) {
	run := runner(t)
	proj := laravelProject(t)
	run("init", "--path", proj, "--name", "myapp")
	t.Setenv("LANE_ACTIVE_PROJECT", "myapp")

	stdout, _, code := run("unuse", "--shell", "fish")
	if code != cmd.ExitOK {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(stdout, "set -e LANE_ACTIVE_PROJECT") {
		t.Errorf("missing fish-style unset: %s", stdout)
	}
	if strings.Contains(stdout, "unset ") {
		t.Errorf("fish output should not contain 'unset ': %s", stdout)
	}
}

func TestExport_FishShell_EmitsFishSyntax(t *testing.T) {
	run := runner(t)
	proj := laravelProject(t)
	run("init", "--path", proj, "--name", "myapp")

	t.Chdir(proj)

	stdout, _, code := run("export", "--shell", "fish")
	if code != cmd.ExitOK {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(stdout, "set -gx LANE_ACTIVE_PROJECT 'myapp'") {
		t.Errorf("missing fish-style export: %s", stdout)
	}
}

func TestShellFlag_InvalidValue_ReturnsUsage(t *testing.T) {
	run := runner(t)
	proj := laravelProject(t)
	run("init", "--path", proj, "--name", "myapp")

	_, stderr, code := run("use", "--shell", "powershell", "myapp")
	if code != cmd.ExitUsage {
		t.Errorf("code = %d, want %d", code, cmd.ExitUsage)
	}
	if !strings.Contains(stderr, "invalid --shell") {
		t.Errorf("stderr missing invalid-shell message: %s", stderr)
	}
}

func TestShellFlag_PosixDefault_StillWorks(t *testing.T) {
	// Existing flow with no --shell should keep emitting POSIX.
	run := runner(t)
	proj := laravelProject(t)
	run("init", "--path", proj, "--name", "myapp")

	stdout, _, code := run("use", "myapp")
	if code != cmd.ExitOK {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(stdout, "export LANE_ACTIVE_PROJECT='myapp'") {
		t.Errorf("default should be POSIX, got: %s", stdout)
	}
}
