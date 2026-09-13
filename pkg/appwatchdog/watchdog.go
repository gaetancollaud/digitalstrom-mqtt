// Package appwatchdog manages this App's Supervisor startup defaults.
package appwatchdog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type state struct {
	Initialized        bool `json:"initialized"`
	Resume             bool `json:"resume"`
	UpdatesInitialized bool `json:"updates_initialized,omitempty"`
}

type supervisorInfo struct {
	Watchdog *bool `json:"watchdog"`
	Build    *bool `json:"build"`
}

type Watchdog struct {
	URL    string
	Token  string
	Path   string
	Client *http.Client
}

func FromEnvironment() *Watchdog {
	return &Watchdog{URL: "http://supervisor/addons/self", Token: os.Getenv("SUPERVISOR_TOKEN"),
		Path: "/data/watchdog-state.json", Client: &http.Client{Timeout: 10 * time.Second}}
}

func (w *Watchdog) read() (state, error) {
	var s state
	b, err := os.ReadFile(w.Path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	err = json.Unmarshal(b, &s)
	return s, err
}

func (w *Watchdog) save(s state) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(w.Path), ".watchdog-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(b)
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), w.Path)
}

func (w *Watchdog) request(ctx context.Context, method, endpoint string, body []byte) (supervisorInfo, error) {
	if w.Token == "" {
		return supervisorInfo{}, errors.New("Supervisor token is missing")
	}
	req, err := http.NewRequestWithContext(ctx, method, w.URL+endpoint, bytes.NewReader(body))
	if err != nil {
		return supervisorInfo{}, err
	}
	req.Header.Set("Authorization", "Bearer "+w.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := w.Client.Do(req)
	if err != nil {
		return supervisorInfo{}, errors.New("Supervisor App settings request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return supervisorInfo{}, fmt.Errorf("Supervisor App settings request returned HTTP %d", resp.StatusCode)
	}
	var result struct {
		Result string         `json:"result"`
		Data   supervisorInfo `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return supervisorInfo{}, errors.New("invalid Supervisor App settings response")
	}
	if result.Result != "ok" {
		return supervisorInfo{}, errors.New("Supervisor rejected App settings request")
	}
	return result.Data, nil
}

// Pause records our intent before changing Supervisor, so an interrupted start
// can resume protection. A user's later manual disable is preserved.
func (w *Watchdog) Pause(ctx context.Context) error {
	s, err := w.read()
	if err != nil {
		return err
	}
	info, err := w.request(ctx, http.MethodGet, "/info", nil)
	if err != nil {
		return err
	}
	if info.Watchdog == nil {
		return errors.New("Supervisor response is missing watchdog state")
	}
	if !*info.Watchdog {
		return nil
	}
	s.Resume = true
	if err := w.save(s); err != nil {
		return err
	}
	_, err = w.request(ctx, http.MethodPost, "/options", []byte(`{"watchdog":false}`))
	return err
}

// ResumeAfterCrash restores only a recorded pause, without treating a failed
// startup as successful initialization or enabling a manually disabled watchdog.
func (w *Watchdog) ResumeAfterCrash(ctx context.Context) error {
	s, err := w.read()
	if err != nil || !s.Resume {
		return err
	}
	if _, err := w.request(ctx, http.MethodPost, "/options", []byte(`{"watchdog":true}`)); err != nil {
		return err
	}
	s.Resume = false
	return w.save(s)
}

// Arm is called only after the entire controller has started successfully.
func (w *Watchdog) Arm(ctx context.Context) error {
	s, err := w.read()
	if err != nil {
		return err
	}
	if s.Initialized && !s.Resume {
		return nil
	}
	if _, err = w.request(ctx, http.MethodPost, "/options", []byte(`{"watchdog":true}`)); err != nil {
		return err
	}
	s.Initialized, s.Resume = true, false
	return w.save(s)
}

// EnableAutoUpdatesOnce runs only after complete startup. The marker lives in
// /data, so replacing the container does not override a later manual disable.
// Locally built Apps (including the PR test copy) never opt in automatically.
func (w *Watchdog) EnableAutoUpdatesOnce(ctx context.Context) error {
	s, err := w.read()
	if err != nil || s.UpdatesInitialized {
		return err
	}
	info, err := w.request(ctx, http.MethodGet, "/info", nil)
	if err != nil {
		return err
	}
	if info.Build == nil {
		return errors.New("Supervisor response is missing App build type")
	}
	if *info.Build {
		return nil
	}
	if _, err := w.request(ctx, http.MethodPost, "/options", []byte(`{"auto_update":true}`)); err != nil {
		return err
	}
	s.UpdatesInitialized = true
	return w.save(s)
}
