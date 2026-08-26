package brfields

import "testing"

func TestNormalizePhone(t *testing.T){
	tests:=[]struct{input,want string}{
		{input:"(11) 3333-4444",want:"1133334444"},
		{input:"(11) 99999-8888",want:"11999998888"},
		{input:"+55 (11) 99999-8888",want:"5511999998888"},
	}
	for _,test:=range tests{
		got,err:=NormalizePhone(test.input,true)
		if err!=nil{t.Fatalf("NormalizePhone(%q): %v",test.input,err)}
		if got!=test.want{t.Fatalf("NormalizePhone(%q)=%q, want %q",test.input,got,test.want)}
	}
	if _,err:=NormalizePhone("123",true);err==nil{t.Fatal("expected short phone to be rejected")}
}

func TestNormalizePostalCode(t *testing.T){
	got,err:=NormalizePostalCode("01310-100",true)
	if err!=nil{t.Fatalf("NormalizePostalCode: %v",err)}
	if got!="01310100"{t.Fatalf("got %q",got)}
	if _,err:=NormalizePostalCode("0131",true);err==nil{t.Fatal("expected invalid CEP to be rejected")}
}

func TestDisplayFormatting(t *testing.T){
	if got:=FormatCPF("52998224725");got!="529.982.247-25"{t.Fatalf("CPF=%q",got)}
	if got:=FormatCNPJ("11222333000181");got!="11.222.333/0001-81"{t.Fatalf("CNPJ=%q",got)}
	if got:=FormatCNPJ("00000000E08G12");got!="00.000.000/E08G-12"{t.Fatalf("alphanumeric CNPJ=%q",got)}
	if got:=FormatPhone("11999998888");got!="(11) 99999-8888"{t.Fatalf("phone=%q",got)}
	if got:=FormatPostalCode("01310100");got!="01310-100"{t.Fatalf("CEP=%q",got)}
}
