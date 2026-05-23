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

func TestBuildNextArgs_DashP(t *testing.T) {
	got := buildNextArgs(nil, "3001")
	want := []string{"npx", "next", "dev", "-p", "3001"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBuildNextArgs_ForwardsExtras(t *testing.T) {
	got := buildNextArgs([]string{"--turbo"}, "3001")
	want := []string{"npx", "next", "dev", "-p", "3001", "--turbo"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBuildFlaskArgs_PortFlag(t *testing.T) {
	got := buildFlaskArgs(nil, "5001")
	want := []string{"flask", "run", "--port=5001"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBuildFlaskArgs_ForwardsExtras(t *testing.T) {
	got := buildFlaskArgs([]string{"--debug", "--host=0.0.0.0"}, "5001")
	want := []string{"flask", "run", "--port=5001", "--debug", "--host=0.0.0.0"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBuildDjangoArgs_PositionalPort(t *testing.T) {
	got := buildDjangoArgs(nil, "8001")
	want := []string{"python", "manage.py", "runserver", "8001"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBuildDjangoArgs_ForwardsExtras(t *testing.T) {
	got := buildDjangoArgs([]string{"--noreload"}, "8001")
	want := []string{"python", "manage.py", "runserver", "8001", "--noreload"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// -- portBasesFor precedence --------------------------------------------

func TestPortBasesFor_LaravelAlone(t *testing.T) {
	got := portBasesFor([]string{"laravel", "php"})
	if got["APP_PORT"] != 8080 {
		t.Errorf("Laravel APP_PORT = %d, want 8080", got["APP_PORT"])
	}
}

func TestPortBasesFor_DjangoAlone(t *testing.T) {
	got := portBasesFor([]string{"django", "python"})
	if got["APP_PORT"] != 8000 {
		t.Errorf("Django APP_PORT = %d, want 8000", got["APP_PORT"])
	}
}

func TestPortBasesFor_FlaskAlone(t *testing.T) {
	got := portBasesFor([]string{"flask", "python"})
	if got["APP_PORT"] != 5000 {
		t.Errorf("Flask APP_PORT = %d, want 5000", got["APP_PORT"])
	}
}

func TestPortBasesFor_NextjsAlone(t *testing.T) {
	got := portBasesFor([]string{"nextjs", "node"})
	if got["APP_PORT"] != 3000 {
		t.Errorf("Next.js APP_PORT = %d, want 3000", got["APP_PORT"])
	}
}

func TestPortBasesFor_LaravelBeatsNextjs(t *testing.T) {
	// Precedence rule: laravel > nextjs when both present.
	got := portBasesFor([]string{"laravel", "nextjs", "node", "php"})
	if got["APP_PORT"] != 8080 {
		t.Errorf("Laravel+Next APP_PORT = %d, want 8080 (laravel wins)", got["APP_PORT"])
	}
}

func TestPortBasesFor_DjangoBeatsFlask(t *testing.T) {
	got := portBasesFor([]string{"django", "flask", "python"})
	if got["APP_PORT"] != 8000 {
		t.Errorf("Django+Flask APP_PORT = %d, want 8000 (django wins)", got["APP_PORT"])
	}
}

func TestPortBasesFor_MysqlBeatsPostgres(t *testing.T) {
	got := portBasesFor([]string{"mysql", "postgres"})
	if got["FORWARD_DB_PORT"] != 33060 {
		t.Errorf("mysql+postgres FORWARD_DB_PORT = %d, want 33060", got["FORWARD_DB_PORT"])
	}
}

func TestPortBasesFor_ViteIndependentFromAppPort(t *testing.T) {
	// Laravel sets APP_PORT, vite sets VITE_PORT — they coexist.
	got := portBasesFor([]string{"laravel", "vite", "node", "php"})
	if got["APP_PORT"] != 8080 {
		t.Errorf("APP_PORT = %d, want 8080", got["APP_PORT"])
	}
	if got["VITE_PORT"] != 5173 {
		t.Errorf("VITE_PORT = %d, want 5173", got["VITE_PORT"])
	}
}

func TestPortBasesFor_NoFrameworkKnown_ReturnsEmpty(t *testing.T) {
	got := portBasesFor([]string{"docker"})
	if len(got) != 0 {
		t.Errorf("no recognized framework should yield no allocations, got %v", got)
	}
}
