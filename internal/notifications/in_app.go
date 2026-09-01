package notifications

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
)

type EventDefinition struct {
	Code  string
	Label string
	Group string
}

var EventDefinitions = []EventDefinition{
	{Code: "presentation.opened", Label: "Cliente abriu uma apresentação", Group: "Apresentações"},
	{Code: "proposal.opened", Label: "Cliente abriu uma proposta", Group: "Propostas"},
	{Code: "proposal.accepted", Label: "Cliente aceitou uma proposta", Group: "Propostas"},
	{Code: "onboarding.submitted", Label: "Cliente enviou os dados de contratação", Group: "Cadastros"},
	{Code: "contract.opened", Label: "Cliente abriu um contrato", Group: "Contratos"},
	{Code: "contract.signed", Label: "Contrato foi assinado", Group: "Contratos"},
	{Code: "activation.submitted", Label: "Cliente enviou os dados de ativação", Group: "Ativação"},
	{Code: "activation.activated", Label: "Operação do cliente foi liberada", Group: "Ativação"},
}

func IsSupportedEvent(code string) bool {
	for _, event := range EventDefinitions {
		if event.Code == code {
			return true
		}
	}
	return false
}

type InAppStore struct {
	pool *pgxpool.Pool
}

func NewInAppStore(pool *pgxpool.Pool) *InAppStore { return &InAppStore{pool: pool} }

type Event struct {
	OwnerUserID  string
	ActorUserID  string
	EventType    string
	Title        string
	Body         string
	ResourceType string
	ResourceID   string
	TargetURL    string
	DedupeKey    string
}

func (s *InAppStore) Publish(ctx context.Context, event Event) error {
	if event.OwnerUserID == "" || !IsSupportedEvent(event.EventType) {
		return nil
	}
	if event.DedupeKey == "" {
		event.DedupeKey = event.EventType + ":" + event.ResourceID
	}
	_, err := s.pool.Exec(ctx, `
		with candidates as (
			select u.id,
			       case when u.id=$1::uuid then 'own' else 'all' end as scope
			from users u
			where u.status='active'
			  and exists(select 1 from effective_user_permissions(u.id) where permission_code='notification.read')
			  and (
				u.id=$1::uuid
				or exists(select 1 from effective_user_permissions(u.id) where permission_code='notification.receive_others')
			  )
		), eligible as (
			select c.id,c.scope
			from candidates c
			where coalesce(
				(select pref.enabled from user_notification_preferences pref
				 where pref.user_id=c.id and pref.event_type=$3 and pref.scope=c.scope),
				c.scope='own'
			)
		)
		insert into in_app_notifications(
			recipient_user_id,owner_user_id,actor_user_id,event_type,title,body,
			resource_type,resource_id,target_url,dedupe_key
		)
		select e.id,$1::uuid,nullif($2,'')::uuid,$3,$4,nullif($5,''),nullif($6,''),nullif($7,'')::uuid,nullif($8,''),$9
		from eligible e
		on conflict (recipient_user_id,dedupe_key) where dedupe_key is not null do nothing
	`, event.OwnerUserID, event.ActorUserID, event.EventType, event.Title, event.Body, event.ResourceType, event.ResourceID, event.TargetURL, event.DedupeKey)
	if err != nil {
		return fmt.Errorf("publish in-app notification: %w", err)
	}
	return nil
}

func (s *InAppStore) PublishToPermission(ctx context.Context, permission, excludedUserID string, event Event) error {
	if event.OwnerUserID == "" || permission == "" || !IsSupportedEvent(event.EventType) {
		return nil
	}
	if event.DedupeKey == "" {
		event.DedupeKey = event.EventType + ":" + event.ResourceID
	}
	_, err := s.pool.Exec(ctx, `
		with eligible as (
			select u.id
			from users u
			where u.status='active'
			  and ($10='' or u.id<>$10::uuid)
			  and exists(select 1 from effective_user_permissions(u.id) where permission_code='notification.read')
			  and exists(select 1 from effective_user_permissions(u.id) where permission_code=$11)
			  and coalesce(
				(select pref.enabled from user_notification_preferences pref
				 where pref.user_id=u.id and pref.event_type=$3 and pref.scope='all'),
				true
			  )
		)
		insert into in_app_notifications(
			recipient_user_id,owner_user_id,actor_user_id,event_type,title,body,
			resource_type,resource_id,target_url,dedupe_key
		)
		select e.id,$1::uuid,nullif($2,'')::uuid,$3,$4,nullif($5,''),nullif($6,''),nullif($7,'')::uuid,nullif($8,''),$9
		from eligible e
		on conflict (recipient_user_id,dedupe_key) where dedupe_key is not null do nothing
	`, event.OwnerUserID, event.ActorUserID, event.EventType, event.Title, event.Body, event.ResourceType, event.ResourceID, event.TargetURL, event.DedupeKey, excludedUserID, permission)
	if err != nil {
		return fmt.Errorf("publish permission notification: %w", err)
	}
	return nil
}

func (s *InAppStore) List(ctx context.Context, userID string, limit int) ([]domain.InAppNotification, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		select id::text,event_type,title,coalesce(body,''),coalesce(resource_type,''),
		       coalesce(resource_id::text,''),
		       case when event_type='presentation.opened' then '/admin/pipeline' else coalesce(target_url,'') end,
		       read_at,created_at
		from in_app_notifications
		where recipient_user_id=$1
		order by created_at desc
		limit $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.InAppNotification{}
	for rows.Next() {
		var item domain.InAppNotification
		if err := rows.Scan(&item.ID, &item.EventType, &item.Title, &item.Body, &item.ResourceType, &item.ResourceID, &item.TargetURL, &item.ReadAt, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *InAppStore) MarkRead(ctx context.Context, userID, notificationID string) error {
	_, err := s.pool.Exec(ctx, `
		update in_app_notifications set read_at=coalesce(read_at,now())
		where id=$1 and recipient_user_id=$2
	`, notificationID, userID)
	return err
}

func (s *InAppStore) MarkAllRead(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(ctx, `
		update in_app_notifications set read_at=now()
		where recipient_user_id=$1 and read_at is null
	`, userID)
	return err
}
