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
	tests:=[]struct{input,want string}{
		{input:"11.222.333/0001-81",want:"11222333000181"},
		{input:"00.000.000/E08G-12",want:"00000000E08G12"},
		{input:"00.000.000/e08g-12",want:"00000000E08G12"},
	}
	for _,test:=range tests{
		valid,err:=cleanCNPJ(test.input)
		if err!=nil{t.Fatalf("expected valid CNPJ %q: %v",test.input,err)}
		if valid!=test.want{t.Fatalf("CNPJ %q normalized to %q, want %q",test.input,valid,test.want)}
	}

	invalid:=[]string{"00.000.000/0000-00","11.222.333/0001-82","00.000.000/E08G-13","00.000.000/E08G-AA","123","00.000.000/E08@-12"}
	for _,value:=range invalid{
		if _,err:=cleanCNPJ(value);err==nil{t.Fatalf("expected CNPJ %q to be rejected",value)}
	}
}
