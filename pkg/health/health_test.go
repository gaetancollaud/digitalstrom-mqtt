package health

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestShutdownHTTPServerStartsFreshTimeout(t *testing.T) {
	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	requestDone := make(chan error, 1)
	var releaseOnce sync.Once
	release := func() {
		releaseOnce.Do(func() { close(releaseRequest) })
	}
	t.Cleanup(release)

	server := &http.Server{Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(requestStarted)
		<-releaseRequest
	})}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer server.Close()

	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			requestDone <- err
		}
	}()
	go func() {
		response, err := http.Get("http://" + listener.Addr().String())
		if response != nil {
			response.Body.Close()
		}
		requestDone <- err
	}()

	select {
	case <-requestStarted:
	case <-time.After(time.Second):
		t.Fatal("request did not reach health server")
	}

	timeout := 40 * time.Millisecond
	time.Sleep(2 * timeout)
	shutdownStarted := time.Now()
	err = shutdownHTTPServer(server, timeout)
	elapsed := time.Since(shutdownStarted)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected shutdown timeout, got %v", err)
	}
	if elapsed < timeout/2 {
		t.Fatalf("shutdown reused an expired context: elapsed %s, timeout %s", elapsed, timeout)
	}

	release()
	select {
	case err := <-requestDone:
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("request did not finish")
	}
}
