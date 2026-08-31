package templates

import onboardingpkg "github.com/jefersonMarques/apresentacao-viagate/internal/onboarding"

func OnboardingAdminRowClass(item onboardingpkg.AdminItem) string {
	if item.ContractStatus == "signed" {
		return "pipeline-row-signed"
	}
	return ""
}
