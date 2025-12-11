package daemon

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Ringyuki/cf-ip-guard/internal/cloudflare"
	"github.com/Ringyuki/cf-ip-guard/internal/firewall"
	"go.uber.org/zap"
)

func TestUpdateOnceSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"success": true,
			"result": {
				"ipv4_cidrs": ["1.1.1.0/24", "1.1.2.0/24"],
				"ipv6_cidrs": ["2606:4700::/32"],
				"etag": "etag-123"
			}
		}`))
	}))
	defer ts.Close()

	called := false
	var gotCfg firewall.UpdateConfig
	orig := updateIPSetsFunc
	updateIPSetsFunc = func(ctx context.Context, cfg firewall.UpdateConfig) error {
		called = true
		gotCfg = cfg
		return nil
	}
	defer func() { updateIPSetsFunc = orig }()

	client := &cloudflare.Client{APIURL: ts.URL}
	cfg := Config{
		IPv4SetName: "v4",
		IPv6SetName: "v6",
		Logger:      zap.NewNop().Sugar(),
	}

	res, err := updateOnce(context.Background(), cfg.Logger, client, cfg, "")
	if err != nil {
		t.Fatalf("updateOnce error: %v", err)
	}
	if res.NotModified {
		t.Fatalf("expected NotModified=false")
	}
	if res.ETag != "etag-123" {
		t.Fatalf("unexpected etag: %s", res.ETag)
	}
	if res.IPv4Count != 2 || res.IPv6Count != 1 {
		t.Fatalf("unexpected counts: v4=%d v6=%d", res.IPv4Count, res.IPv6Count)
	}
	if !called {
		t.Fatalf("expected updateIPSetsFunc to be called")
	}
	if gotCfg.IPv4SetName != "v4" || gotCfg.IPv6SetName != "v6" {
		t.Fatalf("unexpected set names: %+v", gotCfg)
	}
	if len(gotCfg.IPv4CIDRs) != 2 || len(gotCfg.IPv6CIDRs) != 1 {
		t.Fatalf("unexpected cidrs: %+v", gotCfg)
	}
	if res.Duration <= 0 || res.Duration > time.Second {
		t.Fatalf("unexpected duration: %s", res.Duration)
	}
}

func TestUpdateOnceNotModified(t *testing.T) {
	const prevETag = "etag-prev"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("If-None-Match"); got != prevETag {
			t.Fatalf("expected If-None-Match %q, got %q", prevETag, got)
		}
		w.WriteHeader(http.StatusNotModified)
	}))
	defer ts.Close()

	called := false
	orig := updateIPSetsFunc
	updateIPSetsFunc = func(ctx context.Context, cfg firewall.UpdateConfig) error {
		called = true
		return nil
	}
	defer func() { updateIPSetsFunc = orig }()

	client := &cloudflare.Client{APIURL: ts.URL}
	cfg := Config{
		IPv4SetName: "v4",
		IPv6SetName: "v6",
		Logger:      zap.NewNop().Sugar(),
	}

	res, err := updateOnce(context.Background(), cfg.Logger, client, cfg, prevETag)
	if err != nil {
		t.Fatalf("updateOnce error: %v", err)
	}
	if !res.NotModified {
		t.Fatalf("expected NotModified=true")
	}
	if res.ETag != prevETag {
		t.Fatalf("etag should carry prev: %s", res.ETag)
	}
	if called {
		t.Fatalf("updateIPSetsFunc should not be called on 304")
	}
}
