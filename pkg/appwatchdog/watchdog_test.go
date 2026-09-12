package appwatchdog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type supervisorFixture struct {
	on     bool
	fail   bool
	writes []bool
}

func fixture(t *testing.T) (*Watchdog, *supervisorFixture) {
	t.Helper()
	f := &supervisorFixture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/info" {
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "ok", "data": map[string]bool{"watchdog": f.on}})
			return
		}
		if r.Method != http.MethodPost || r.URL.Path != "/options" {
			w.WriteHeader(404)
			return
		}
		if f.fail {
			w.WriteHeader(503)
			return
		}
		var body map[string]bool
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body) != 1 {
			w.WriteHeader(400)
			return
		}
		var ok bool
		f.on, ok = body["watchdog"]
		if !ok {
			w.WriteHeader(400)
			return
		}
		f.writes = append(f.writes, f.on)
		_, _ = w.Write([]byte(`{"result":"ok","data":{}}`))
	}))
	t.Cleanup(server.Close)
	return &Watchdog{URL: server.URL, Token: "test-token", Path: filepath.Join(t.TempDir(), "state.json"), Client: server.Client()}, f
}

func TestFirstSuccessfulStartAndManualDisable(t *testing.T) {
	w, server := fixture(t)
	ctx := context.Background()
	require.NoError(t, w.Pause(ctx))
	require.Empty(t, server.writes, "first start must not arm before initialization")
	require.NoError(t, w.Arm(ctx))
	require.True(t, server.on)
	require.NoError(t, w.Pause(ctx))
	require.False(t, server.on)
	// Reconstruct the manager like a fresh process after a paused restart.
	restarted := *w
	require.NoError(t, restarted.Arm(ctx))
	require.True(t, server.on)
	server.on = false // User explicitly disables it in Home Assistant.
	require.NoError(t, restarted.Arm(ctx))
	require.NoError(t, restarted.Pause(ctx))
	require.NoError(t, restarted.Arm(ctx))
	require.False(t, server.on, "must preserve manual disable across retries and restarts")
	require.Equal(t, []bool{true, false, true}, server.writes)
}

func TestFailedPausePreservesResumeIntent(t *testing.T) {
	w, server := fixture(t)
	ctx := context.Background()
	require.NoError(t, w.Arm(ctx))
	server.fail = true
	require.Error(t, w.Pause(ctx))
	s, err := w.read()
	require.NoError(t, err)
	require.True(t, s.Resume)
	server.fail = false
	require.NoError(t, w.Pause(ctx))
	require.False(t, server.on)
	require.NoError(t, w.Arm(ctx))
	require.True(t, server.on)
}

func TestFailedActivationIsRetried(t *testing.T) {
	w, server := fixture(t)
	server.fail = true
	require.Error(t, w.Arm(context.Background()))
	s, err := w.read()
	require.NoError(t, err)
	require.False(t, s.Initialized)
	server.fail = false
	require.NoError(t, w.Arm(context.Background()))
	require.True(t, server.on)
}

func TestInvalidSupervisorResponseDoesNotAuthorizeStartup(t *testing.T) {
	for _, body := range []string{`{"result":"ok","data":{}}`, `{"result":"error"}`, `not json`} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body)) }))
			defer server.Close()
			w := &Watchdog{URL: server.URL, Token: "test-token", Path: filepath.Join(t.TempDir(), "state"), Client: server.Client()}
			require.Error(t, w.Pause(context.Background()))
		})
	}
}
