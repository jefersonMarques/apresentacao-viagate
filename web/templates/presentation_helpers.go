package templates

import (
	"strings"

	"github.com/jefersonMarques/apresentacao-viagate/internal/catalog"
	"github.com/jefersonMarques/apresentacao-viagate/internal/presentations"
)

func PresentationModuleSelected(input presentations.EditorInput,id string) bool {
	for _,value:=range input.SelectedModules { if value==id{return true} }
	return false
}

func PresentationGroups(selected []string) []catalog.Group {
	allowed:=map[string]bool{}
	for _,id:=range selected{allowed[id]=true}
	result:=make([]catalog.Group,0,len(selected))
	for _,group:=range catalog.Groups{if allowed[group.ID]{result=append(result,group)}}
	return result
}

func PresentationClientName(value presentations.PublicPresentation) string {
	if strings.TrimSpace(value.ClientTradeName)!=""{return value.ClientTradeName}
	if strings.TrimSpace(value.ClientLegalName)!=""{return value.ClientLegalName}
	return "sua operação"
}

func BoolString(value bool) string {
	if value{return "true"}
	return "false"
}
