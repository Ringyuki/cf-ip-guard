package firewall

import (
	"context"
	"fmt"
)

func CheckEnv(ctx context.Context) error {
	if err := runner.Run(ctx, "ipset", "list"); err != nil {
		return fmt.Errorf("ipset not available or permission denied: %w", err)
	}
	return nil
}
