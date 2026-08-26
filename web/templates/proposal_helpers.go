package templates

import (
	"strings"

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
