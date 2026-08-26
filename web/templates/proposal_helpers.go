package templates

import (
	"strings"

	"github.com/jefersonMarques/apresentacao-viagate/internal/catalog"
	"github.com/jefersonMarques/apresentacao-viagate/internal/proposals"
)

func ProposalEditorItem(input proposals.EditorInput,catalogID string)(proposals.EditorItem,bool){
	for _,item:=range input.Items{if item.CatalogID==catalogID{return item,true}}
	return proposals.EditorItem{},false
}

func IsPricingModel(input proposals.EditorInput,model string)bool{
	if input.PricingModel==""{return model=="per_item"}
	return input.PricingModel==model
}

func EditorLines(values []string)string{return strings.Join(values,"\n")}

func CustomConditions(values []string)string{
	standard:=map[string]bool{}
	for _,condition:=range catalog.StandardConditions{standard[condition.Text]=true}
	custom:=make([]string,0,len(values))
	for _,value:=range values{if !standard[value]{custom=append(custom,value)}}
	return strings.Join(custom,"\n")
}
