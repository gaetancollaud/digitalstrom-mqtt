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
	on           bool
	fail         bool
	writes       []bool
	build        bool
	autoUpdate   bool
	updateWrites []bool
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
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "ok", "data": map[string]bool{"watchdog": f.on, "build": f.build}})
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
		if enabled, ok := body["watchdog"]; ok {
			f.on = enabled
			f.writes = append(f.writes, f.on)
		} else if enabled, ok := body["auto_update"]; ok {
			f.autoUpdate = enabled
			f.updateWrites = append(f.updateWrites, enabled)
		} else {
			w.WriteHeader(400)
			return
		}
		_, _ = w.Write([]byte(`{"result":"ok","data":{}}`))
	}))
	t.Cleanup(server.Close)
	return &Watchdog{URL: server.URL, Token: "test-token", Path: filepath.Join(t.TempDir(), "state.json"), Client: server.Client()}, f
}

func TestAutoUpdatesPreserveManualDisableAcrossContainerReplacement(t *testing.T) {
	w, server := fixture(t)
	ctx := context.Background()
	require.NoError(t, w.Pause(ctx))
	require.Empty(t, server.updateWrites, "setup must not enable updates")
	require.NoError(t, w.Arm(ctx))
	require.NoError(t, w.EnableAutoUpdatesOnce(ctx))
	require.True(t, server.autoUpdate)
	server.autoUpdate = false
	server.on = false
	// A replacement process after an update has no state except the /data file.
	replacement := &Watchdog{URL: w.URL, Token: w.Token, Path: w.Path, Client: w.Client}
	require.NoError(t, replacement.Pause(ctx))
	require.NoError(t, replacement.Arm(ctx))
	require.NoError(t, replacement.EnableAutoUpdatesOnce(ctx))
	require.False(t, server.autoUpdate)
	require.False(t, server.on)
	require.Equal(t, []bool{true}, server.updateWrites)
	require.Equal(t, []bool{true}, server.writes)
}

func TestLocallyBuiltAppDoesNotEnableAutoUpdates(t *testing.T) {
	w, server := fixture(t)
	server.build = true
	require.NoError(t, w.Arm(context.Background()))
	require.NoError(t, w.EnableAutoUpdatesOnce(context.Background()))
	require.True(t, server.on, "local tests still exercise watchdog recovery")
	require.False(t, server.autoUpdate)
	require.Empty(t, server.updateWrites)
}

func TestAutoUpdateFailureIsRetriedWithoutRearmingManuallyDisabledWatchdog(t *testing.T) {
	w, server := fixture(t)
	ctx := context.Background()
	require.NoError(t, w.Arm(ctx))
	server.on = false
	server.fail = true
	require.Error(t, w.EnableAutoUpdatesOnce(ctx))
	s, err := w.read()
	require.NoError(t, err)
	require.False(t, s.UpdatesInitialized)
	server.fail = false
	require.NoError(t, w.Arm(ctx))
	require.NoError(t, w.EnableAutoUpdatesOnce(ctx))
	require.False(t, server.on)
	require.True(t, server.autoUpdate)
	// Watchdog writes must preserve the separate update-initialization marker.
	server.on = true
	server.autoUpdate = false
	require.NoError(t, w.Pause(ctx))
	require.NoError(t, w.Arm(ctx))
	require.NoError(t, w.EnableAutoUpdatesOnce(ctx))
	require.False(t, server.autoUpdate)
}

func TestMissingBuildTypeDoesNotEnableAutoUpdates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Error("must not write without knowing the App build type")
		}
		_, _ = w.Write([]byte(`{"result":"ok","data":{"watchdog":false}}`))
	}))
	defer server.Close()
	w := &Watchdog{URL: server.URL, Token: "test-token", Path: filepath.Join(t.TempDir(), "state"), Client: server.Client()}
	require.Error(t, w.EnableAutoUpdatesOnce(context.Background()))
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

func TestCrashRestoresOnlyPausedProtection(t *testing.T) {
	for _, test := range []struct {
		name        string
		initialized bool
		enabled     bool
		pause       bool
	}{
		{"first start", false, false, true},
		{"first start with user enabled protection", false, true, true},
		{"restart with protection", true, true, true},
		{"restart after manual disable", true, false, true},
		{"runtime crash with protection", true, true, false},
		{"runtime crash after manual disable", true, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			w, server := fixture(t)
			ctx := context.Background()
			require.NoError(t, w.save(state{Initialized: test.initialized, UpdatesInitialized: true}))
			server.on = test.enabled
			if test.pause {
				require.NoError(t, w.Pause(ctx))
				require.False(t, server.on)
			}
			// The crash handler runs as a separate process, sharing only /data.
			restarted := &Watchdog{URL: w.URL, Token: w.Token, Path: w.Path, Client: w.Client}
			require.NoError(t, restarted.ResumeAfterCrash(ctx))
			require.Equal(t, test.enabled, server.on)
			s, err := restarted.read()
			require.NoError(t, err)
			require.False(t, s.Resume)
			require.Equal(t, test.initialized, s.Initialized, "a crash is not a successful startup")
			require.True(t, s.UpdatesInitialized, "crash recovery must preserve update preferences")
			if test.enabled && test.pause {
				require.Equal(t, []bool{false, true}, server.writes)
			} else {
				require.Empty(t, server.writes)
			}
		})
	}
}

func TestCrashRestoreFailureRetainsIntentForRetry(t *testing.T) {
	w, server := fixture(t)
	ctx := context.Background()
	require.NoError(t, w.Arm(ctx))
	require.NoError(t, w.Pause(ctx))
	server.fail = true
	require.Error(t, w.ResumeAfterCrash(ctx))
	s, err := w.read()
	require.NoError(t, err)
	require.True(t, s.Resume)
	require.False(t, server.on)
	server.fail = false
	require.NoError(t, w.ResumeAfterCrash(ctx))
	require.True(t, server.on)
	server.on = false
	require.NoError(t, w.ResumeAfterCrash(ctx))
	require.False(t, server.on, "completed restore must not undo a later manual disable")
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
