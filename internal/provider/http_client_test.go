package provider

import (
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestNewHTTPClientDefaultProxy(t *testing.T) {
	client, err := NewHTTPClient(time.Second, "")
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport = %T, want *http.Transport", client.Transport)
	}
	if transport.Proxy == nil {
		t.Fatal("expected default environment proxy function")
	}
}

func TestNewHTTPClientExplicitProxy(t *testing.T) {
	client, err := NewHTTPClient(time.Second, " http://127.0.0.1:7890 ")
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport = %T, want *http.Transport", client.Transport)
	}
	proxyURL, err := transport.Proxy(&http.Request{URL: &url.URL{Scheme: "https", Host: "api.test"}})
	if err != nil {
		t.Fatalf("proxy lookup: %v", err)
	}
	if proxyURL == nil || proxyURL.String() != "http://127.0.0.1:7890" {
		t.Fatalf("proxy = %v, want http://127.0.0.1:7890", proxyURL)
	}
}

func TestNewHTTPClientRejectsInvalidProxy(t *testing.T) {
	for _, proxyURL := range []string{"http://[::1", "127.0.0.1:7890", "http://"} {
		if _, err := NewHTTPClient(time.Second, proxyURL); err == nil {
			t.Fatalf("expected error for proxy URL %q", proxyURL)
		}
	}
}
