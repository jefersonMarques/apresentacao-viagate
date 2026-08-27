package httpapp

import (
	"encoding/json"
	"net/http"

	"github.com/jefersonMarques/apresentacao-viagate/internal/legaltext"
	"github.com/jefersonMarques/apresentacao-viagate/internal/proposals"
)

func (a *App) writeProposalContractData(w http.ResponseWriter, r *http.Request, proposal proposals.PublicProposal) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")

	if lookupValue := r.URL.Query().Get("cnpj"); lookupValue != "" {
		cnpj, err := cleanCNPJ(lookupValue)
		if err != nil {
			http.Error(w, `{"error":"CNPJ inválido."}`, http.StatusBadRequest)
			return
		}
		company, err := a.registry.Lookup(r.Context(), cnpj)
		if err != nil {
			http.Error(w, `{"error":"CNPJ não encontrado."}`, http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"cnpj": company.CNPJ,
			"legal_name": company.LegalName,
			"trade_name": company.TradeName,
			"email": company.Email,
			"phone": company.Phone,
			"postal_code": company.PostalCode,
			"street": company.Street,
			"street_number": company.Number,
			"complement": company.Complement,
			"district": company.District,
			"city": company.City,
			"state": company.State,
		})
		return
	}

	templateVersionID := proposalSnapshotString(proposal.Content, "proposal", "contract_template_version_id")
	contractLabel := ""
	if templateVersionID != "" {
		var name string
		var version int
		if err := a.pool.QueryRow(r.Context(), `
			select t.name,v.version_number
			from contract_template_versions v
			join contract_templates t on t.id=v.contract_template_id
			where v.id=$1
		`, templateVersionID).Scan(&name, &version); err == nil {
			contractLabel = name
			if version > 0 {
				contractLabel += " · v" + fmtInt(version)
			}
		}
	}

	payload := map[string]any{
		"contract_assigned": templateVersionID != "",
		"contract_label": contractLabel,
		"acceptance_text": legaltext.ProposalAcceptanceText,
		"company": map[string]string{
			"cnpj": proposal.ClientCNPJ,
			"legal_name": proposalSnapshotString(proposal.Content, "client", "legal_name"),
			"trade_name": proposalSnapshotString(proposal.Content, "client", "trade_name"),
			"postal_code": proposalSnapshotString(proposal.Content, "client", "postal_code"),
			"street": proposalSnapshotString(proposal.Content, "client", "street"),
			"street_number": proposalSnapshotString(proposal.Content, "client", "street_number"),
			"complement": proposalSnapshotString(proposal.Content, "client", "complement"),
			"district": proposalSnapshotString(proposal.Content, "client", "district"),
			"city": proposalSnapshotString(proposal.Content, "client", "city"),
			"state": proposalSnapshotString(proposal.Content, "client", "state"),
		},
		"responsible": map[string]string{
			"name": proposalSnapshotString(proposal.Content, "contact", "name"),
			"role": proposalSnapshotString(proposal.Content, "contact", "role"),
			"email": proposalSnapshotString(proposal.Content, "contact", "email"),
			"phone": proposalSnapshotString(proposal.Content, "contact", "phone"),
		},
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func fmtInt(value int) string {
	if value == 0 {
		return "0"
	}
	buf := [20]byte{}
	index := len(buf)
	for value > 0 {
		index--
		buf[index] = byte('0' + value%10)
		value /= 10
	}
	return string(buf[index:])
}
