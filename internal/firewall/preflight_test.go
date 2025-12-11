package firewall

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestCheckEnvOK(t *testing.T) {
	fr := &fakeRunner{failAt: -1}
	orig := runner
	runner = fr
	defer func() { runner = orig }()
	SetLogger(zap.NewNop().Sugar())

	if err := CheckEnv(context.Background()); err != nil {
		t.Fatalf("CheckEnv error: %v", err)
	}

	if len(fr.calls) != 1 || fr.calls[0] != "ipset list" {
		t.Fatalf("unexpected calls: %v", fr.calls)
	}
}

func TestCheckEnvError(t *testing.T) {
	fr := &fakeRunner{
		failAt: 0,
		err:    errors.New("boom"),
	}
	orig := runner
	runner = fr
	defer func() { runner = orig }()
	SetLogger(zap.NewNop().Sugar())

	err := CheckEnv(context.Background())
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected wrapped boom error, got %v", err)
	}
}
