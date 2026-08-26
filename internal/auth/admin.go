package auth

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (s *Store) UpdateAccess(ctx context.Context,targetUserID,status,roleCode string) error {
	if status!="active"&&status!="disabled"&&status!="invited"{
		return fmt.Errorf("invalid user status")
	}
	tx,err:=s.pool.BeginTx(ctx,pgx.TxOptions{IsoLevel:pgx.Serializable})
	if err!=nil{return err}
	defer tx.Rollback(ctx)

	var currentStatus string
	var isSuperAdmin bool
	if err:=tx.QueryRow(ctx,`
		select u.status::text,exists(
			select 1 from user_roles ur join roles r on r.id=ur.role_id
			where ur.user_id=u.id and r.code='super_admin'
		)
		from users u where u.id=$1 for update
	`,targetUserID).Scan(&currentStatus,&isSuperAdmin);err!=nil{return err}
	_ = currentStatus

	var roleExists bool
	if err:=tx.QueryRow(ctx,`select exists(select 1 from roles where code=$1)`,roleCode).Scan(&roleExists);err!=nil{return err}
	if !roleExists{return fmt.Errorf("invalid role")}

	removingLastSuperAdmin:=isSuperAdmin&&(status!="active"||roleCode!="super_admin")
	if removingLastSuperAdmin{
		var activeSuperAdmins int
		if err:=tx.QueryRow(ctx,`
			select count(distinct u.id)
			from users u
			join user_roles ur on ur.user_id=u.id
			join roles r on r.id=ur.role_id
			where u.status='active' and r.code='super_admin'
		`).Scan(&activeSuperAdmins);err!=nil{return err}
		if activeSuperAdmins<=1{return fmt.Errorf("cannot remove the last active super admin")}
	}

	if _,err:=tx.Exec(ctx,`update users set status=$2,updated_at=now() where id=$1`,targetUserID,status);err!=nil{return err}
	if _,err:=tx.Exec(ctx,`delete from user_roles where user_id=$1`,targetUserID);err!=nil{return err}
	if _,err:=tx.Exec(ctx,`
		insert into user_roles(user_id,role_id)
		select $1,id from roles where code=$2
	`,targetUserID,roleCode);err!=nil{return err}
	if status=="disabled"{
		if _,err:=tx.Exec(ctx,`update sessions set revoked_at=coalesce(revoked_at,now()) where user_id=$1 and revoked_at is null`,targetUserID);err!=nil{return err}
	}
	return tx.Commit(ctx)
}

func (s *Store) UserByID(ctx context.Context,id string)(string,string,string,error){
	var name,email,status string
	err:=s.pool.QueryRow(ctx,`select name,email::text,status::text from users where id=$1`,id).Scan(&name,&email,&status)
	return name,email,status,err
}
