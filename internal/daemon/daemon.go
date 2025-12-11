package daemon

import (
	"context"
	"net/http"
	"time"

	"github.com/Ringyuki/cf-ip-guard/internal/cloudflare"
	"github.com/Ringyuki/cf-ip-guard/internal/firewall"
	"github.com/Ringyuki/cf-ip-guard/internal/logging"
)

var updateIPSetsFunc = firewall.UpdateIPSets

type Config struct {
	Interval      time.Duration
	IPv4SetName   string
	IPv6SetName   string
	CloudflareAPI string
	Once          bool
	Logger        logging.Logger
}

type updateStats struct {
	Success         uint64
	Fail            uint64
	ConsecutiveFail uint64
	LastDuration    time.Duration
	LastETag        string
	LastUpdate      time.Time
}

func Run(ctx context.Context, cfg Config) error {
	if cfg.Logger == nil {
		cfg.Logger = logging.L().Named("daemon")
	}
	logger := cfg.Logger

	if cfg.Interval <= 0 {
		cfg.Interval = 30 * time.Minute
	}
	if cfg.IPv4SetName == "" {
		cfg.IPv4SetName = "cloudflare4"
	}
	if cfg.IPv6SetName == "" {
		cfg.IPv6SetName = "cloudflare6"
	}
	if cfg.CloudflareAPI == "" {
		cfg.CloudflareAPI = "https://api.cloudflare.com/client/v4/ips"
	}

	if err := firewall.CheckEnv(ctx); err != nil {
		logger.Errorw("preflight check failed", "err", err)
		return err
	}
	firewall.SetLogger(logger.Named("firewall"))

	client := &cloudflare.Client{
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		APIURL:     cfg.CloudflareAPI,
	}

	logger.Infow("cf-ip-guard daemon starting",
		"interval", cfg.Interval,
		"ipset4", cfg.IPv4SetName,
		"ipset6", cfg.IPv6SetName,
		"api", cfg.CloudflareAPI,
		"once", cfg.Once)

	stats := &updateStats{}
	var lastETag string

	if res, err := updateOnce(ctx, logger, client, cfg, lastETag); err != nil {
		markFailure(stats, logger, err)
		logger.Errorw("initial update failed", "err", err)
	} else {
		markSuccess(stats, logger, res)
		if !res.NotModified && res.ETag != "" {
			lastETag = res.ETag
		}
	}

	if cfg.Once {
		logger.Infow("daemon once mode finished",
			"success", stats.Success,
			"fail", stats.Fail,
			"last_etag", stats.LastETag,
			"consecutive_fail", stats.ConsecutiveFail)
		return nil
	}

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Infow("daemon stopped", "err", ctx.Err())
			return ctx.Err()
		case <-ticker.C:
			if res, err := updateOnce(ctx, logger, client, cfg, lastETag); err != nil {
				markFailure(stats, logger, err)
				logger.Errorw("update failed", "err", err)
			} else {
				markSuccess(stats, logger, res)
				if !res.NotModified && res.ETag != "" {
					lastETag = res.ETag
				}
			}
			logger.Infow("stats",
				"success", stats.Success,
				"fail", stats.Fail,
				"last_duration", stats.LastDuration,
				"last_etag", stats.LastETag,
				"last_update", stats.LastUpdate.Format(time.RFC3339),
				"consecutive_fail", stats.ConsecutiveFail)
		}
	}
}

type updateResult struct {
	IPv4Count   int
	IPv6Count   int
	ETag        string
	Duration    time.Duration
	NotModified bool
}

func updateOnce(ctx context.Context, logger logging.Logger, client *cloudflare.Client, cfg Config, prevETag string) (updateResult, error) {
	start := time.Now()

	ipv4, ipv6, etag, notModified, err := client.FetchIPs(ctx, prevETag)
	if err != nil {
		return updateResult{}, err
	}

	if notModified {
		return updateResult{
			ETag:        etag,
			NotModified: true,
			Duration:    time.Since(start),
		}, nil
	}

	logger.Infow("fetched Cloudflare IPs", "ipv4", len(ipv4), "ipv6", len(ipv6), "etag", etag)

	fwCfg := firewall.UpdateConfig{
		IPv4CIDRs:   ipv4,
		IPv6CIDRs:   ipv6,
		IPv4SetName: cfg.IPv4SetName,
		IPv6SetName: cfg.IPv6SetName,
	}

	if err := updateIPSetsFunc(ctx, fwCfg); err != nil {
		return updateResult{}, err
	}

	return updateResult{
		IPv4Count: len(ipv4),
		IPv6Count: len(ipv6),
		ETag:      etag,
		Duration:  time.Since(start),
	}, nil
}

func markSuccess(stats *updateStats, logger logging.Logger, res updateResult) {
	stats.LastDuration = res.Duration
	if res.ETag != "" {
		stats.LastETag = res.ETag
	}
	stats.LastUpdate = time.Now()
	stats.Success++
	stats.ConsecutiveFail = 0

	if res.NotModified {
		logger.Infow("ipsets unchanged", "etag", res.ETag, "duration", res.Duration)
	} else {
		logger.Infow("ipsets updated successfully",
			"ipv4", res.IPv4Count,
			"ipv6", res.IPv6Count,
			"etag", res.ETag,
			"duration", res.Duration)
	}
}

func markFailure(stats *updateStats, logger logging.Logger, err error) {
	stats.Fail++
	stats.ConsecutiveFail++
	logger.Warnw("ipset update failed",
		"consecutive_fail", stats.ConsecutiveFail,
		"err", err)
}
