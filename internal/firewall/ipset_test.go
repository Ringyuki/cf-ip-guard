package firewall

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.uber.org/zap"
)

type fakeRunner struct {
	calls  []string
	failAt int
	err    error
}

func (f *fakeRunner) Run(ctx context.Context, name string, args ...string) error {
	call := name + " " + strings.Join(args, " ")
	if f.failAt >= 0 && len(f.calls) == f.failAt {
		return f.err
	}
	f.calls = append(f.calls, call)
	return nil
}

func TestUpdateIPSetsSuccess(t *testing.T) {
	fr := &fakeRunner{failAt: -1}
	orig := runner
	runner = fr
	defer func() { runner = orig }()
	SetLogger(zap.NewNop().Sugar())

	cfg := UpdateConfig{
		IPv4CIDRs:   []string{"1.1.1.0/24", "1.1.2.0/24"},
		IPv6CIDRs:   []string{"2606:4700::/32"},
		IPv4SetName: "v4",
		IPv6SetName: "v6",
	}

	if err := UpdateIPSets(context.Background(), cfg); err != nil {
		t.Fatalf("UpdateIPSets error: %v", err)
	}

	expected := []string{
		"ipset create v4_tmp hash:net -exist",
		"ipset flush v4_tmp",
		"ipset add v4_tmp 1.1.1.0/24 -exist",
		"ipset add v4_tmp 1.1.2.0/24 -exist",
		"ipset create v6_tmp hash:net family inet6 -exist",
		"ipset flush v6_tmp",
		"ipset add v6_tmp 2606:4700::/32 -exist",
		"ipset create v4 hash:net -exist",
		"ipset create v6 hash:net family inet6 -exist",
		"ipset swap v4 v4_tmp",
		"ipset swap v6 v6_tmp",
		"ipset destroy v4_tmp",
		"ipset destroy v6_tmp",
	}

	if len(fr.calls) != len(expected) {
		t.Fatalf("unexpected call count: got %d want %d", len(fr.calls), len(expected))
	}
	for i, c := range expected {
		if fr.calls[i] != c {
			t.Fatalf("call %d mismatch: got %q want %q", i, fr.calls[i], c)
		}
	}
}

func TestUpdateIPSetsStopsOnError(t *testing.T) {
	fr := &fakeRunner{
		failAt: 2, // fail on third command
		err:    errors.New("boom"),
	}
	orig := runner
	runner = fr
	defer func() { runner = orig }()
	SetLogger(zap.NewNop().Sugar())

	cfg := UpdateConfig{
		IPv4CIDRs:   []string{"1.1.1.0/24"},
		IPv6CIDRs:   []string{},
		IPv4SetName: "v4",
		IPv6SetName: "v6",
	}

	err := UpdateIPSets(context.Background(), cfg)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected boom error, got %v", err)
	}

	// Should stop at the failing command (no further calls)
	if len(fr.calls) != fr.failAt {
		t.Fatalf("expected %d calls before failure, got %d: %v", fr.failAt, len(fr.calls), fr.calls)
	}
}
