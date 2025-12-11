package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/Ringyuki/cf-ip-guard/internal/daemon"
	"github.com/Ringyuki/cf-ip-guard/internal/logging"
)

var (
	flagInterval   time.Duration
	flagIPv4Set    string
	flagIPv6Set    string
	flagCloudflare string
	flagOnce       bool
	flagLogLevel   string
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run cf-ip-guard in daemon mode",
	Long:  "Scrape Cloudflare IP ranges from /ips api and update ipset",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		logger, err := logging.Init(flagLogLevel, "", "")
		if err != nil {
			return err
		}

		cfg := daemon.Config{
			Interval:      flagInterval,
			IPv4SetName:   flagIPv4Set,
			IPv6SetName:   flagIPv6Set,
			CloudflareAPI: flagCloudflare,
			Once:          flagOnce,
			Logger:        logger,
		}

		return daemon.Run(ctx, cfg)
	},
}

func init() {
	rootCmd.AddCommand(daemonCmd)

	daemonCmd.Flags().DurationVarP(&flagInterval, "interval", "i", 30*time.Minute,
		"update interval, e.g. 10m, 1h")
	daemonCmd.Flags().StringVar(&flagIPv4Set, "ipset4", "cloudflare4",
		"ipset name for Cloudflare IPv4 ranges")
	daemonCmd.Flags().StringVar(&flagIPv6Set, "ipset6", "cloudflare6",
		"ipset name for Cloudflare IPv6 ranges")
	daemonCmd.Flags().StringVar(&flagCloudflare, "api-url",
		"https://api.cloudflare.com/client/v4/ips",
		"Cloudflare IP ranges API URL")
	daemonCmd.Flags().BoolVar(&flagOnce, "once", false,
		"run only one fetch-and-update cycle and exit")
	daemonCmd.Flags().StringVar(&flagLogLevel, "log-level", "info",
		"log level: debug, info, warn, error")
}
