package httpapp

import "testing"

func TestParseMoney(t *testing.T){
	tests:=[]struct{
		name string
		input string
		want float64
		wantErr bool
	}{
		{name:"empty",input:"",want:0},
		{name:"decimal point",input:"1234.56",want:1234.56},
		{name:"decimal comma",input:"1234,56",want:1234.56},
		{name:"brazilian thousands",input:"1.234,56",want:1234.56},
		{name:"negative",input:"-1",wantErr:true},
		{name:"invalid",input:"abc",wantErr:true},
		{name:"nan",input:"NaN",wantErr:true},
		{name:"infinity",input:"Inf",wantErr:true},
	}
	for _,test:=range tests{
		t.Run(test.name,func(t *testing.T){
			got,err:=parseMoney(test.input)
			if test.wantErr{
				if err==nil{t.Fatalf("expected error, got value %v",got)}
				return
			}
			if err!=nil{t.Fatalf("unexpected error: %v",err)}
			if got!=test.want{t.Fatalf("got %v, want %v",got,test.want)}
		})
	}
}

func TestNormalizedConditionsRemovesBlankAndDuplicateValues(t *testing.T){
	got:=normalizedConditions([]string{"Condição A","Condição A","  Condição B  "},"\nCondição B\nCondição C\n")
	want:=[]string{"Condição A","Condição B","Condição C"}
	if len(got)!=len(want){t.Fatalf("got %d conditions, want %d: %#v",len(got),len(want),got)}
	for index:=range want{
		if got[index]!=want[index]{t.Fatalf("condition %d = %q, want %q",index,got[index],want[index])}
	}
}
