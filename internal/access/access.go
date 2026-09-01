package access

import "github.com/jefersonMarques/apresentacao-viagate/internal/domain"

const (
	ProposalCreate         = "proposal.create"
	ProposalReadOwn        = "proposal.read_own"
	ProposalReadAll        = "proposal.read_all"
	ProposalPriceEdit      = "proposal.price.edit"
	ProposalConditionsEdit = "proposal.conditions.edit"
	ProposalPublish        = "proposal.publish"
	ProposalDuplicate      = "proposal.duplicate"
	ProposalCancel         = "proposal.cancel"

	PresentationCreate       = "presentation.create"
	PresentationReadOwn      = "presentation.read_own"
	PresentationReadAll      = "presentation.read_all"
	PresentationPublish      = "presentation.publish"
	PresentationActivityRead = "presentation.activity.read"

	CustomerReadOwn          = "customer.read_own"
	CustomerReadAll          = "customer.read_all"
	CustomerResponsesReadOwn = "customer.responses.read_own"
	CustomerResponsesReadAll = "customer.responses.read_all"
	CustomerSensitiveRead    = "customer.sensitive_data.read"
	CustomerDocumentsRead    = "customer.documents.read"

	ContractReadOwn        = "contract.read_own"
	ContractReadAll        = "contract.read_all"
	ContractEvidenceRead   = "contract.evidence.read"
	ContractTemplateManage = "contract.template.manage"

	OnboardingReview  = "onboarding.review"
	ActivationReadAll = "activation.read_all"
	ActivationManage  = "activation.manage"

	UserManage            = "user.manage"
	UserPermissionsManage = "user.permissions.manage"
	UserStatusManage      = "user.status.manage"

	ActivityReadOwn    = "activity.read_own"
	ActivityReadAll    = "activity.read_all"
	AuditRead          = "audit.read"
	AuditTechnicalRead = "audit.technical.read"

	NotificationRead          = "notification.read"
	NotificationReceiveOthers = "notification.receive_others"
	SettingsManage            = "settings.manage"
	SystemTechnical           = "system.technical"
)

func HasRole(user domain.User, role string) bool {
	for _, current := range user.Roles {
		if current == role {
			return true
		}
	}
	return false
}

func Can(user domain.User, permission string) bool {
	for _, current := range user.Permissions {
		if current == permission {
			return true
		}
	}
	return false
}

func IsSuperAdmin(user domain.User) bool { return HasRole(user, "super_admin") }
func IsAdmin(user domain.User) bool      { return HasRole(user, "admin") }
func IsUser(user domain.User) bool       { return HasRole(user, "user") }

// CanManageAccount prevents privilege escalation. Admins may manage User
// accounts; only Superadmins may manage Admin/Superadmin accounts or promote
// an account to an elevated profile.
func CanManageAccount(actor, target domain.User) bool {
	if actor.ID == target.ID {
		return false
	}
	if IsSuperAdmin(actor) {
		return true
	}
	if !IsAdmin(actor) {
		return false
	}
	return !IsAdmin(target) && !IsSuperAdmin(target)
}

func CanAssignRole(actor domain.User, role string) bool {
	if IsSuperAdmin(actor) {
		return role == "user" || role == "admin" || role == "super_admin"
	}
	return IsAdmin(actor) && role == "user"
}

func CanRemoveActiveSuperAdmin(activeSuperAdmins int) bool {
	return activeSuperAdmins > 1
}
