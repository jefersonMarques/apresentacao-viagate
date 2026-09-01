package access

import (
	"testing"

	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
)

func TestAdminCannotManageElevatedAccounts(t *testing.T) {
	admin := domain.User{ID: "admin-1", Roles: []string{"admin"}}
	user := domain.User{ID: "user-1", Roles: []string{"user"}}
	otherAdmin := domain.User{ID: "admin-2", Roles: []string{"admin"}}
	superAdmin := domain.User{ID: "super-1", Roles: []string{"super_admin"}}

	if !CanManageAccount(admin, user) {
		t.Fatal("admin should manage regular users")
	}
	if CanManageAccount(admin, otherAdmin) {
		t.Fatal("admin must not manage another admin")
	}
	if CanManageAccount(admin, superAdmin) {
		t.Fatal("admin must not manage a superadmin")
	}
	if CanManageAccount(admin, admin) {
		t.Fatal("actor must not manage their own account through admin controls")
	}
}

func TestAdminCannotAssignElevatedRole(t *testing.T) {
	admin := domain.User{ID: "admin-1", Roles: []string{"admin"}}
	if !CanAssignRole(admin, "user") {
		t.Fatal("admin should be able to assign user role")
	}
	if CanAssignRole(admin, "admin") {
		t.Fatal("admin must not promote a user to admin")
	}
	if CanAssignRole(admin, "super_admin") {
		t.Fatal("admin must not promote a user to superadmin")
	}
}

func TestSuperAdminCanAssignManagedRoles(t *testing.T) {
	superAdmin := domain.User{ID: "super-1", Roles: []string{"super_admin"}}
	for _, role := range []string{"user", "admin", "super_admin"} {
		if !CanAssignRole(superAdmin, role) {
			t.Fatalf("superadmin should be able to assign %s", role)
		}
	}
	if CanAssignRole(superAdmin, "legal") {
		t.Fatal("legacy roles must not be assignable")
	}
}

func TestCanUsesEffectivePermissions(t *testing.T) {
	user := domain.User{Permissions: []string{ProposalCreate, ProposalPublish}}
	if !Can(user, ProposalPublish) {
		t.Fatal("effective permission should be recognized")
	}
	if Can(user, ProposalPriceEdit) {
		t.Fatal("missing permission must remain denied")
	}
}

func TestLastActiveSuperAdminCannotBeRemoved(t *testing.T) {
	if CanRemoveActiveSuperAdmin(1) {
		t.Fatal("the last active superadmin must remain protected")
	}
	if !CanRemoveActiveSuperAdmin(2) {
		t.Fatal("one superadmin may be removed when another active superadmin remains")
	}
}
