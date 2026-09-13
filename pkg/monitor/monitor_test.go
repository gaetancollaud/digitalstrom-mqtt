package monitor

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStalledWorkAndIdleHome(t *testing.T) {
	m := New()
	now := time.Now()
	m.now = func() time.Time { return now }
	require.NoError(t, m.Check())
	done := m.Begin("notification callback")
	now = now.Add(WorkTimeout + time.Second)
	require.ErrorContains(t, m.Check(), "notification callback")
	// Unrelated work must not conceal the blocked callback.
	m.Begin("other callback")()
	require.Error(t, m.Check())
	done()
	require.NoError(t, m.Check())
	now = now.Add(24 * time.Hour)
	require.NoError(t, m.Check(), "a quiet home is healthy")
}

func TestOnlyPermanentFailuresRequireUserAction(t *testing.T) {
	m := New()
	m.Report(errors.New("network unavailable"))
	select {
	case <-m.Failures():
		t.Fatal("network failure was classified as permanent")
	default:
	}
	err := fmt.Errorf("startup: %w", Permanent(errors.New("credentials rejected")))
	m.Report(err)
	require.Equal(t, err, <-m.Failures())
	require.Equal(t, 78, ExitCode(err))
	require.Equal(t, 75, ExitCode(errors.New("connection timeout")))
}
