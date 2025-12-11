package cloudflare

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchIPsSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"success": true,
			"result": {
				"ipv4_cidrs": ["1.1.1.0/24"],
				"ipv6_cidrs": ["2606:4700::/32"],
				"etag": "etag-123"
			}
		}`))
	}))
	defer ts.Close()

	c := &Client{APIURL: ts.URL}
	v4, v6, etag, notModified, err := c.FetchIPs(context.Background(), "")
	if err != nil {
		t.Fatalf("FetchIPs error: %v", err)
	}
	if notModified {
		t.Fatalf("expected notModified=false")
	}
	if etag != "etag-123" {
		t.Fatalf("unexpected etag: %s", etag)
	}
	if len(v4) != 1 || v4[0] != "1.1.1.0/24" {
		t.Fatalf("unexpected ipv4 cidrs: %v", v4)
	}
	if len(v6) != 1 || v6[0] != "2606:4700::/32" {
		t.Fatalf("unexpected ipv6 cidrs: %v", v6)
	}
}

func TestFetchIPsNotModified(t *testing.T) {
	const prevETag = "etag-prev"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("If-None-Match"); got != prevETag {
			t.Fatalf("expected If-None-Match header %q, got %q", prevETag, got)
		}
		w.WriteHeader(http.StatusNotModified)
	}))
	defer ts.Close()

	c := &Client{APIURL: ts.URL}
	_, _, etag, notModified, err := c.FetchIPs(context.Background(), prevETag)
	if err != nil {
		t.Fatalf("FetchIPs error: %v", err)
	}
	if !notModified {
		t.Fatalf("expected notModified=true")
	}
	if etag != prevETag {
		t.Fatalf("etag should carry prev etag, got %s", etag)
	}
}

func TestFetchIPsNon200(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer ts.Close()

	c := &Client{APIURL: ts.URL}
	_, _, _, _, err := c.FetchIPs(context.Background(), "")
	if err == nil {
		t.Fatalf("expected error on non-200")
	}
}
