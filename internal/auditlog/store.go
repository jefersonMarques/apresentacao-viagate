package auditlog

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

type Entry struct {
	ID           string
	Source       string
	EventType    string
	ResourceType string
	ResourceID   string
	Actor        string
	IPAddress    string
	OccurredAt   time.Time
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) List(ctx context.Context, query string, limit int) ([]Entry, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	query = strings.TrimSpace(query)
	rows, err := s.pool.Query(ctx, `
		with entries as (
			select 'audit:'||a.id::text as id,'audit'::text as source,a.event_type,a.resource_type,coalesce(a.resource_id::text,'') as resource_id,
			       coalesce(u.name,case a.actor_type when 'customer' then 'Cliente' when 'system' then 'Sistema ViaGate' else a.actor_type end) as actor,
			       coalesce(a.ip_address::text,'') as ip_address,a.created_at
			from audit_events a
			left join users u on u.id=a.actor_user_id
			union all
			select 'signature:'||s.id::text,'signature',s.event_type,'contract',s.contract_id::text,
			       coalesce(cs.name,'Cliente'),coalesce(s.ip_address::text,''),s.created_at
			from signature_events s
			left join contract_signers cs on cs.id=s.contract_signer_id
			union all
			select 'document:'||d.id::text,'document',d.event_type,d.document_kind,d.document_version_id::text,
			       'Visitante',coalesce(d.ip_address::text,''),d.created_at
			from document_events d
		)
		select id,source,event_type,resource_type,resource_id,actor,ip_address,created_at
		from entries
		where $1='' or event_type ilike '%'||$1||'%' or resource_type ilike '%'||$1||'%' or resource_id ilike '%'||$1||'%' or actor ilike '%'||$1||'%' or ip_address ilike '%'||$1||'%'
		order by created_at desc
		limit $2
	`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []Entry{}
	for rows.Next() {
		var entry Entry
		if err := rows.Scan(&entry.ID, &entry.Source, &entry.EventType, &entry.ResourceType, &entry.ResourceID, &entry.Actor, &entry.IPAddress, &entry.OccurredAt); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}
