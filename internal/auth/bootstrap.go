package auth

import (
	"context"
	"time"
)

func (s *Store) BootstrapAdmin(ctx context.Context,email,name string,tokenHash []byte,expiresAt time.Time)(bool,error){
	var exists bool
	if err:=s.pool.QueryRow(ctx,`select exists(select 1 from users)`).Scan(&exists);err!=nil{return false,err}
	if exists{return false,nil}
	tx,err:=s.pool.Begin(ctx);if err!=nil{return false,err};defer tx.Rollback(ctx)
	var userID string
	if err:=tx.QueryRow(ctx,`insert into users(email,name,status) values($1,$2,'invited') returning id::text`,email,name).Scan(&userID);err!=nil{return false,err}
	if _,err:=tx.Exec(ctx,`insert into user_roles(user_id,role_id) select $1,id from roles where code='super_admin'`,userID);err!=nil{return false,err}
	if _,err:=tx.Exec(ctx,`insert into user_invitations(user_id,token_hash,expires_at) values($1,$2,$3)`,userID,tokenHash,expiresAt);err!=nil{return false,err}
	if err:=tx.Commit(ctx);err!=nil{return false,err}
	return true,nil
}
