package templates

import (
	"fmt"
	"strings"
	"time"

	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
)

func HasRole(user domain.User, role string) bool {
	for _, current := range user.Roles {
		if current == role { return true }
	}
	return false
}

func Join(values []string) string { return strings.Join(values, ", ") }

func Money(value float64) string {
	text := fmt.Sprintf("%.2f", value)
	parts := strings.Split(text, ".")
	integer := parts[0]
	for i := len(integer)-3; i>0; i-=3 {
		integer = integer[:i] + "." + integer[i:]
	}
	return "R$ " + integer + "," + parts[1]
}

func Date(value *time.Time) string {
	if value == nil { return "—" }
	return value.Format("02/01/2006")
}
