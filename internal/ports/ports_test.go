package ports_test

import (
	"errors"
	"net"
	"sort"
	"testing"

	"github.com/tincke10/lane/internal/ports"
)

// bindPort listens on 127.0.0.1 at the given port (0 = OS picks one) and
// returns the listener plus the actual port bound. Caller must Close.
func bindPort(t *testing.T, port int) (net.Listener, int) {
	t.Helper()
	addr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: port}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Fatalf("bind port %d: %v", port, err)
	}
	return l, l.Addr().(*net.TCPAddr).Port
}

func TestIsFree_BoundPort_ReturnsFalse(t *testing.T) {
	l, port := bindPort(t, 0)
	defer l.Close()

	if ports.IsFree(port) {
		t.Errorf("IsFree(%d) = true while bound, want false", port)
	}
}

func TestIsFree_AfterReleaseReturnsTrue(t *testing.T) {
	l, port := bindPort(t, 0)
	if err := l.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if !ports.IsFree(port) {
		t.Errorf("IsFree(%d) = false after release, want true", port)
	}
}

func TestAllocate_ReturnsBaseWhenFree(t *testing.T) {
	// Pick a definitely-free port via OS, release, then expect Allocate to return it.
	l, base := bindPort(t, 0)
	l.Close()

	got, err := ports.Allocate(base, map[int]struct{}{})
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	if got != base {
		t.Errorf("Allocate(%d, empty) = %d, want %d", base, got, base)
	}
}

func TestAllocate_SkipsReserved(t *testing.T) {
	l, base := bindPort(t, 0)
	l.Close()

	reserved := map[int]struct{}{
		base:     {},
		base + 1: {},
	}
	got, err := ports.Allocate(base, reserved)
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	if got == base || got == base+1 {
		t.Errorf("Allocate returned reserved port %d", got)
	}
	if got < base+2 {
		t.Errorf("Allocate = %d, expected >= %d", got, base+2)
	}
}

func TestAllocate_SkipsBoundPorts(t *testing.T) {
	// Keep base bound; Allocate must skip it.
	l, base := bindPort(t, 0)
	defer l.Close()

	got, err := ports.Allocate(base, map[int]struct{}{})
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	if got == base {
		t.Errorf("Allocate returned bound port %d", got)
	}
	if got <= base {
		t.Errorf("Allocate = %d, expected > %d", got, base)
	}
}

func TestAllocate_NoFreePort_ReturnsErrNoFreePort(t *testing.T) {
	// Fill the entire scan window with reserved entries so Allocate exhausts.
	base := 40000
	reserved := make(map[int]struct{}, ports.MaxScan)
	for i := 0; i < ports.MaxScan; i++ {
		reserved[base+i] = struct{}{}
	}

	_, err := ports.Allocate(base, reserved)
	if !errors.Is(err, ports.ErrNoFreePort) {
		t.Errorf("want ErrNoFreePort, got %v", err)
	}
}

func TestCheckCollisions_AllFree_ReturnsEmpty(t *testing.T) {
	// Grab two free ports.
	l1, p1 := bindPort(t, 0)
	l1.Close()
	l2, p2 := bindPort(t, 0)
	l2.Close()

	got := ports.CheckCollisions(map[string]int{
		"APP_PORT":  p1,
		"VITE_PORT": p2,
	})
	if len(got) != 0 {
		t.Errorf("CheckCollisions: got %d collisions, want 0 — %+v", len(got), got)
	}
}

func TestCheckCollisions_BoundPort_ReturnsCollision(t *testing.T) {
	l, bound := bindPort(t, 0)
	defer l.Close()

	free, freePort := bindPort(t, 0)
	free.Close()

	got := ports.CheckCollisions(map[string]int{
		"APP_PORT":  bound,
		"VITE_PORT": freePort,
	})
	if len(got) != 1 {
		t.Fatalf("CheckCollisions: got %d collisions, want 1 — %+v", len(got), got)
	}
	if got[0].Name != "APP_PORT" || got[0].Port != bound {
		t.Errorf("collision = %+v, want {APP_PORT %d}", got[0], bound)
	}
}

func TestCheckCollisions_ReturnsSortedByName(t *testing.T) {
	// Bind three ports so all collide; verify deterministic order.
	l1, p1 := bindPort(t, 0)
	defer l1.Close()
	l2, p2 := bindPort(t, 0)
	defer l2.Close()
	l3, p3 := bindPort(t, 0)
	defer l3.Close()

	got := ports.CheckCollisions(map[string]int{
		"CHARLIE": p1,
		"ALPHA":   p2,
		"BRAVO":   p3,
	})
	if len(got) != 3 {
		t.Fatalf("want 3 collisions, got %d", len(got))
	}
	names := []string{got[0].Name, got[1].Name, got[2].Name}
	want := []string{"ALPHA", "BRAVO", "CHARLIE"}
	if !sort.StringsAreSorted(names) {
		t.Errorf("names not sorted: %v", names)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("index %d: got %q, want %q", i, names[i], want[i])
		}
	}
}
