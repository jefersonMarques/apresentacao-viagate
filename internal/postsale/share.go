package postsale

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Store) CreateSharedAccess(
	ctx context.Context,
	contractID string,
	userID string,
	name string,
	email string,
	delivery string,
	tokenHash []byte,
	expiresAt time.Time,
) error {
	if delivery != "copy" && delivery != "email" {
		return fmt.Errorf("invalid sharing action")
	}
	if userID == "" {
		return fmt.Errorf("missing staff user")
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var activationID string
	if err := tx.QueryRow(ctx,`
		select a.id::text
		from activation_profiles a
		join contracts c on c.id=a.contract_id
		where c.id=$1 and c.status='signed' and c.deleted_at is null
		  and a.deleted_at is null and a.status in ('pending','in_progress')
		for update of a
	`,contractID).Scan(&activationID); err != nil {
		return err
	}

	var recent int
	if err := tx.QueryRow(ctx,`
		select count(*) from post_sale_share_links
		where contract_id=$1 and created_at>now()-interval '1 hour'
	`,contractID).Scan(&recent); err != nil {
		return err
	}
	if recent>=10 {
		return fmt.Errorf("post-sale sharing rate limit reached")
	}

	_, err = tx.Exec(ctx,`
		update activation_access_tokens set revoked_at=now()
		where revoked_at is null and id in (
			select activation_access_token_id from post_sale_share_links where contract_id=$1
		)
	`,contractID)
	if err != nil {
		return err
	}

	var tokenID string
	if err := tx.QueryRow(ctx,`
		insert into activation_access_tokens(
			activation_id,token_hash,access_type,section,name,email,expires_at
		) values ($1,$2,'delegate','all',nullif($3,''),nullif($4,'')::citext,$5)
		returning id::text
	`,activationID,tokenHash,name,email,expiresAt).Scan(&tokenID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,`
		insert into post_sale_share_links(
			contract_id,activation_access_token_id,created_by_user_id,delivery
		) values($1,$2,$3,$4)
	`,contractID,tokenID,userID,delivery); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,`
		insert into audit_events(actor_user_id,actor_type,event_type,resource_type,resource_id,metadata)
		values($1,'user','post_sale.link_created','contract',$2,
		       jsonb_build_object('delivery',$3::text,'token_id',$4::uuid))
	`,userID,contractID,delivery,tokenID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
