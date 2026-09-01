package templates

import "github.com/jefersonMarques/apresentacao-viagate/internal/domain"

func NotificationRowClass(item domain.InAppNotification) string {
	if item.ReadAt == nil {
		return "admin-notification-row is-unread"
	}
	return "admin-notification-row"
}
