package presentations

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct { pool *pgxpool.Pool }

type Presentation struct {
	ID string
	ClientName string
	Title string
	Status string
	CurrentVersion int
	CreatedBy string
	UpdatedAt time.Time
}

func NewStore(pool *pgxpool.Pool)*Store{return &Store{pool:pool}}

func (s *Store) List(ctx context.Context,userID string,all bool)([]Presentation,error){
	query:=`
		select p.id::text,coalesce(c.legal_name,''),p.title,p.status::text,p.current_version,p.created_by::text,p.updated_at
		from presentations p left join clients c on c.id=p.client_id
	`
	args:=[]any{}
	if !all {query+=` where p.created_by=$1`;args=append(args,userID)}
	query+=` order by p.updated_at desc`
	rows,err:=s.pool.Query(ctx,query,args...);if err!=nil{return nil,err};defer rows.Close()
	var items []Presentation
	for rows.Next(){var item Presentation;if err:=rows.Scan(&item.ID,&item.ClientName,&item.Title,&item.Status,&item.CurrentVersion,&item.CreatedBy,&item.UpdatedAt);err!=nil{return nil,err};items=append(items,item)}
	return items,rows.Err()
}
