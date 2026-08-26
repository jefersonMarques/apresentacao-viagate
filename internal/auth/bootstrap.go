package auth

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Store) BootstrapAdmin(ctx context.Context,email,name string,tokenHash []byte,expiresAt time.Time)(bool,error){
	tx,err:=s.pool.BeginTx(ctx,pgx.TxOptions{IsoLevel:pgx.Serializable})
	if err!=nil{return false,err}
	defer tx.Rollback(ctx)

	var total int
	if err:=tx.QueryRow(ctx,`select count(*) from users`).Scan(&total);err!=nil{return false,err}
	if total==0{
		var userID string
		if err:=tx.QueryRow(ctx,`insert into users(email,name,status) values($1,$2,'invited') returning id::text`,email,name).Scan(&userID);err!=nil{return false,err}
		if _,err:=tx.Exec(ctx,`insert into user_roles(user_id,role_id) select $1,id from roles where code='super_admin'`,userID);err!=nil{return false,err}
		if _,err:=tx.Exec(ctx,`insert into user_invitations(user_id,token_hash,expires_at) values($1,$2,$3)`,userID,tokenHash,expiresAt);err!=nil{return false,err}
		if err:=tx.Commit(ctx);err!=nil{return false,err}
		return true,nil
	}

	if total!=1{return false,nil}
	var userID,status string
	err=tx.QueryRow(ctx,`select id::text,status::text from users where email=$1 for update`,email).Scan(&userID,&status)
	if err==pgx.ErrNoRows{return false,nil}
	if err!=nil{return false,err}
	if status!="invited"{return false,nil}

	var activeInvitation bool
	if err:=tx.QueryRow(ctx,`
		select exists(
			select 1 from user_invitations
			where user_id=$1 and accepted_at is null and expires_at>now()
		)
	`,userID).Scan(&activeInvitation);err!=nil{return false,err}
	if activeInvitation{return false,nil}

	if _,err:=tx.Exec(ctx,`update users set name=$2,updated_at=now() where id=$1`,userID,name);err!=nil{return false,err}
	if _,err:=tx.Exec(ctx,`
		insert into user_roles(user_id,role_id)
		select $1,id from roles where code='super_admin'
		on conflict do nothing
	`,userID);err!=nil{return false,err}
	if _,err:=tx.Exec(ctx,`insert into user_invitations(user_id,token_hash,expires_at) values($1,$2,$3)`,userID,tokenHash,expiresAt);err!=nil{return false,err}
	if err:=tx.Commit(ctx);err!=nil{return false,err}
	return true,nil
}
