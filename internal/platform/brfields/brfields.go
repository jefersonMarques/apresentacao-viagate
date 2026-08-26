package brfields

import (
	"fmt"
	"strings"
	"unicode"
)

func Digits(value string) string {
	var builder strings.Builder
	for _, r := range value {
		if unicode.IsDigit(r) && r <= unicode.MaxASCII {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func NormalizePhone(value string, required bool) (string, error) {
	normalized := Digits(value)
	if normalized == "" {
		if required {
			return "", fmt.Errorf("phone is required")
		}
		return "", nil
	}

	if len(normalized) == 10 || len(normalized) == 11 {
		return normalized, nil
	}
	if strings.HasPrefix(normalized, "55") && (len(normalized) == 12 || len(normalized) == 13) {
		return normalized, nil
	}
	return "", fmt.Errorf("invalid Brazilian phone")
}

func NormalizePostalCode(value string, required bool) (string, error) {
	normalized := Digits(value)
	if normalized == "" {
		if required {
			return "", fmt.Errorf("postal code is required")
		}
		return "", nil
	}
	if len(normalized) != 8 {
		return "", fmt.Errorf("invalid Brazilian postal code")
	}
	return normalized, nil
}

func FormatCPF(value string) string {
	digits := Digits(value)
	if len(digits) != 11 {
		return value
	}
	return digits[:3] + "." + digits[3:6] + "." + digits[6:9] + "-" + digits[9:]
}

func FormatCNPJ(value string) string {
	normalized := normalizeCNPJForDisplay(value)
	if len(normalized) != 14 {
		return value
	}
	return normalized[:2] + "." + normalized[2:5] + "." + normalized[5:8] + "/" + normalized[8:12] + "-" + normalized[12:]
}

func FormatPhone(value string) string {
	digits := Digits(value)
	switch {
	case len(digits) == 10:
		return "(" + digits[:2] + ") " + digits[2:6] + "-" + digits[6:]
	case len(digits) == 11:
		return "(" + digits[:2] + ") " + digits[2:7] + "-" + digits[7:]
	case strings.HasPrefix(digits, "55") && len(digits) == 12:
		return "+55 (" + digits[2:4] + ") " + digits[4:8] + "-" + digits[8:]
	case strings.HasPrefix(digits, "55") && len(digits) == 13:
		return "+55 (" + digits[2:4] + ") " + digits[4:9] + "-" + digits[9:]
	default:
		return value
	}
}

func FormatPostalCode(value string) string {
	digits := Digits(value)
	if len(digits) != 8 {
		return value
	}
	return digits[:5] + "-" + digits[5:]
}

func normalizeCNPJForDisplay(value string) string {
	var builder strings.Builder
	builder.Grow(14)
	for _, r := range strings.ToUpper(strings.TrimSpace(value)) {
		switch {
		case r >= '0' && r <= '9', r >= 'A' && r <= 'Z':
			builder.WriteRune(r)
		case r == '.', r == '/', r == '-', unicode.IsSpace(r):
			continue
		default:
			return value
		}
	}
	return builder.String()
}
