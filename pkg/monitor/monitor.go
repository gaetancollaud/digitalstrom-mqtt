// Package monitor tracks in-flight bridge work without polling devices or
// treating an idle home or a disconnected dependency as a deadlock.
package monitor

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

const WorkTimeout = 2 * time.Minute

type permanentError struct{ error }

func (e permanentError) Unwrap() error { return e.error }

func Permanent(err error) error { return permanentError{err} }

func IsPermanent(err error) bool {
	var target permanentError
	return errors.As(err, &target)
}

// ExitCode is used only by the App launcher: 75 retries, 78 needs user action.
func ExitCode(err error) int {
	if IsPermanent(err) {
		return 78
	}
	return 75
}

type work struct {
	name  string
	since time.Time
}

type Monitor struct {
	mu      sync.Mutex
	next    uint64
	active  map[uint64]work
	failure chan error
	now     func() time.Time
}

func New() *Monitor {
	return &Monitor{active: make(map[uint64]work), failure: make(chan error, 1), now: time.Now}
}

// Begin must wrap actual work, not an idle socket read or a reconnect delay.
func (m *Monitor) Begin(name string) func() {
	if m == nil {
		return func() {}
	}
	m.mu.Lock()
	m.next++
	id := m.next
	m.active[id] = work{name: name, since: m.now()}
	m.mu.Unlock()
	return func() { m.mu.Lock(); delete(m.active, id); m.mu.Unlock() }
}

func (m *Monitor) Check() error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, w := range m.active {
		if m.now().Sub(w.since) > WorkTimeout {
			return fmt.Errorf("bridge work stalled: %s", w.name)
		}
	}
	return nil
}

func (m *Monitor) Report(err error) {
	if m == nil || !IsPermanent(err) {
		return
	}
	select {
	case m.failure <- err:
	default:
	}
}

func (m *Monitor) Failures() <-chan error { return m.failure }
