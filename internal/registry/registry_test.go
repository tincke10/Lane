package registry_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tincke10/lane/internal/registry"
)

func newEmpty(t *testing.T) (*registry.Registry, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "projects.toml")
	r, err := registry.LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom empty: %v", err)
	}
	return r, path
}

func sampleProject() registry.Project {
	return registry.Project{
		Path:  "/home/dev/proyecto-a",
		Stack: []string{"laravel", "vite", "mysql"},
		Ports: map[string]int{
			"APP_PORT":         8081,
			"VITE_PORT":        5174,
			"FORWARD_DB_PORT":  33061,
			"FORWARD_REDIS":    63791,
		},
		Created: time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
	}
}

func TestLoadFrom_MissingFile_ReturnsEmptyRegistry(t *testing.T) {
	r, _ := newEmpty(t)
	if len(r.Projects) != 0 {
		t.Fatalf("expected empty Projects, got %d", len(r.Projects))
	}
}

func TestSaveLoad_Roundtrip(t *testing.T) {
	r, path := newEmpty(t)

	want := sampleProject()
	if err := r.Add("proyecto-a", want); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := r.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	r2, err := registry.LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	got, ok := r2.Get("proyecto-a")
	if !ok {
		t.Fatal("expected proyecto-a after reload")
	}
	if got.Path != want.Path {
		t.Errorf("Path: got %q, want %q", got.Path, want.Path)
	}
	if len(got.Stack) != len(want.Stack) {
		t.Errorf("Stack length: got %d, want %d", len(got.Stack), len(want.Stack))
	}
	if got.Ports["APP_PORT"] != want.Ports["APP_PORT"] {
		t.Errorf("APP_PORT: got %d, want %d", got.Ports["APP_PORT"], want.Ports["APP_PORT"])
	}
	if !got.Created.Equal(want.Created) {
		t.Errorf("Created: got %v, want %v", got.Created, want.Created)
	}
}

func TestSave_CreatesConfigDir(t *testing.T) {
	nested := filepath.Join(t.TempDir(), "deeply", "nested", "projects.toml")
	r, err := registry.LoadFrom(nested)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Add("a", sampleProject()); err != nil {
		t.Fatal(err)
	}
	if err := r.Save(); err != nil {
		t.Fatalf("Save into nested path: %v", err)
	}
	if _, err := os.Stat(nested); err != nil {
		t.Fatalf("expected file at %s: %v", nested, err)
	}
}

func TestAdd_Duplicate_ReturnsErrAlreadyExists(t *testing.T) {
	r, _ := newEmpty(t)
	p := sampleProject()
	if err := r.Add("a", p); err != nil {
		t.Fatal(err)
	}
	err := r.Add("a", p)
	if !errors.Is(err, registry.ErrAlreadyExists) {
		t.Errorf("want ErrAlreadyExists, got %v", err)
	}
}

func TestRemove_Missing_ReturnsErrNotFound(t *testing.T) {
	r, _ := newEmpty(t)
	err := r.Remove("does-not-exist")
	if !errors.Is(err, registry.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestRemove_Existing_RemovesProject(t *testing.T) {
	r, _ := newEmpty(t)
	if err := r.Add("a", sampleProject()); err != nil {
		t.Fatal(err)
	}
	if err := r.Remove("a"); err != nil {
		t.Fatalf("Remove existing: %v", err)
	}
	if _, ok := r.Get("a"); ok {
		t.Error("expected a to be removed")
	}
}

func TestList_ReturnsSortedByName(t *testing.T) {
	r, _ := newEmpty(t)
	for _, name := range []string{"charlie", "alpha", "bravo"} {
		if err := r.Add(name, sampleProject()); err != nil {
			t.Fatal(err)
		}
	}
	got := r.List()
	want := []string{"alpha", "bravo", "charlie"}
	if len(got) != len(want) {
		t.Fatalf("length: got %d, want %d", len(got), len(want))
	}
	for i, np := range got {
		if np.Name != want[i] {
			t.Errorf("index %d: got %q, want %q", i, np.Name, want[i])
		}
	}
}

func TestLoadFrom_CorruptTOML_ReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projects.toml")
	if err := os.WriteFile(path, []byte("this is not [ valid toml ="), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.LoadFrom(path); err == nil {
		t.Fatal("expected error from corrupt TOML, got nil")
	}
}

func TestSave_NoPath_ReturnsErrNoPath(t *testing.T) {
	r := &registry.Registry{Projects: map[string]registry.Project{}}
	err := r.Save()
	if !errors.Is(err, registry.ErrNoPath) {
		t.Errorf("want ErrNoPath, got %v", err)
	}
}

func TestDefaultPath(t *testing.T) {
	tests := []struct {
		name string
		xdg  string
		want func(home string) string
	}{
		{
			name: "honors XDG_CONFIG_HOME when set",
			xdg:  "/tmp/customxdg",
			want: func(string) string {
				return "/tmp/customxdg/lane/projects.toml"
			},
		},
		{
			name: "falls back to $HOME/.config when XDG empty",
			xdg:  "",
			want: func(home string) string {
				return filepath.Join(home, ".config", "lane", "projects.toml")
			},
		},
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", tc.xdg)
			got := registry.DefaultPath()
			want := tc.want(home)
			if got != want {
				t.Errorf("DefaultPath = %q, want %q", got, want)
			}
		})
	}
}
