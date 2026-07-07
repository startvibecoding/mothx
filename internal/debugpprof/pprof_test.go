package debugpprof

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestListenAddrDefaultsToLocalhost(t *testing.T) {
	t.Setenv(AddrEnv, "")

	if got := listenAddr(); got != DefaultAddr {
		t.Fatalf("listenAddr() = %q, want %q", got, DefaultAddr)
	}
}

func TestListenAddrUsesEnvOverride(t *testing.T) {
	t.Setenv(AddrEnv, "127.0.0.1:0")

	if got := listenAddr(); got != "127.0.0.1:0" {
		t.Fatalf("listenAddr() = %q, want 127.0.0.1:0", got)
	}
}

func TestMuxServesPprofIndex(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	rec := httptest.NewRecorder()

	newMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestStartServesPprof(t *testing.T) {
	t.Setenv(AddrEnv, "127.0.0.1:0")

	addr, _, err := Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	client := &http.Client{Timeout: time.Second}
	resp, err := client.Get("http://" + addr + "/debug/pprof/")
	if err != nil {
		t.Fatalf("GET pprof index: %v", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}
