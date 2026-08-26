package httpapp

import (
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/security"
)

const customerSessionCookie = "viagate_customer_session"

func render(ctx context.Context,w http.ResponseWriter,status int,component templ.Component) {
	w.Header().Set("Content-Type","text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = component.Render(ctx,w)
}

func hashToken(value string) []byte { return security.HashToken(value) }

func setSecureCookie(w http.ResponseWriter,name,value string,expires time.Time,production bool) {
	http.SetCookie(w,&http.Cookie{
		Name:name,Value:value,Path:"/",Expires:expires,HttpOnly:true,Secure:production,SameSite:http.SameSiteStrictMode,
	})
}

func requestIP(r *http.Request) net.IP {
	host,_,err:=net.SplitHostPort(r.RemoteAddr)
	if err!=nil { host=r.RemoteAddr }
	return net.ParseIP(host)
}

func newUUID() (string,error) {
	value:=make([]byte,16)
	if _,err:=rand.Read(value);err!=nil{return "",err}
	value[6]=(value[6]&0x0f)|0x40
	value[8]=(value[8]&0x3f)|0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		value[0:4],value[4:6],value[6:8],value[8:10],value[10:16]),nil
}

func nullableUUID(value string) any {
	if strings.TrimSpace(value)==""{return nil}
	return value
}

var nonDigits=regexp.MustCompile(`\D`)

func digits(value string) string { return nonDigits.ReplaceAllString(value,"") }

func cleanCPF(value string) (string,error) {
	value=digits(value)
	if len(value)!=11 { return "",fmt.Errorf("CPF deve possuir 11 dígitos") }
	return value,nil
}

func cleanCNPJ(value string) (string,error) {
	value=digits(value)
	if len(value)!=14 { return "",fmt.Errorf("CNPJ deve possuir 14 dígitos") }
	return value,nil
}

func sanitizeFilename(value string) string {
	value=filepath.Base(value)
	value=strings.Map(func(r rune) rune {
		if r>='a'&&r<='z'||r>='A'&&r<='Z'||r>='0'&&r<='9'||r=='.'||r=='-'||r=='_' { return r }
		return '_'
	},value)
	if len(value)>120 { value=value[len(value)-120:] }
	return value
}

func formValuesAligned(names,phones,emails []string) [][3]string {
	max:=len(names);if len(phones)>max{max=len(phones)};if len(emails)>max{max=len(emails)}
	result:=make([][3]string,0,max)
	for i:=0;i<max;i++{
		var row [3]string
		if i<len(names){row[0]=strings.TrimSpace(names[i])}
		if i<len(phones){row[1]=strings.TrimSpace(phones[i])}
		if i<len(emails){row[2]=strings.TrimSpace(emails[i])}
		if row[0]!=""||row[1]!=""||row[2]!=""{result=append(result,row)}
	}
	return result
}
