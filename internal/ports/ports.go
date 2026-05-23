// Package ports verifies TCP port availability on the loopback interface
// and allocates free ports above a base, skipping a set of ports already
// reserved by other registered projects.
package ports

import (
	"errors"
	"fmt"
	"net"
	"sort"
	"strconv"
)

// MaxScan caps how many consecutive ports Allocate will probe before giving up.
const MaxScan = 1000

// ErrNoFreePort is returned when Allocate exhausts MaxScan without finding a
// port that is both unreserved and currently bindable.
var ErrNoFreePort = errors.New("no free port available")

// Collision describes a port assignment that cannot be claimed because the
// port is currently in use on the loopback interface.
type Collision struct {
	Name string
	Port int
}

// IsFree reports whether port can be bound on 127.0.0.1 right now.
// The result is inherently racy: another process can claim the port between
// this check and the caller's actual use. Callers must treat it as advisory.
func IsFree(port int) bool {
	l, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		return false
	}
	_ = l.Close()
	return true
}

// Allocate returns the lowest port p in [base, base+MaxScan) such that p is
// not present in reserved and IsFree(p) is true. Returns ErrNoFreePort when
// the window is exhausted.
func Allocate(base int, reserved map[int]struct{}) (int, error) {
	for i := 0; i < MaxScan; i++ {
		p := base + i
		if _, taken := reserved[p]; taken {
			continue
		}
		if IsFree(p) {
			return p, nil
		}
	}
	return 0, fmt.Errorf("%w: scanned %d ports from %d", ErrNoFreePort, MaxScan, base)
}

// CheckCollisions returns, sorted by Name, the entries of assignments whose
// port is not currently free on the loopback interface.
func CheckCollisions(assignments map[string]int) []Collision {
	out := make([]Collision, 0)
	for name, port := range assignments {
		if !IsFree(port) {
			out = append(out, Collision{Name: name, Port: port})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}
