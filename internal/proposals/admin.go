package proposals

import (
	"context"

	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
)

func (s *Store) List(ctx context.Context,userID string,all bool) ([]domain.Proposal,error) {
	query:=`
		select p.id::text,p.client_id::text,coalesce(nullif(c.trade_name,''),nullif(c.legal_name,''),'Cliente não identificado'),p.title,p.status::text,p.current_version,
		       coalesce(v.public_token::text,''),p.valid_until,p.created_by::text,u.name,p.updated_at
		from proposals p
		join clients c on c.id=p.client_id
		join users u on u.id=p.created_by
		left join proposal_versions v on v.proposal_id=p.id and v.version_number=p.current_version and v.published_at is not null
	`
	args:=[]any{}
	if !all { query += ` where p.created_by=$1`; args=append(args,userID) }
	query += ` order by p.updated_at desc`
	rows,err:=s.pool.Query(ctx,query,args...)
	if err!=nil{return nil,err}
	defer rows.Close()
	var items []domain.Proposal
	for rows.Next(){
		var item domain.Proposal
		if err:=rows.Scan(&item.ID,&item.ClientID,&item.ClientName,&item.Title,&item.Status,&item.CurrentVersion,&item.PublicToken,&item.ValidUntil,&item.CreatedBy,&item.CreatedByName,&item.UpdatedAt);err!=nil{return nil,err}
		items=append(items,item)
	}
	return items,rows.Err()
}
