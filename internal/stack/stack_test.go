package stack_test

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/tincke10/lane/internal/stack"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestDetect_EmptyDir_ReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	got, err := stack.Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Detect empty dir = %v, want []", got)
	}
}

func TestDetect_NonexistentPath_ReturnsError(t *testing.T) {
	_, err := stack.Detect("/path/that/does/not/exist/lane-test")
	if err == nil {
		t.Fatal("expected error for nonexistent path, got nil")
	}
}

func TestDetect_ComposerWithLaravel_ReturnsLaravelAndPhp(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "composer.json", `{
  "require": {
    "php": "^8.2",
    "laravel/framework": "^11.0"
  }
}`)

	got, err := stack.Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	want := []string{"laravel", "php"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Detect = %v, want %v", got, want)
	}
}

func TestDetect_ComposerWithoutLaravel_ReturnsOnlyPhp(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "composer.json", `{
  "require": {
    "php": "^8.2",
    "symfony/console": "^7.0"
  }
}`)

	got, err := stack.Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	want := []string{"php"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Detect = %v, want %v", got, want)
	}
}

func TestDetect_ComposerMalformed_StillReturnsPhp(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "composer.json", `{ this is not valid json`)

	got, err := stack.Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	want := []string{"php"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Detect with malformed composer = %v, want %v", got, want)
	}
}

func TestDetect_PackageJsonWithVite_ReturnsNodeAndVite(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
  "name": "frontend",
  "devDependencies": {
    "vite": "^5.0.0"
  }
}`)

	got, err := stack.Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	want := []string{"node", "vite"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Detect = %v, want %v", got, want)
	}
}

func TestDetect_PackageJsonViteInDependencies_ReturnsVite(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
  "name": "frontend",
  "dependencies": {
    "vite": "^5.0.0"
  }
}`)

	got, err := stack.Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	want := []string{"node", "vite"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Detect = %v, want %v", got, want)
	}
}

func TestDetect_PackageJsonWithoutVite_ReturnsOnlyNode(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
  "name": "lib",
  "dependencies": {
    "lodash": "^4.0.0"
  }
}`)

	got, err := stack.Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	want := []string{"node"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Detect = %v, want %v", got, want)
	}
}

func TestDetect_RequirementsTxt_ReturnsPython(t *testing.T) {
	dir := t.TempDir()
	// requests is a plain library that does not match flask/django markers.
	writeFile(t, dir, "requirements.txt", "requests==2.31\n")

	got, err := stack.Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	want := []string{"python"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Detect = %v, want %v", got, want)
	}
}

func TestDetect_PyprojectToml_ReturnsPython(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", "[tool.poetry]\nname = \"x\"\n")

	got, err := stack.Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	want := []string{"python"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Detect = %v, want %v", got, want)
	}
}

func TestDetect_DockerComposeWithServices_ReturnsDockerAndServices(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "docker-compose.yml", `
services:
  app:
    build: .
  db:
    image: mysql:8.0
  cache:
    image: redis:7
`)

	got, err := stack.Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	want := []string{"docker", "mysql", "redis"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Detect = %v, want %v", got, want)
	}
}

func TestDetect_ComposeYaml_AlsoDetected(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "compose.yaml", `
services:
  db:
    image: postgres:16
`)

	got, err := stack.Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	want := []string{"docker", "postgres"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Detect = %v, want %v", got, want)
	}
}

func TestDetect_PackageJsonWithNext_ReturnsNodeAndNextjs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
  "name": "site",
  "dependencies": {"next": "^14.0.0", "react": "^18.0.0"}
}`)
	got, err := stack.Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	want := []string{"nextjs", "node"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Detect = %v, want %v", got, want)
	}
}

func TestDetect_RequirementsWithFlask_ReturnsFlask(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "flask==3.0.0\ngunicorn==21\n")
	got, _ := stack.Detect(dir)
	want := []string{"flask", "python"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Detect = %v, want %v", got, want)
	}
}

func TestDetect_RequirementsWithDjango_ReturnsDjango(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "Django==5.0\npsycopg2==2.9\n")
	got, _ := stack.Detect(dir)
	want := []string{"django", "python"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Detect = %v, want %v", got, want)
	}
}

func TestDetect_ManagePy_ReturnsDjango(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "manage.py", "#!/usr/bin/env python\nimport django\n")
	got, _ := stack.Detect(dir)
	want := []string{"django", "python"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Detect = %v, want %v", got, want)
	}
}

func TestDetect_PyprojectWithFlask_ReturnsFlask(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", `[project]
name = "myapp"
dependencies = ["flask>=3.0", "requests"]
`)
	got, _ := stack.Detect(dir)
	want := []string{"flask", "python"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Detect = %v, want %v", got, want)
	}
}

func TestDetect_FullLaravelStack_ReturnsAllMarkers(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "composer.json", `{"require":{"laravel/framework":"^11.0"}}`)
	writeFile(t, dir, "package.json", `{"devDependencies":{"vite":"^5.0.0"}}`)
	writeFile(t, dir, "docker-compose.yml", `
services:
  db:
    image: mysql:8.0
  cache:
    image: redis:7
`)

	got, err := stack.Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	want := []string{"docker", "laravel", "mysql", "node", "php", "redis", "vite"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Detect full stack = %v, want %v", got, want)
	}
	if !sort.StringsAreSorted(got) {
		t.Errorf("result not sorted: %v", got)
	}
}
