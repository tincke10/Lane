// Package registry persists the set of registered Lane projects and their
// reserved port assignments to a TOML file under the user's config directory.
package registry

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/pelletier/go-toml/v2"
)

var (
	ErrNotFound      = errors.New("project not found")
	ErrAlreadyExists = errors.New("project already exists")
	ErrNoPath        = errors.New("registry has no source path; use LoadFrom")
)

// Project is a registered project with its detected stack and reserved ports.
type Project struct {
	Path    string         `toml:"path"`
	Stack   []string       `toml:"stack"`
	Ports   map[string]int `toml:"ports"`
	Created time.Time      `toml:"created"`
}

// NamedProject pairs a project with its registry key.
type NamedProject struct {
	Name    string
	Project Project
}

// Registry is the persisted state of all registered projects.
type Registry struct {
	Projects map[string]Project `toml:"projects"`

	path string
}

// DefaultPath returns the canonical location of projects.toml.
// Honors $XDG_CONFIG_HOME, otherwise falls back to $HOME/.config/lane/projects.toml.
func DefaultPath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "lane", "projects.toml")
}

// Load reads the registry from DefaultPath(). A missing file is not an error;
// it returns an empty registry ready to be populated and saved.
func Load() (*Registry, error) {
	return LoadFrom(DefaultPath())
}

// LoadFrom reads the registry from an explicit path.
func LoadFrom(path string) (*Registry, error) {
	r := &Registry{
		Projects: make(map[string]Project),
		path:     path,
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return r, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read registry: %w", err)
	}

	if err := toml.Unmarshal(data, r); err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}
	if r.Projects == nil {
		r.Projects = make(map[string]Project)
	}
	return r, nil
}

// Save writes the registry atomically: marshal to a sibling .tmp file, then
// rename over the destination. Prevents corruption on Ctrl-C mid-write.
func (r *Registry) Save() error {
	if r.path == "" {
		return ErrNoPath
	}

	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := toml.Marshal(r)
	if err != nil {
		return fmt.Errorf("marshal registry: %w", err)
	}

	tmp := r.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write tmp file: %w", err)
	}
	if err := os.Rename(tmp, r.path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename tmp file: %w", err)
	}
	return nil
}

// Add registers a new project under name. Returns ErrAlreadyExists if taken.
func (r *Registry) Add(name string, p Project) error {
	if _, exists := r.Projects[name]; exists {
		return fmt.Errorf("%w: %s", ErrAlreadyExists, name)
	}
	r.Projects[name] = p
	return nil
}

// Get fetches a project by name. The bool reports whether it existed.
func (r *Registry) Get(name string) (Project, bool) {
	p, ok := r.Projects[name]
	return p, ok
}

// Remove deletes a project by name. Returns ErrNotFound if absent.
func (r *Registry) Remove(name string) error {
	if _, exists := r.Projects[name]; !exists {
		return fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	delete(r.Projects, name)
	return nil
}

// List returns all projects sorted by name for stable, predictable output.
func (r *Registry) List() []NamedProject {
	out := make([]NamedProject, 0, len(r.Projects))
	for name, p := range r.Projects {
		out = append(out, NamedProject{Name: name, Project: p})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}
