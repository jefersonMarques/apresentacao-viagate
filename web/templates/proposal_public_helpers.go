package templates

import (
	"strings"

	"github.com/jefersonMarques/apresentacao-viagate/internal/catalog"
	"github.com/jefersonMarques/apresentacao-viagate/internal/proposals"
)

type ProposalSolution struct {
	Title   string
	Summary string
	Status  string
}

type ProposalPriceGroup struct {
	Name        string
	Items       []proposals.Item
	AllOptional bool
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

func ProposalPriceGroups(proposal proposals.PublicProposal)[]ProposalPriceGroup{
	indexes:=map[string]int{}
	groups:=[]ProposalPriceGroup{}
	for _,item:=range proposal.Items{
		index,ok:=indexes[item.GroupName]
		if !ok{
			index=len(groups)
			indexes[item.GroupName]=index
			groups=append(groups,ProposalPriceGroup{Name:item.GroupName,AllOptional:true})
		}
		groups[index].Items=append(groups[index].Items,item)
		if !item.IsOptional{groups[index].AllOptional=false}
	}
	return groups
}

func ProposalModelCards(value string)[]catalog.PricingModel{
	result:=[]catalog.PricingModel{}
	for _,model:=range catalog.PricingModels{
		if value=="item_and_bundle"{
			if model.ID=="per_item"||model.ID=="bundle"{result=append(result,model)}
			continue
		}
		if model.ID==value{result=append(result,model)}
	}
	return result
}

func ProposalItemStatus(item proposals.Item) string {
	if item.IsOptional{return "Opcional"}
	return "Proposto"
}

func ProposalModelLabel(value string) string {
	for _,model:=range catalog.PricingModels{if model.ID==value{return model.Title}}
	return value
}

func ProposalSalesperson(proposal proposals.PublicProposal,key string) string {
	group,ok:=proposal.Content["salesperson"].(map[string]any);if !ok{return ""}
	value,_:=group[key].(string);return strings.TrimSpace(value)
}

func ProposalSalespersonRole(proposal proposals.PublicProposal)string{
	if value:=ProposalSalesperson(proposal,"role");value!=""{return value}
	return ProposalSalesperson(proposal,"job_title")
}

func ProposalContentString(proposal proposals.PublicProposal,key string)string{
	value,_:=proposal.Content[key].(string);return strings.TrimSpace(value)
}

func ProposalContentStrings(proposal proposals.PublicProposal,key string)[]string{
	values,ok:=proposal.Content[key].([]any);if !ok{return nil}
	result:=make([]string,0,len(values))
	for _,value:=range values{if text,ok:=value.(string);ok&&strings.TrimSpace(text)!=""{result=append(result,strings.TrimSpace(text))}}
	return result
}

func ProposalSectionString(proposal proposals.PublicProposal,section,key string)string{
	group,ok:=proposal.Content[section].(map[string]any);if !ok{return ""}
	value,_:=group[key].(string);return strings.TrimSpace(value)
}

func ProposalClientDisplayName(proposal proposals.PublicProposal)string{
	if strings.TrimSpace(proposal.ClientTradeName)!=""{return proposal.ClientTradeName}
	return proposal.ClientName
}
