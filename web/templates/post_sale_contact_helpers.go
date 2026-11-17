package templates

func TeamContactValue(mode, value string) string {
	if mode == "team" {
		return value
	}
	return ""
}
