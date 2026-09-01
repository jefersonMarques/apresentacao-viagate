package templates

import (
	"fmt"
	"strings"
	"time"

	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/brfields"
)

func HasRole(user domain.User, role string) bool {
	for _, current := range user.Roles {
		if current == role {
			return true
		}
	}
	return false
}

func Join(values []string) string { return strings.Join(values, ",") }

func UserInitials(name string) string {
	parts := strings.Fields(strings.TrimSpace(name))
	if len(parts) == 0 {
		return "VG"
	}
	first := []rune(parts[0])
	if len(parts) == 1 {
		if len(first) >= 2 {
			return strings.ToUpper(string(first[:2]))
		}
		return strings.ToUpper(string(first))
	}
	last := []rune(parts[len(parts)-1])
	return strings.ToUpper(string(first[0]) + string(last[0]))
}

func Money(value float64) string {
	text := fmt.Sprintf("%.2f", value)
	parts := strings.Split(text, ".")
	integer := parts[0]
	for i := len(integer) - 3; i > 0; i -= 3 {
		integer = integer[:i] + "." + integer[i:]
	}
	return "R$ " + integer + "," + parts[1]
}

func Date(value *time.Time) string {
	if value == nil {
		return "—"
	}
	return value.Format("02/01/2006")
}

func DateTime(value *time.Time) string {
	if value == nil {
		return "—"
	}
	return value.Format("02/01/2006 15:04")
}

func DateTimeSeconds(value *time.Time) string {
	if value == nil {
		return "—"
	}
	return value.Format("02/01/2006 15:04:05")
}

func CPF(value string) string        { return brfields.FormatCPF(value) }
func CNPJ(value string) string       { return brfields.FormatCNPJ(value) }
func Phone(value string) string      { return brfields.FormatPhone(value) }
func PostalCode(value string) string { return brfields.FormatPostalCode(value) }

func CustomerCPF(user domain.User, value string) string {
	if UserCan(user, "customer.sensitive_data.read") {
		return CPF(value)
	}
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	return "***.***.***-**"
}
