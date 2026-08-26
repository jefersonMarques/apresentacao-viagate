package brtaxid

import "testing"

func TestCleanCNPJSupportsLegacyAndAlphanumericFormats(t *testing.T){
	tests:=[]struct{input,want string}{
		{input:"11.222.333/0001-81",want:"11222333000181"},
		{input:"00.000.000/E08G-12",want:"00000000E08G12"},
		{input:"12.ABC.345/01DE-35",want:"12ABC34501DE35"},
	}
	for _,test:=range tests{
		got,err:=CleanCNPJ(test.input)
		if err!=nil{t.Fatalf("CleanCNPJ(%q): %v",test.input,err)}
		if got!=test.want{t.Fatalf("CleanCNPJ(%q)=%q, want %q",test.input,got,test.want)}
	}
}

func TestCleanCNPJRejectsInvalidValues(t *testing.T){
	values:=[]string{
		"00.000.000/0000-00",
		"11.111.111/1111-11",
		"00.000.000/E08G-13",
		"12.ABC.345/01DE-34",
		"00.000.000/E08G-AA",
		"00.000.000/E08@-12",
		"123",
	}
	for _,value:=range values{
		if _,err:=CleanCNPJ(value);err==nil{t.Fatalf("expected %q to be invalid",value)}
	}
}
