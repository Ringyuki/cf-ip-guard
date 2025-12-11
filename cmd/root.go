package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cf-ip-guard",
	Short: "Keep ipsets in sync with Cloudflare IP ranges",
	Long:  "Scrape Cloudflare IP ranges from /ips api and update ipset",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {}

func init() {
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
}
