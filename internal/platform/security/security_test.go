package security

import (
	"bytes"
	"testing"
)

func TestHashPasswordAndVerify(t *testing.T){
	hash,err:=HashPassword("uma-senha-segura-123")
	if err!=nil{t.Fatalf("HashPassword returned error: %v",err)}
	if !VerifyPassword(hash,"uma-senha-segura-123"){t.Fatal("expected password to verify")}
	if VerifyPassword(hash,"senha-incorreta"){t.Fatal("wrong password must not verify")}
}

func TestHashPasswordRejectsShortPassword(t *testing.T){
	if _,err:=HashPassword("curta");err==nil{t.Fatal("expected short password to be rejected")}
}

func TestRandomTokenReturnsMatchingHash(t *testing.T){
	plain,hash,err:=RandomToken(32)
	if err!=nil{t.Fatalf("RandomToken returned error: %v",err)}
	if plain==""{t.Fatal("expected a non-empty token")}
	if !bytes.Equal(hash,HashToken(plain)){t.Fatal("token hash does not match HashToken")}
}
