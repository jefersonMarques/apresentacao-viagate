package templates

import "github.com/jefersonMarques/apresentacao-viagate/internal/domain"

func PrimaryRole(user domain.User) string {
	if len(user.Roles)==0{return "commercial"}
	return user.Roles[0]
}
