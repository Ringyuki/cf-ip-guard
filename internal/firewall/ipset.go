package firewall

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/Ringyuki/cf-ip-guard/internal/logging"
	"go.uber.org/zap"
)

type Runner interface {
	Run(ctx context.Context, name string, args ...string) error
}

type execRunner struct{}

func (r *execRunner) Run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %v failed: %w (output: %s)", name, args, err, string(out))
	}
	return nil
}

var (
	runner Runner             = &execRunner{}
	logger *zap.SugaredLogger = logging.L().Named("firewall")
)

type UpdateConfig struct {
	IPv4CIDRs   []string
	IPv6CIDRs   []string
	IPv4SetName string
	IPv6SetName string
}

func SetLogger(l *zap.SugaredLogger) {
	if l != nil {
		logger = l
	}
}

func UpdateIPSets(ctx context.Context, cfg UpdateConfig) error {
	v4set := cfg.IPv4SetName
	v6set := cfg.IPv6SetName

	tmp4 := v4set + "_tmp"
	tmp6 := v6set + "_tmp"

	// IPv4
	if err := runner.Run(ctx, "ipset", "create", tmp4, "hash:net", "-exist"); err != nil {
		return err
	}
	if err := runner.Run(ctx, "ipset", "flush", tmp4); err != nil {
		return err
	}
	for _, cidr := range cfg.IPv4CIDRs {
		if err := runner.Run(ctx, "ipset", "add", tmp4, cidr, "-exist"); err != nil {
			return err
		}
	}

	// IPv6
	if err := runner.Run(ctx, "ipset", "create", tmp6, "hash:net", "family", "inet6", "-exist"); err != nil {
		return err
	}
	if err := runner.Run(ctx, "ipset", "flush", tmp6); err != nil {
		return err
	}
	for _, cidr := range cfg.IPv6CIDRs {
		if err := runner.Run(ctx, "ipset", "add", tmp6, cidr, "-exist"); err != nil {
			return err
		}
	}

	if err := runner.Run(ctx, "ipset", "create", v4set, "hash:net", "-exist"); err != nil {
		return err
	}
	if err := runner.Run(ctx, "ipset", "create", v6set, "hash:net", "family", "inet6", "-exist"); err != nil {
		return err
	}

	if err := runner.Run(ctx, "ipset", "swap", v4set, tmp4); err != nil {
		return err
	}
	if err := runner.Run(ctx, "ipset", "swap", v6set, tmp6); err != nil {
		return err
	}

	if err := runner.Run(ctx, "ipset", "destroy", tmp4); err != nil {
		logger.Warnw("destroy tmp set failed", "set", tmp4, "err", err)
	}
	if err := runner.Run(ctx, "ipset", "destroy", tmp6); err != nil {
		logger.Warnw("destroy tmp set failed", "set", tmp6, "err", err)
	}

	return nil
}
