package httpapp

import (
	"context"
	"net/http"
	"time"
)

func (a *App) ready(w http.ResponseWriter,r *http.Request){
	ctx,cancel:=context.WithTimeout(r.Context(),3*time.Second)
	defer cancel()
	if err:=a.pool.Ping(ctx);err!=nil{
		a.logger.Error("readiness postgres failed","error",err)
		http.Error(w,"postgres unavailable",http.StatusServiceUnavailable)
		return
	}
	if err:=a.storage.Check(ctx);err!=nil{
		a.logger.Error("readiness S3 failed","error",err)
		http.Error(w,"S3 unavailable",http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
