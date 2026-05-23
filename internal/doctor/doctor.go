// Package doctor diagnoses health issues with the Lane registry: missing
// project paths, port collisions across projects, ports currently bound
// by other processes, stack drift since registration, and orphaned
// active-project env vars.
//
// Doctor is pure logic — it does not print or read shell state. The
// caller passes the registry and the value of LANE_ACTIVE_PROJECT (or
// the empty string), and receives a Report.
package doctor

import (
	"fmt"
	"os"
	"sort"

	"github.com/tincke10/lane/internal/ports"
	"github.com/tincke10/lane/internal/registry"
	"github.com/tincke10/lane/internal/stack"
)

// Severity ranks the diagnostic findings.
type Severity string

const (
	SeverityOK    Severity = "ok"
	SeverityWarn  Severity = "warn"
	SeverityError Severity = "error"
)

var severityRank = map[Severity]int{
	SeverityOK:    0,
	SeverityWarn:  1,
	SeverityError: 2,
}

// Issue is a single diagnostic finding attached to a project or to the
// overall environment.
type Issue struct {
	Severity Severity
	Message  string
}

// ProjectReport bundles the registry data for a project with the issues
// doctor found while inspecting it.
type ProjectReport struct {
	Name   string
	Path   string
	Stack  []string
	Ports  map[string]int
	Issues []Issue
}

// MaxSeverity returns the worst severity among the project's issues, or
// SeverityOK when there are none.
func (pr ProjectReport) MaxSeverity() Severity {
	worst := SeverityOK
	for _, i := range pr.Issues {
		if severityRank[i.Severity] > severityRank[worst] {
			worst = i.Severity
		}
	}
	return worst
}

// Report is the full doctor output: one entry per project plus any
// global issues that are not tied to a specific project (e.g. an
// orphaned LANE_ACTIVE_PROJECT).
type Report struct {
	Projects     []ProjectReport
	GlobalIssues []Issue
}

// MaxSeverity returns the worst severity across all projects and global issues.
func (r Report) MaxSeverity() Severity {
	worst := SeverityOK
	for _, p := range r.Projects {
		if ps := p.MaxSeverity(); severityRank[ps] > severityRank[worst] {
			worst = ps
		}
	}
	for _, i := range r.GlobalIssues {
		if severityRank[i.Severity] > severityRank[worst] {
			worst = i.Severity
		}
	}
	return worst
}

// Check inspects reg and the running environment and returns a Report.
// activeProject is the value of LANE_ACTIVE_PROJECT (empty when unset).
//
// Diagnostics:
//   - error: project path does not exist or is not a directory
//   - error: same port reserved by two or more projects in the registry
//   - warn:  reserved port is currently bound (often the project itself running)
//   - warn:  stack drift — registered stack differs from current detection
//   - warn:  LANE_ACTIVE_PROJECT references a project not in the registry
func Check(reg *registry.Registry, activeProject string) Report {
	rep := Report{}

	// Build a port → owners map up front so each project can report
	// cross-project collisions without re-scanning.
	portOwners := make(map[int][]string, 16)
	for name, p := range reg.Projects {
		for _, port := range p.Ports {
			portOwners[port] = append(portOwners[port], name)
		}
	}
	for port := range portOwners {
		sort.Strings(portOwners[port])
	}

	for _, np := range reg.List() {
		pr := ProjectReport{
			Name:  np.Name,
			Path:  np.Project.Path,
			Stack: np.Project.Stack,
			Ports: np.Project.Ports,
		}

		pathOK := checkPath(np.Project.Path, &pr)
		if pathOK {
			checkStackDrift(np.Project, &pr)
		}
		checkCrossProjectCollisions(np.Name, np.Project.Ports, portOwners, &pr)
		checkBoundPorts(np.Project.Ports, &pr)

		rep.Projects = append(rep.Projects, pr)
	}

	if activeProject != "" {
		if _, ok := reg.Get(activeProject); !ok {
			rep.GlobalIssues = append(rep.GlobalIssues, Issue{
				Severity: SeverityWarn,
				Message: fmt.Sprintf(
					"LANE_ACTIVE_PROJECT=%q is set but the project is not in the registry",
					activeProject),
			})
		}
	}

	return rep
}

func checkPath(path string, pr *ProjectReport) bool {
	info, err := os.Stat(path)
	if err != nil {
		pr.Issues = append(pr.Issues, Issue{
			Severity: SeverityError,
			Message:  fmt.Sprintf("path does not exist: %s", path),
		})
		return false
	}
	if !info.IsDir() {
		pr.Issues = append(pr.Issues, Issue{
			Severity: SeverityError,
			Message:  fmt.Sprintf("path is not a directory: %s", path),
		})
		return false
	}
	return true
}

func checkStackDrift(p registry.Project, pr *ProjectReport) {
	detected, err := stack.Detect(p.Path)
	if err != nil {
		// Stat passed but Detect failed — surface but don't block.
		return
	}
	if !sameStringSet(p.Stack, detected) {
		pr.Issues = append(pr.Issues, Issue{
			Severity: SeverityWarn,
			Message: fmt.Sprintf(
				"stack drift: registered as %v, now detected as %v", p.Stack, detected),
		})
	}
}

func checkCrossProjectCollisions(
	name string,
	portsMap map[string]int,
	portOwners map[int][]string,
	pr *ProjectReport,
) {
	for env, port := range portsMap {
		owners := portOwners[port]
		if len(owners) <= 1 {
			continue
		}
		others := make([]string, 0, len(owners)-1)
		for _, o := range owners {
			if o != name {
				others = append(others, o)
			}
		}
		pr.Issues = append(pr.Issues, Issue{
			Severity: SeverityError,
			Message: fmt.Sprintf(
				"%s=%d also reserved by: %v", env, port, others),
		})
	}
}

func checkBoundPorts(portsMap map[string]int, pr *ProjectReport) {
	for _, c := range ports.CheckCollisions(portsMap) {
		pr.Issues = append(pr.Issues, Issue{
			Severity: SeverityWarn,
			Message:  fmt.Sprintf("%s=%d is currently in use", c.Name, c.Port),
		})
	}
}

// sameStringSet reports whether a and b contain the same elements,
// regardless of order. Both slices are expected to have no duplicates.
func sameStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	set := make(map[string]struct{}, len(a))
	for _, s := range a {
		set[s] = struct{}{}
	}
	for _, s := range b {
		if _, ok := set[s]; !ok {
			return false
		}
	}
	return true
}
