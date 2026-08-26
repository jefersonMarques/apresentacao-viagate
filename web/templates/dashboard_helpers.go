package templates

import (
	"sort"
	"time"

	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
	"github.com/jefersonMarques/apresentacao-viagate/internal/presentations"
)

type DashboardRecord struct {
	ID           string
	Kind         string
	KindLabel    string
	ClientName   string
	Title        string
	OwnerName    string
	Status       string
	Version      int
	ValidUntil   *time.Time
	UpdatedAt    time.Time
	EditURL      string
	PublicURL    string
}

func DashboardRecords(proposalItems []domain.Proposal,presentationItems []presentations.Presentation) []DashboardRecord {
	items:=make([]DashboardRecord,0,len(proposalItems)+len(presentationItems))
	for _,item:=range proposalItems{
		publicURL:=""
		if item.PublicToken!=""{publicURL="/p/"+item.PublicToken}
		items=append(items,DashboardRecord{
			ID:item.ID,Kind:"proposal",KindLabel:"Proposta",ClientName:item.ClientName,Title:item.Title,
			OwnerName:item.CreatedByName,Status:item.Status,Version:item.CurrentVersion,ValidUntil:item.ValidUntil,
			UpdatedAt:item.UpdatedAt,EditURL:"/admin/proposals/"+item.ID+"/edit",PublicURL:publicURL,
		})
	}
	for _,item:=range presentationItems{
		publicURL:=""
		if item.PublicToken!=""{publicURL="/a/"+item.PublicToken}
		items=append(items,DashboardRecord{
			ID:item.ID,Kind:"presentation",KindLabel:"Apresentação",ClientName:item.ClientName,Title:item.Title,
			OwnerName:item.CreatedByName,Status:item.Status,Version:item.CurrentVersion,UpdatedAt:item.UpdatedAt,
			EditURL:"/admin/presentations/"+item.ID+"/edit",PublicURL:publicURL,
		})
	}
	sort.SliceStable(items,func(i,j int)bool{return items[i].UpdatedAt.After(items[j].UpdatedAt)})
	return items
}

func DashboardCountByKind(items []DashboardRecord,kind string) int {
	count:=0
	for _,item:=range items{if item.Kind==kind{count++}}
	return count
}

func DashboardActiveCount(items []DashboardRecord) int {
	count:=0
	for _,item:=range items{if item.Status=="published"||item.Status=="accepted"{count++}}
	return count
}
