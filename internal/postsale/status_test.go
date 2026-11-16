package postsale

import "testing"

func TestStatusLabel(t *testing.T) {
	cases := []struct{
		item Item
		want string
	}{
		{Item{},"Aguardando contato"},
		{Item{HasContact:true},"Em preparação"},
		{Item{ActivationStatus:"completed"},"Pronto para ativação"},
		{Item{ActivationStatus:"under_internal_setup"},"Em implantação"},
		{Item{ActivationStatus:"activated"},"Operação liberada"},
	}
	for _,test:=range cases{
		if got:=StatusLabel(test.item);got!=test.want{
			t.Errorf("status label=%q, want %q",got,test.want)
		}
	}
}
