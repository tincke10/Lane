package cmd

import (
	"reflect"
	"testing"
)

// Build-arg helpers are unit-tested directly because they're pure.
// The exec side of cmdServe/cmdVite is exercised by smoke tests at the
// CLI level — syscall.Exec replaces the process so it cannot be unit-tested
// without an integration harness.

func TestBuildServeArgs_DefaultPort(t *testing.T) {
	got := buildServeArgs(nil, "8000")
	want := []string{"php", "artisan", "serve", "--port=8000"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBuildServeArgs_WithLanePort(t *testing.T) {
	got := buildServeArgs(nil, "8081")
	want := []string{"php", "artisan", "serve", "--port=8081"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBuildServeArgs_ForwardsExtras(t *testing.T) {
	got := buildServeArgs([]string{"--host=0.0.0.0", "--tries=5"}, "8081")
	want := []string{"php", "artisan", "serve", "--port=8081", "--host=0.0.0.0", "--tries=5"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBuildServeArgs_PortIsInjectedBeforeExtras(t *testing.T) {
	// We want the user's extras to be able to OVERRIDE the injected
	// port (later args win in CLI tools), so --port must come first.
	got := buildServeArgs([]string{"--port=9999"}, "8081")
	if got[3] != "--port=8081" {
		t.Errorf("Lane's --port should be argv[3], got: %v", got)
	}
	if got[4] != "--port=9999" {
		t.Errorf("user override should come after Lane's port, got: %v", got)
	}
}

func TestBuildViteArgs_DefaultPort(t *testing.T) {
	got := buildViteArgs(nil, "5173")
	want := []string{"npx", "vite", "--port=5173"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBuildViteArgs_WithLanePort(t *testing.T) {
	got := buildViteArgs(nil, "5174")
	want := []string{"npx", "vite", "--port=5174"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBuildViteArgs_ForwardsExtras(t *testing.T) {
	got := buildViteArgs([]string{"--host", "0.0.0.0", "--open"}, "5174")
	want := []string{"npx", "vite", "--port=5174", "--host", "0.0.0.0", "--open"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestEnvOrDefault_EmptyFallsBack(t *testing.T) {
	t.Setenv("LANE_TEST_VAR_DOES_NOT_EXIST", "")
	if got := envOrDefault("LANE_TEST_VAR_DOES_NOT_EXIST", "fallback"); got != "fallback" {
		t.Errorf("got %q, want fallback", got)
	}
}

func TestEnvOrDefault_NonEmptyReturned(t *testing.T) {
	t.Setenv("LANE_TEST_VAR", "from-env")
	if got := envOrDefault("LANE_TEST_VAR", "fallback"); got != "from-env" {
		t.Errorf("got %q, want from-env", got)
	}
}
