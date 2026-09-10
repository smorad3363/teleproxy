package forcedjoin

import (
	"context"
	"database/sql"
	"fmt"
)

type MembershipClient interface {
	IsChatMember(context.Context, string, int64) (bool, error)
}

type CheckResult struct {
	Missing []Channel
}

func CheckRequired(ctx context.Context, db *sql.DB, client MembershipClient, telegramID int64) (CheckResult, error) {
	if telegramID <= 0 {
		return CheckResult{}, fmt.Errorf("Telegram ID must be greater than zero")
	}
	if client == nil {
		return CheckResult{}, fmt.Errorf("forced join membership client is required")
	}
	channels, err := ListRequired(ctx, db)
	if err != nil {
		return CheckResult{}, err
	}
	result := CheckResult{}
	for _, channel := range channels {
		member, err := client.IsChatMember(ctx, channel.ChatRef, telegramID)
		if err != nil {
			return CheckResult{}, fmt.Errorf("verify forced join membership: %w", err)
		}
		if !member {
			result.Missing = append(result.Missing, channel)
		}
	}
	return result, nil
}
