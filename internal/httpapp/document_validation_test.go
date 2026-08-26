package httpapp

import "testing"

func TestCleanCPF(t *testing.T){
	valid,err:=cleanCPF("529.982.247-25")
	if err!=nil{t.Fatalf("expected valid CPF: %v",err)}
	if valid!="52998224725"{t.Fatalf("unexpected normalized CPF: %s",valid)}

	invalid:=[]string{"000.000.000-00","529.982.247-24","123"}
	for _,value:=range invalid{
		if _,err:=cleanCPF(value);err==nil{t.Fatalf("expected CPF %q to be rejected",value)}
	}
}

func TestCleanCNPJ(t *testing.T){
	valid,err:=cleanCNPJ("11.222.333/0001-81")
	if err!=nil{t.Fatalf("expected valid CNPJ: %v",err)}
	if valid!="11222333000181"{t.Fatalf("unexpected normalized CNPJ: %s",valid)}

	invalid:=[]string{"00.000.000/0000-00","11.222.333/0001-82","123"}
	for _,value:=range invalid{
		if _,err:=cleanCNPJ(value);err==nil{t.Fatalf("expected CNPJ %q to be rejected",value)}
	}
}
