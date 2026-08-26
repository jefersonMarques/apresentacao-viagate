package templates

import (
	"strings"

	"github.com/jefersonMarques/apresentacao-viagate/internal/catalog"
	"github.com/jefersonMarques/apresentacao-viagate/internal/proposals"
)

type ProposalSolution struct {
	Title  string
	Summary string
	Status string
}

func ProposalSolutions(proposal proposals.PublicProposal) []ProposalSolution {
	seen:=map[string]*ProposalSolution{}
	order:=[]string{}
	for _,item:=range proposal.Items{
		key:=item.GroupName
		status:=ProposalItemStatus(item)
		if current,ok:=seen[key];ok{
			if current.Status!=status{current.Status="Incluído + opcional"}
			continue
		}
		title:=key;summary:="Solução selecionada para esta negociação."
		for _,group:=range catalog.Groups{if group.Title==key{title=group.ShortTitle;summary=group.Summary;break}}
		seen[key]=&ProposalSolution{Title:title,Summary:summary,Status:status};order=append(order,key)
	}
	result:=make([]ProposalSolution,0,len(order));for _,key:=range order{result=append(result,*seen[key])};return result
}

func ProposalItemStatus(item proposals.Item) string {
	if item.IsOptional{return "Opcional"}
	return "Incluído"
}

func ProposalModelLabel(value string) string {
	for _,model:=range catalog.PricingModels{if model.ID==value{return model.Title}}
	return value
}

func ProposalSalesperson(proposal proposals.PublicProposal,key string) string {
	group,ok:=proposal.Content["salesperson"].(map[string]any);if !ok{return ""}
	value,_:=group[key].(string);return strings.TrimSpace(value)
}
