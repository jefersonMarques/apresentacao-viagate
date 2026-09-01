package templates

import (
	"fmt"

	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
)

func NotificationCompactClass(compact bool) string {
	if compact {
		return " is-compact"
	}
	return ""
}

func NotificationUnreadLabel(items []domain.InAppNotification) string {
	count := 0
	for _, item := range items {
		if item.ReadAt == nil {
			count++
		}
	}
	if count == 0 {
		return "Tudo lido"
	}
	if count == 1 {
		return "1 nova"
	}
	return fmt.Sprintf("%d novas", count)
}
