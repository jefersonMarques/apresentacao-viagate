package templates

import "strings"

func AdminStatusBadgeClass(status string) string {
	switch strings.TrimSpace(status) {
	case "accepted", "approved", "signed", "completed", "activated", "active":
		return "status-success"
	case "expired", "cancelled", "disabled", "correction_requested", "failed":
		return "status-danger"
	case "published", "sent", "submitted", "under_review", "under_internal_setup", "in_progress", "invited":
		return "status-progress"
	default:
		return "status-neutral"
	}
}

func AdminRoleLabel(role string) string {
	switch strings.TrimSpace(role) {
	case "super_admin":
		return "Superadmin"
	case "admin":
		return "Admin"
	case "user":
		return "Usuário"
	case "commercial":
		return "Usuário · legado comercial"
	case "operations":
		return "Usuário · legado operações"
	case "legal":
		return "Usuário · legado jurídico"
	default:
		return role
	}
}

func AdminListSingular(noun string) string {
	return strings.NewReplacer("(ões)", "", "(s)", "").Replace(strings.TrimSpace(noun))
}

func AdminListPlural(noun string) string {
	return strings.NewReplacer("(ões)", "ões", "(s)", "s").Replace(strings.TrimSpace(noun))
}

func AdminListNoun(total int, noun string) string {
	if total == 1 {
		return AdminListSingular(noun)
	}
	return AdminListPlural(noun)
}
