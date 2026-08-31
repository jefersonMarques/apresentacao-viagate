package templates

import "github.com/jefersonMarques/apresentacao-viagate/internal/contracts"

func ContractAdminRowClass(item contracts.AdminContractItem) string {
	if item.Status == "signed" {
		return "pipeline-row-signed"
	}
	return ""
}
