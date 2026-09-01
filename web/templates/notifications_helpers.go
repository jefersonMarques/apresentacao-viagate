package templates

import (
	"strconv"

	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
)

func NotificationRowClass(item domain.InAppNotification) string {
	if item.ReadAt == nil {
		return "admin-notification-row is-unread"
	}
	return "admin-notification-row"
}

func NotificationCount(total int) string {
	if total > 99 {
		return "99+"
	}
	return strconv.Itoa(total)
}
