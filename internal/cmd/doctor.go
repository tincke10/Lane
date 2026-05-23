package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tincke10/lane/internal/activator"
	"github.com/tincke10/lane/internal/doctor"
	"github.com/tincke10/lane/internal/registry"
)

func cmdDoctor(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "lane: usage: lane doctor")
		return ExitUsage
	}

	reg, err := registry.Load()
	if err != nil {
		fmt.Fprintf(stderr, "lane: load registry: %v\n", err)
		return ExitError
	}

	active := os.Getenv(activator.ActiveProjectEnvVar)
	report := doctor.Check(reg, active)

	formatDoctorReport(stdout, report, active)

	if report.MaxSeverity() == doctor.SeverityError {
		return ExitError
	}
	return ExitOK
}

func formatDoctorReport(w io.Writer, r doctor.Report, active string) {
	if len(r.Projects) == 0 && len(r.GlobalIssues) == 0 {
		fmt.Fprintln(w, "no projects registered. run `lane init` inside a project.")
		return
	}

	if len(r.Projects) > 0 {
		fmt.Fprintf(w, "checking %d registered project(s)...\n\n", len(r.Projects))
	}

	okCount, warnCount, errCount := 0, 0, 0
	for _, p := range r.Projects {
		switch p.MaxSeverity() {
		case doctor.SeverityOK:
			okCount++
		case doctor.SeverityWarn:
			warnCount++
		case doctor.SeverityError:
			errCount++
		}
		writeProjectReport(w, p)
	}

	if active != "" {
		fmt.Fprintf(w, "active project: %s\n", active)
	} else {
		fmt.Fprintln(w, "active project: (none)")
	}
	for _, issue := range r.GlobalIssues {
		fmt.Fprintf(w, "  - [%s] %s\n", string(issue.Severity), issue.Message)
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "%d project(s) checked: %d ok, %d warn, %d error\n",
		len(r.Projects), okCount, warnCount, errCount)
}

func writeProjectReport(w io.Writer, p doctor.ProjectReport) {
	tag := fmt.Sprintf("[%s]", string(p.MaxSeverity()))
	fmt.Fprintf(w, "%-8s %s\n", tag, p.Name)
	fmt.Fprintf(w, "         path:  %s\n", p.Path)
	if len(p.Stack) > 0 {
		fmt.Fprintf(w, "         stack: %s\n", strings.Join(p.Stack, ", "))
	}
	for _, env := range sortedEnvKeys(p.Ports) {
		fmt.Fprintf(w, "         %s=%d\n", env, p.Ports[env])
	}
	for _, issue := range p.Issues {
		fmt.Fprintf(w, "         - [%s] %s\n", string(issue.Severity), issue.Message)
	}
	fmt.Fprintln(w)
}
