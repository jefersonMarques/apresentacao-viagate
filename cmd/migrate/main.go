package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/database"
)

func main(){
	command:="up"
	if len(os.Args)>1{command=os.Args[1]}
	if command!="up"{
		fmt.Fprintln(os.Stderr,"uso: go run ./cmd/migrate up")
		os.Exit(2)
	}

	databaseURL:=os.Getenv("DATABASE_URL")
	if databaseURL==""{
		fmt.Fprintln(os.Stderr,"DATABASE_URL é obrigatório")
		os.Exit(1)
	}
	directory:=os.Getenv("MIGRATIONS_DIR")
	if directory==""{directory="migrations"}

	ctx,cancel:=context.WithTimeout(context.Background(),2*time.Minute)
	defer cancel()
	pool,err:=database.Open(ctx,databaseURL)
	if err!=nil{
		fmt.Fprintln(os.Stderr,err)
		os.Exit(1)
	}
	defer pool.Close()

	result,err:=database.MigrateUp(ctx,pool,directory)
	if err!=nil{
		fmt.Fprintln(os.Stderr,err)
		os.Exit(1)
	}
	for _,name:=range result.Applied{fmt.Println("applied",name)}
	for _,name:=range result.Skipped{fmt.Println("skipped",name)}
	fmt.Printf("migrations concluídas: %d aplicadas, %d já existentes\n",len(result.Applied),len(result.Skipped))
}
