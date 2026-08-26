package database

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MigrationResult struct {
	Applied []string
	Skipped []string
}

func MigrateUp(ctx context.Context,pool *pgxpool.Pool,directory string)(MigrationResult,error){
	entries,err:=os.ReadDir(directory)
	if err!=nil{return MigrationResult{},fmt.Errorf("read migrations directory: %w",err)}
	files:=make([]string,0,len(entries))
	for _,entry:=range entries{
		if entry.IsDir()||!strings.HasSuffix(strings.ToLower(entry.Name()),".sql"){continue}
		files=append(files,entry.Name())
	}
	sort.Strings(files)

	conn,err:=pool.Acquire(ctx)
	if err!=nil{return MigrationResult{},fmt.Errorf("acquire migration connection: %w",err)}
	defer conn.Release()

	if _,err:=conn.Exec(ctx,`select pg_advisory_lock(hashtext('viagate-commercial-migrations'))`);err!=nil{return MigrationResult{},fmt.Errorf("lock migrations: %w",err)}
	defer func(){_,_=conn.Exec(context.Background(),`select pg_advisory_unlock(hashtext('viagate-commercial-migrations'))`)}()

	if _,err:=conn.Exec(ctx,`
		create table if not exists schema_migrations (
			filename text primary key,
			sha256 bytea not null,
			applied_at timestamptz not null default now()
		)
	`);err!=nil{return MigrationResult{},fmt.Errorf("ensure schema_migrations: %w",err)}

	result:=MigrationResult{}
	for _,name:=range files{
		path:=filepath.Join(directory,name)
		content,err:=os.ReadFile(path)
		if err!=nil{return result,fmt.Errorf("read migration %s: %w",name,err)}
		hash:=sha256.Sum256(content)

		var storedHash []byte
		err=conn.QueryRow(ctx,`select sha256 from schema_migrations where filename=$1`,name).Scan(&storedHash)
		if err==nil{
			if string(storedHash)!=string(hash[:]){return result,fmt.Errorf("applied migration %s was modified",name)}
			result.Skipped=append(result.Skipped,name)
			continue
		}
		if !isNoRows(err){return result,fmt.Errorf("check migration %s: %w",name,err)}

		tx,err:=conn.Begin(ctx)
		if err!=nil{return result,fmt.Errorf("begin migration %s: %w",name,err)}
		if _,err:=tx.Exec(ctx,string(content));err!=nil{
			_ = tx.Rollback(ctx)
			return result,fmt.Errorf("apply migration %s: %w",name,err)
		}
		if _,err:=tx.Exec(ctx,`insert into schema_migrations(filename,sha256) values($1,$2)`,name,hash[:]);err!=nil{
			_ = tx.Rollback(ctx)
			return result,fmt.Errorf("record migration %s: %w",name,err)
		}
		if err:=tx.Commit(ctx);err!=nil{return result,fmt.Errorf("commit migration %s: %w",name,err)}
		result.Applied=append(result.Applied,name)
	}
	return result,nil
}
