package convention_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/tincke10/lane/internal/convention"
)

func writeCompose(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestValidate_NoComposeFile_ReturnsNil(t *testing.T) {
	dir := t.TempDir()
	got, err := convention.Validate(dir, []string{"APP_PORT"})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestValidate_AllVarsReferenced_ReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	writeCompose(t, dir, "docker-compose.yml", `
services:
  app:
    ports:
      - "${APP_PORT:-80}:80"
  db:
    ports:
      - "${FORWARD_DB_PORT:-3306}:3306"
`)
	got, err := convention.Validate(dir, []string{"APP_PORT", "FORWARD_DB_PORT"})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected no missing vars, got %v", got)
	}
}

func TestValidate_NoVarsReferenced_ReturnsAll(t *testing.T) {
	dir := t.TempDir()
	writeCompose(t, dir, "docker-compose.yml", `
services:
  app:
    ports:
      - "80:80"
  db:
    ports:
      - "3306:3306"
`)
	got, err := convention.Validate(dir, []string{"APP_PORT", "FORWARD_DB_PORT"})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	want := []string{"APP_PORT", "FORWARD_DB_PORT"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestValidate_PartialReferences_ReturnsOnlyMissing(t *testing.T) {
	dir := t.TempDir()
	writeCompose(t, dir, "docker-compose.yml", `
services:
  app:
    ports:
      - "${APP_PORT:-80}:80"
  db:
    ports:
      - "3306:3306"
`)
	got, err := convention.Validate(dir, []string{"APP_PORT", "FORWARD_DB_PORT"})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	want := []string{"FORWARD_DB_PORT"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestValidate_BraceFormPlain(t *testing.T) {
	dir := t.TempDir()
	writeCompose(t, dir, "compose.yml", `services: { app: { ports: ["${APP_PORT}:80"] } }`)
	got, _ := convention.Validate(dir, []string{"APP_PORT"})
	if len(got) != 0 {
		t.Errorf("${APP_PORT} not detected: missing=%v", got)
	}
}

func TestValidate_BraceFormWithError(t *testing.T) {
	dir := t.TempDir()
	writeCompose(t, dir, "compose.yml", `services: { app: { ports: ["${APP_PORT:?required}:80"] } }`)
	got, _ := convention.Validate(dir, []string{"APP_PORT"})
	if len(got) != 0 {
		t.Errorf("${APP_PORT:?required} not detected: missing=%v", got)
	}
}

func TestValidate_BareForm(t *testing.T) {
	dir := t.TempDir()
	writeCompose(t, dir, "compose.yml", "services:\n  app:\n    image: nginx\n    command: nginx -p $APP_PORT\n")
	got, _ := convention.Validate(dir, []string{"APP_PORT"})
	if len(got) != 0 {
		t.Errorf("bare $APP_PORT not detected: missing=%v", got)
	}
}

func TestValidate_PrefixCollision_NotMatched(t *testing.T) {
	dir := t.TempDir()
	// APP_PORT_EXTRA contains APP_PORT as prefix — must NOT match APP_PORT.
	writeCompose(t, dir, "docker-compose.yml", `
services:
  app:
    environment:
      - "${APP_PORT_EXTRA}=foo"
`)
	got, _ := convention.Validate(dir, []string{"APP_PORT"})
	if len(got) != 1 || got[0] != "APP_PORT" {
		t.Errorf("expected APP_PORT to be reported missing, got %v", got)
	}
}

func TestValidate_FindsAlternativeFilenames(t *testing.T) {
	cases := []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeCompose(t, dir, name, `ports: ["${APP_PORT:-80}:80"]`)
			got, err := convention.Validate(dir, []string{"APP_PORT"})
			if err != nil {
				t.Fatalf("Validate: %v", err)
			}
			if len(got) != 0 {
				t.Errorf("missing in %s: %v", name, got)
			}
		})
	}
}

func TestValidate_EmptyExpected_ReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	writeCompose(t, dir, "docker-compose.yml", `services: {}`)
	got, err := convention.Validate(dir, []string{})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}
