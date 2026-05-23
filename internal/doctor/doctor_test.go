package doctor_test

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tincke10/lane/internal/doctor"
	"github.com/tincke10/lane/internal/registry"
)

func newRegistry(t *testing.T) *registry.Registry {
	t.Helper()
	r, err := registry.LoadFrom(filepath.Join(t.TempDir(), "projects.toml"))
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	return r
}

func addProject(t *testing.T, r *registry.Registry, name string, p registry.Project) {
	t.Helper()
	if p.Created.IsZero() {
		p.Created = time.Now().UTC()
	}
	if err := r.Add(name, p); err != nil {
		t.Fatalf("Add %q: %v", name, err)
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return port
}

func bindPort(t *testing.T) (net.Listener, int) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	return l, l.Addr().(*net.TCPAddr).Port
}

func makeProjectDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

func TestCheck_EmptyRegistry_NoProjectsNoIssues(t *testing.T) {
	reg := newRegistry(t)
	report := doctor.Check(reg, "")
	if len(report.Projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(report.Projects))
	}
	if len(report.GlobalIssues) != 0 {
		t.Errorf("expected 0 global issues, got %d", len(report.GlobalIssues))
	}
	if report.MaxSeverity() != doctor.SeverityOK {
		t.Errorf("MaxSeverity = %v, want OK", report.MaxSeverity())
	}
}

func TestCheck_HealthyProject_NoIssues(t *testing.T) {
	reg := newRegistry(t)
	dir := makeProjectDir(t, map[string]string{
		"composer.json": `{"require":{"laravel/framework":"^11"}}`,
	})
	addProject(t, reg, "p", registry.Project{
		Path:  dir,
		Stack: []string{"laravel", "php"},
		Ports: map[string]int{"APP_PORT": freePort(t)},
	})

	report := doctor.Check(reg, "")
	if len(report.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(report.Projects))
	}
	pr := report.Projects[0]
	if pr.MaxSeverity() != doctor.SeverityOK {
		t.Errorf("MaxSeverity = %v, issues=%+v", pr.MaxSeverity(), pr.Issues)
	}
}

func TestCheck_MissingPath_ReturnsError(t *testing.T) {
	reg := newRegistry(t)
	addProject(t, reg, "ghost", registry.Project{
		Path:  "/does/not/exist/lane-ghost",
		Stack: []string{"laravel"},
		Ports: map[string]int{"APP_PORT": freePort(t)},
	})

	report := doctor.Check(reg, "")
	pr := report.Projects[0]
	if pr.MaxSeverity() != doctor.SeverityError {
		t.Errorf("MaxSeverity = %v, want Error", pr.MaxSeverity())
	}
	if !hasIssue(pr.Issues, doctor.SeverityError, "path") {
		t.Errorf("expected error mentioning 'path', got: %+v", pr.Issues)
	}
}

func TestCheck_PortCollisionBetweenProjects_ReturnsError(t *testing.T) {
	reg := newRegistry(t)
	dir1 := makeProjectDir(t, map[string]string{"composer.json": `{}`})
	dir2 := makeProjectDir(t, map[string]string{"composer.json": `{}`})
	port := freePort(t)

	addProject(t, reg, "a", registry.Project{
		Path: dir1, Stack: []string{"php"}, Ports: map[string]int{"APP_PORT": port},
	})
	addProject(t, reg, "b", registry.Project{
		Path: dir2, Stack: []string{"php"}, Ports: map[string]int{"APP_PORT": port},
	})

	report := doctor.Check(reg, "")
	foundA, foundB := false, false
	for _, pr := range report.Projects {
		if hasIssue(pr.Issues, doctor.SeverityError, "also reserved") {
			if pr.Name == "a" {
				foundA = true
			}
			if pr.Name == "b" {
				foundB = true
			}
		}
	}
	if !foundA || !foundB {
		t.Errorf("expected collision errors on both projects, foundA=%v foundB=%v\n%+v", foundA, foundB, report)
	}
}

func TestCheck_BoundPort_ReturnsWarning(t *testing.T) {
	reg := newRegistry(t)
	dir := makeProjectDir(t, map[string]string{"composer.json": `{}`})
	l, port := bindPort(t)
	defer l.Close()

	addProject(t, reg, "p", registry.Project{
		Path: dir, Stack: []string{"php"}, Ports: map[string]int{"APP_PORT": port},
	})

	report := doctor.Check(reg, "")
	pr := report.Projects[0]
	if pr.MaxSeverity() != doctor.SeverityWarn {
		t.Errorf("MaxSeverity = %v, want Warn — issues=%+v", pr.MaxSeverity(), pr.Issues)
	}
	if !hasIssue(pr.Issues, doctor.SeverityWarn, "in use") {
		t.Errorf("expected 'in use' warn, got: %+v", pr.Issues)
	}
}

func TestCheck_StackDrift_ReturnsWarning(t *testing.T) {
	reg := newRegistry(t)
	dir := t.TempDir() // empty dir — no markers
	addProject(t, reg, "drifted", registry.Project{
		Path:  dir,
		Stack: []string{"laravel"}, // claims laravel but no composer.json present
		Ports: map[string]int{"APP_PORT": freePort(t)},
	})

	report := doctor.Check(reg, "")
	pr := report.Projects[0]
	if !hasIssue(pr.Issues, doctor.SeverityWarn, "drift") {
		t.Errorf("expected drift warn, got: %+v", pr.Issues)
	}
}

func TestCheck_OrphanedActiveProject_ReturnsGlobalWarning(t *testing.T) {
	reg := newRegistry(t)
	report := doctor.Check(reg, "ghost-project")
	if !hasIssue(report.GlobalIssues, doctor.SeverityWarn, "LANE_ACTIVE_PROJECT") {
		t.Errorf("expected global LANE_ACTIVE_PROJECT warn, got: %+v", report.GlobalIssues)
	}
}

func TestCheck_ActiveProjectInRegistry_NoGlobalIssue(t *testing.T) {
	reg := newRegistry(t)
	dir := makeProjectDir(t, map[string]string{"composer.json": `{}`})
	addProject(t, reg, "p", registry.Project{
		Path: dir, Stack: []string{"php"}, Ports: map[string]int{"APP_PORT": freePort(t)},
	})

	report := doctor.Check(reg, "p")
	if len(report.GlobalIssues) != 0 {
		t.Errorf("expected no global issues, got: %+v", report.GlobalIssues)
	}
}

func TestReport_MaxSeverity_PicksWorst(t *testing.T) {
	r := doctor.Report{
		Projects: []doctor.ProjectReport{
			{Name: "a", Issues: []doctor.Issue{{Severity: doctor.SeverityWarn, Message: "x"}}},
			{Name: "b", Issues: []doctor.Issue{{Severity: doctor.SeverityError, Message: "y"}}},
		},
	}
	if got := r.MaxSeverity(); got != doctor.SeverityError {
		t.Errorf("MaxSeverity = %v, want Error", got)
	}
}

func TestReport_MaxSeverity_OnlyWarnings(t *testing.T) {
	r := doctor.Report{
		Projects: []doctor.ProjectReport{
			{Name: "a", Issues: []doctor.Issue{{Severity: doctor.SeverityWarn, Message: "x"}}},
		},
	}
	if got := r.MaxSeverity(); got != doctor.SeverityWarn {
		t.Errorf("MaxSeverity = %v, want Warn", got)
	}
}

func TestReport_MaxSeverity_GlobalIssueCounts(t *testing.T) {
	r := doctor.Report{
		GlobalIssues: []doctor.Issue{{Severity: doctor.SeverityError, Message: "x"}},
	}
	if got := r.MaxSeverity(); got != doctor.SeverityError {
		t.Errorf("MaxSeverity = %v, want Error", got)
	}
}

// hasIssue reports whether any issue in xs has the given severity and a
// message containing sub (case-insensitive).
func hasIssue(xs []doctor.Issue, sev doctor.Severity, sub string) bool {
	subL := strings.ToLower(sub)
	for _, i := range xs {
		if i.Severity == sev && strings.Contains(strings.ToLower(i.Message), subL) {
			return true
		}
	}
	return false
}
